import CBOR from 'cbor-js'
import * as R from 'ramda'
import type {
  BundleRef,
  AssetResult,
  AssetTuple,
  IndexResult,
  DataResponse,
  ResponseStatus,
  Model,
  Request,
  Asset,
  StateResponse,
  AssetData,
  AssetIndex,
  LoadState,
  AssetSource,
  AssetState,
  Bundle,
  BundleData,
  GameMap,
  GameMod,
  IndexAsset,
  MountData,
} from './types'
import {
  ResponseType,
  RequestType,
  LoadStateType,
  DataType,
  LoadRequestType,
  load,
  result,
} from './types'
import type { DownloadState } from '../types'

import { getBlob as getSavedBlob, saveBlob, haveBlob } from './storage'

class PullError extends Error {}

let indexFetch: Maybe<Promise<AssetIndex>> = null

let assetSources: string[] = []
let assetIndex: Maybe<AssetIndex> = null

function cleanPath(source: string): string {
  const lastSlash = source.lastIndexOf('/')
  if (lastSlash === -1) {
    return ''
  }

  return source.slice(0, lastSlash + 1)
}

const INT_SIZE = 4
function unpackBundle(data: ArrayBuffer): BundleData {
  const view = new DataView(data)

  let offset = 0

  const pathLength = view.getInt32(offset)
  offset += INT_SIZE
  const paths = JSON.parse(
    new TextDecoder().decode(new Uint8Array(data, offset, pathLength))
  )
  offset += pathLength

  const metadataLength = view.getInt32(offset)
  offset += INT_SIZE
  const metadata = JSON.parse(
    new TextDecoder().decode(new Uint8Array(data, offset, metadataLength))
  )
  offset += metadataLength

  return {
    dataOffset: offset,
    buffer: data,
    size: metadata.remote_package_size,
    directories: paths,
    files: metadata.files,
  }
}

async function fetchData(
  url: string,
  progress: (bundle: LoadState) => void
): Promise<ArrayBuffer> {
  const request = new XMLHttpRequest()
  request.open('GET', url, true)
  request.responseType = 'arraybuffer'
  request.onprogress = (event) => {
    if (!event.lengthComputable) {
      return
    }
    progress(
      load.downloading({
        downloadedBytes: event.loaded,
        totalBytes: event.total,
      })
    )
  }
  request.onerror = function (event) {
    throw new PullError('NetworkError for: ' + url)
  }

  return new Promise((resolve, reject) => {
    request.onload = function (event) {
      if (
        request.status == 200 ||
        request.status == 304 ||
        request.status == 206 ||
        (request.status == 0 && request.response)
      ) {
        resolve(request.response)
      } else {
        throw new PullError(request.statusText + ' : ' + request.responseURL)
      }
    }
    request.send(null)
  })
}

async function fetchSourceData(
  source: string,
  id: string,
  progress: (bundle: LoadState) => void
): Promise<ArrayBuffer> {
  return fetchData(`${cleanPath(source)}${id}`, progress)
}

async function loadBlob(
  source: string,
  id: string,
  url: string,
  progress: (bundle: LoadState) => void
): Promise<ArrayBuffer> {
  if (await haveBlob(id)) {
    const buffer = await getSavedBlob(id)
    if (buffer == null) {
      throw new PullError(`Asset ${id} did not exist`)
    }
    return buffer
  }

  progress(
    load.downloading({
      downloadedBytes: 0,
      totalBytes: 0,
    })
  )
  const buffer = await fetchSourceData(source, url, progress)
  await saveBlob(id, buffer)
  return buffer
}

function sourceFromBuffer(buffer: ArrayBuffer): AssetSource {
  return CBOR.decode(buffer)
}

async function loadIndex(
  url: string,
  progress: (bundle: LoadState) => void
): Promise<AssetSource> {
  const shouldCache = !url.startsWith('!')
  const cleanUrl = shouldCache ? url : url.slice(1)

  if (await haveBlob(cleanUrl)) {
    const buffer = await getSavedBlob(cleanUrl)
    if (buffer == null) {
      throw new PullError(`Index ${cleanUrl} did not exist`)
    }
    const source = sourceFromBuffer(buffer)
    source.source = cleanUrl
    return source
  }

  progress(
    load.downloading({
      downloadedBytes: 0,
      totalBytes: 0,
    })
  )
  const buffer = await fetchData(cleanUrl, progress)
  if (shouldCache) {
    await saveBlob(cleanUrl, buffer)
  }
  const source = sourceFromBuffer(buffer)
  source.source = cleanUrl
  return source
}

async function loadAsset(
  asset: Asset,
  progress: (bundle: LoadState) => void
): Promise<MountData> {
  const { id, path } = asset
  if (assetIndex == null) {
    throw new PullError('missing asset index')
  }
  const sourceIndex = assetIndex.assetLookup[id]
  if (sourceIndex == null) {
    throw new PullError(`asset not found: ${id}`)
  }
  const { source } = assetIndex.sources[sourceIndex]
  const buffer = await loadBlob(source, id, id, progress)
  progress(load.ok(buffer.byteLength))

  return {
    files: [
      {
        path,
        data: new Uint8Array(buffer),
      },
    ],
    buffers: [buffer],
  }
}

async function loadBundle(
  id: string,
  progress: (bundle: LoadState) => void
): Promise<MountData> {
  if (assetIndex == null) {
    throw new PullError('missing asset index')
  }
  const bundle = assetIndex.bundleLookup[id]
  if (bundle == null) {
    throw new PullError(`bundle not found: ${id}`)
  }
  const { source } = assetIndex.sources[bundle[0]]
  const buffer = await loadBlob(source, id, `${id}.sour`, progress)
  const data = unpackBundle(buffer)
  progress(load.ok(buffer.byteLength))

  return {
    files: R.map(
      ({ filename, start, end }): AssetData => ({
        path: filename,
        data: new Uint8Array(buffer, data.dataOffset + start, end - start),
      }),
      data.files
    ),
    buffers: [buffer],
  }
}

const resolveBundles = (source: AssetSource, bundles: string[]): Bundle[] =>
  R.chain((id) => {
    const bundle = R.find((v) => v.id === id, source.bundles)
    if (bundle == null) return []
    return [bundle]
  }, bundles)

const resolveAssets = (source: AssetSource, assets: AssetTuple[]): Asset[] =>
  R.chain(([id, path]) => {
    return [
      {
        id,
        path,
      },
    ]
  }, assets)

type LookupResult = {
  assets: AssetTuple[]
  bundles: string[]
}

type ResolvedLookup = {
  assets: Asset[]
  bundles: Bundle[]
}

const makeResolver =
  <T>(
    list: (source: AssetSource) => T[],
    matches: (target: string, item: T, source: AssetSource) => boolean,
    transform: (item: T) => LookupResult
  ) =>
  (source: AssetSource, target: string): Maybe<ResolvedLookup> => {
    const found = R.find((v) => matches(target, v, source), list(source))
    if (found == null) return null
    const { assets: foundAssets, bundles: foundBundles } = transform(found)

    // If anything inside of this result fails to resolve, it's game over,
    // the assets are missing.
    const assets = resolveAssets(source, foundAssets)
    if (assets.length !== foundAssets.length) return null
    const bundles = resolveBundles(source, foundBundles)
    if (bundles.length !== foundBundles.length) return null

    return { assets, bundles }
  }

type Resolver = (source: AssetSource, target: string) => Maybe<ResolvedLookup>

const resolvers: Record<LoadRequestType, Resolver> = {
  [LoadRequestType.Map]: makeResolver<GameMap>(
    ({ maps }) => maps,
    (target, map) => map.name === target || map.id.startsWith(target),
    ({ bundle, assets }) => ({ bundles: [bundle], assets: [] })
  ),
  [LoadRequestType.Model]: makeResolver<Model>(
    ({ models }) => models,
    (target, model) => model.name === target || model.id.startsWith(target),
    ({ id }) => ({ bundles: [id], assets: [] })
  ),
  [LoadRequestType.Texture]: makeResolver<AssetTuple>(
    ({ textures }) => textures,
    (target, [id, path]) => path === target || id.startsWith(target),
    (texture) => ({ bundles: [], assets: [texture] })
  ),
  [LoadRequestType.Mod]: makeResolver<GameMod>(
    ({ mods }) => mods,
    (target, mod) => mod.name === target || mod.id.startsWith(target),
    (mod) => ({ bundles: [mod.id], assets: [] })
  ),
}

const IMAGE_REGEX = /packages\/textures\/images\/(\w+(.png|.jpg))/

// We can only pull a bundle if it's build for the web; otherwise we need to
// break it into its assets.
function crackBundles(resolved: ResolvedLookup): ResolvedLookup {
  const { assets, bundles } = resolved
  if (bundles.length === 0) return resolved

  return {
    assets: [
      ...assets,
      ...R.chain(
        (bundle: Bundle): Asset[] =>
          !bundle.web
            ? R.map(([id, path]) => ({ id, path }), bundle.assets)
            : [],
        bundles
      ),
    ],
    bundles: R.chain((bundle) => (bundle.web ? [bundle] : []), bundles),
  }
}

function resolveRequest(
  type: LoadRequestType,
  target: string
): Maybe<ResolvedLookup> {
  if (assetIndex == null) return null

  const resolver = resolvers[type]
  if (resolver == null) return null
  for (const source of assetIndex.sources) {
    const resolved = resolver(source, target)
    if (resolved == null) continue
    return resolved
  }

  // Textures have special handling because they can refer to images, too
  if (type !== LoadRequestType.Texture) return null

  const image = IMAGE_REGEX.exec(target)
  if (image != null) {
    const [, id] = image

    // Find the source this image points to
    const source = R.find((v: AssetSource): boolean => {
      return R.any(
        (u: string): boolean => id === u,
        [
          ...R.chain(
            ({ image }): string[] => (image != null ? [image] : []),
            v.maps
          ),
          ...R.chain(
            ({ image }): string[] => (image != null ? [image] : []),
            v.mods
          ),
        ]
      )
    }, assetIndex.sources)

    if (source == null) return null

    return {
      bundles: [],
      assets: [
        {
          id,
          path: target,
        },
      ],
    }
  }

  return null
}

const haveType =
  (type: LoadStateType) =>
  (states: AssetState[]): boolean =>
    R.any(({ state }) => state.type === type, states)

const haveWaiting = haveType(LoadStateType.Waiting)
const haveMissing = haveType(LoadStateType.Missing)
const haveFailed = haveType(LoadStateType.Failed)

const aggregateState = (states: AssetState[]): LoadState => {
  if (states.length === 0) {
    return load.waiting()
  }

  // If we have any missing or errors, it's done.
  if (haveMissing(states)) {
    return load.missing()
  }

  if (haveFailed(states)) {
    return load.failed()
  }

  // Now all are either downloading or OK (we have no waiting, missing, or
  // failed)
  const downloadState: DownloadState = R.reduce(
    (a: DownloadState, { state }: AssetState): DownloadState => {
      const individual: DownloadState =
        state.type === LoadStateType.Downloading
          ? {
              downloadedBytes: state.downloadedBytes,
              totalBytes: state.totalBytes,
            }
          : state.type === LoadStateType.Ok
          ? {
              downloadedBytes: state.totalBytes,
              totalBytes: state.totalBytes,
            }
          : {
              downloadedBytes: 0,
              totalBytes: 0,
            }

      return {
        downloadedBytes: a.downloadedBytes + individual.downloadedBytes,
        totalBytes: a.totalBytes + individual.totalBytes,
      }
    },
    {
      downloadedBytes: 0,
      totalBytes: 0,
    },
    states
  )

  if (R.all(({ state }) => state.type === LoadStateType.Ok, states)) {
    return load.ok(downloadState.totalBytes)
  }

  return load.downloading(downloadState)
}

type RequestState = {
  overall: LoadState
  individual: AssetState[]
}

const getOverallState = (overall: LoadState) => ({
  overall,
  individual: [],
})

async function processRequest(
  pullId: string,
  type: LoadRequestType,
  target: string
) {
  if (indexFetch != null) {
    await indexFetch
  }

  if (assetIndex == null) {
    throw new PullError('missing asset index')
  }

  let state: RequestState = {
    overall: load.waiting(),
    individual: [],
  }

  const setState = (newState: RequestState) => {
    const { overall, individual } = newState
    const response: StateResponse = {
      op: ResponseType.State,
      id: pullId,
      type,
      overall,
      individual,
    }
    self.postMessage(response)
    state = newState
  }

  const setDerivedState = (newState: AssetState[]) => {
    setState({
      overall: aggregateState(newState),
      individual: newState,
    })
  }

  const sendResponse = (
    status: ResponseStatus,
    data: Maybe<AssetData[]>,
    buffers?: Transferable[]
  ) => {
    const _result: Maybe<AssetResult> = data != null ? result.asset(data) : null
    const response: DataResponse = {
      op: ResponseType.Data,
      id: pullId,
      type,
      status,
      result: _result,
    }

    self.postMessage(response, buffers ?? [])
  }

  // Initialize state to waiting (and send it)
  setState(getOverallState(load.waiting()))

  const resolved = resolveRequest(type, target)
  if (resolved == null) {
    sendResponse(LoadStateType.Missing, null)
    return
  }
  const { bundles, assets } = crackBundles(resolved)

  setDerivedState([
    ...R.map(
      ({ id }): AssetState => ({
        type: DataType.Bundle,
        state: load.waiting(),
        id,
      }),
      bundles
    ),
    ...R.map(
      ({ id }): AssetState => ({
        type: DataType.Asset,
        state: load.waiting(),
        id,
      }),
      assets
    ),
  ])

  const update = (type: DataType, id: string) => (loadState: LoadState) => {
    setDerivedState(
      R.map((item) => {
        if (item.type !== type || item.id !== id) return item
        return { ...item, state: loadState }
      }, state.individual)
    )
  }

  try {
    const data: MountData[] = await Promise.all([
      ...R.map(
        ({ id }) => loadBundle(id, update(DataType.Bundle, id)),
        bundles
      ),
      ...R.map(
        (asset) => loadAsset(asset, update(DataType.Asset, asset.id)),
        assets
      ),
    ])

    const aggregated: MountData = R.reduce(
      (oldData: MountData, newData: MountData) => ({
        files: [...oldData.files, ...newData.files],
        buffers: [...oldData.buffers, ...newData.buffers],
      }),
      {
        files: [],
        buffers: [],
      },
      data
    )

    sendResponse(LoadStateType.Ok, aggregated.files, aggregated.buffers)
  } catch (e) {
    console.error(e)
    sendResponse(LoadStateType.Failed, null)
  }
}

async function processEnvironment(
  pullId: string,
  indices: string[]
): Promise<AssetIndex> {
  let state: RequestState = {
    overall: load.waiting(),
    individual: [],
  }

  const setState = (newState: RequestState) => {
    const { overall, individual } = newState
    const response: StateResponse = {
      op: ResponseType.State,
      id: pullId,
      type: null,
      overall,
      individual,
    }
    self.postMessage(response)
    state = newState
  }

  const setDerivedState = (newState: AssetState[]) => {
    setState({
      overall: aggregateState(newState),
      individual: newState,
    })
  }

  const sendResponse = (status: ResponseStatus, index: Maybe<AssetIndex>) => {
    const _result: Maybe<IndexResult> =
      index != null ? result.index(index) : null
    const response: DataResponse = {
      op: ResponseType.Data,
      id: pullId,
      type: null,
      status,
      result: _result,
    }

    self.postMessage(response, [])
  }

  // Initialize state to waiting (and send it)
  setState(getOverallState(load.waiting()))

  setDerivedState([
    ...R.map(
      (index): AssetState => ({
        type: DataType.Index,
        state: load.waiting(),
        id: index,
      }),
      indices
    ),
  ])

  const update = (id: string) => (loadState: LoadState) => {
    setDerivedState(
      R.map((item) => {
        if (item.id !== id) return item
        return { ...item, state: loadState }
      }, state.individual)
    )
  }

  try {
    const sources: AssetSource[] = await Promise.all([
      ...R.map((index) => loadIndex(index, update(index)), indices),
    ])

    const assetLookup: Record<string, number> = {}
    const bundleLookup: Record<string, BundleRef> = {}

    for (let i = 0; i < sources.length; i++) {
      const source = sources[i]
      for (const id of source.assets) {
        assetLookup[id] = i
      }
      for (const bundle of source.bundles) {
        bundleLookup[bundle.id] = [i, bundle]
      }
    }

    assetIndex = {
      assetLookup,
      bundleLookup,
      sources,
    }

    sendResponse(LoadStateType.Ok, assetIndex)

    return assetIndex
  } catch (e) {
    sendResponse(LoadStateType.Failed, null)
    throw e
  }
}

self.onmessage = (evt) => {
  const { data } = evt
  const request: Request = data

  if (request.op === RequestType.Environment) {
    const { assetSources: newSources } = request
    assetSources = newSources
    ;(async () => {
      indexFetch = processEnvironment('environment', newSources)
      await indexFetch
      indexFetch = null
    })()
  } else if (request.op === RequestType.Load) {
    const { target, type, id } = request
    processRequest(id, type, target)
  }
}
