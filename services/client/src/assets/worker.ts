import * as R from 'ramda'
import type {
  Asset,
  AssetData,
  Bundle,
  BundleState,
  BundleLoadState,
  GameMod,
  GameMap,
  BundleIndex,
  AssetSource,
  AssetRequest,
  AssetResponse,
  AssetStateResponse,
  IndexResponse,
  AssetBundleResponse,
} from './types'
import { ResponseType, RequestType, BundleLoadStateType } from './types'

import { getBundle as getSavedBundle, saveAsset, haveBundle } from './storage'

class PullError extends Error {}

let assetSources: string[] = []
let bundleIndex: Maybe<BundleIndex> = null
let pullState: BundleState[] = []

async function fetchIndex(source: string): Promise<AssetSource> {
  const response = await fetch(source)
  const index: AssetSource = await response.json()
  index.source = source
  return index
}

async function fetchIndices(): Promise<BundleIndex> {
  const indices = await Promise.all(R.map((v) => fetchIndex(v), assetSources))
  return indices
}

const sendState = (newState: BundleState[]) => {
  pullState = newState
  const update: AssetStateResponse = {
    op: ResponseType.State,
    state: pullState,
  }

  self.postMessage(update)
}

const updateBundle = (target: string, state: BundleState) => {
  const bundle = R.find(({ name }) => name === target, pullState)

  if (bundle == null) {
    sendState([...pullState, state])
    return
  }

  sendState(R.map((v) => (v.name === target ? state : v), pullState))
}

function cleanPath(source: string): string {
  const lastSlash = source.lastIndexOf('/')
  if (lastSlash === -1) {
    return ''
  }

  return source.slice(0, lastSlash + 1)
}

async function fetchAsset(
  source: string,
  asset: string,
  progress: (bundle: BundleLoadState) => void
): Promise<ArrayBuffer> {
  const request = new XMLHttpRequest()
  const packageName = `${cleanPath(source)}${asset}`
  request.open('GET', packageName, true)
  request.responseType = 'arraybuffer'
  request.onprogress = (event) => {
    if (!event.lengthComputable) return
    progress({
      type: BundleLoadStateType.Downloading,
      downloadedBytes: event.loaded,
      totalBytes: event.total,
    })
  }
  request.onerror = function (event) {
    throw new PullError('NetworkError for: ' + packageName)
  }

  return new Promise((resolve, reject) => {
    request.onload = function (event) {
      if (
        request.status == 200 ||
        request.status == 304 ||
        request.status == 206 ||
        (request.status == 0 && request.response)
      ) {
        const packageData = request.response
        resolve(request.response)
      } else {
        throw new PullError(request.statusText + ' : ' + request.responseURL)
      }
    }
    request.send(null)
  })
}

async function loadAsset(
  source: string,
  asset: Asset,
  progress: (bundle: BundleLoadState) => void
): Promise<AssetData> {
  const { id, path } = asset
  if (await haveBundle(id)) {
    const buffer = await getSavedBundle(id)
    if (buffer == null) {
      throw new PullError(`Asset ${id} did not exist`)
    }
    return {
      path,
      data: buffer,
    }
  }

  const buffer = await fetchAsset(source, id, progress)
  await saveAsset(id, buffer)
  return {
    data: buffer,
    path,
  }
}

async function loadAssets(
  source: string,
  assets: Asset[],
  progress: (bundle: BundleLoadState) => void
): Promise<AssetData[]> {
  return Promise.all(R.map((v) => loadAsset(source, v, progress), assets))
}

type FoundBundle = {
  source: string
  assets: Asset[]
}

function findBundle(target: string): Maybe<FoundBundle> {
  if (bundleIndex == null) return null

  for (const source of bundleIndex) {
    for (const mod of source.mods) {
      if (mod.name !== target) continue
      return {
        source: source.source,
        assets: R.map((v) => source.assets[v], mod.assets),
      }
    }

    for (const map of source.maps) {
      if (map.name !== target && map.id !== target) continue
      return {
        source: source.source,
        assets: R.map((v) => source.assets[v], map.assets),
      }
    }
  }

  return null
}

async function processLoad(target: string, id: string) {
  if (bundleIndex == null) {
    bundleIndex = await fetchIndices()
  }

  const found = findBundle(target)

  if (found == null) {
    throw new Error(`No hash for ${target} found in index`)
  }

  const update = (state: BundleLoadState) => {
    updateBundle(target, {
      name: target,
      state,
    })
  }

  update({
    type: BundleLoadStateType.Waiting,
  })

  try {
    const data = await loadAssets(found.source, found.assets, update)

    update({
      type: BundleLoadStateType.Ok,
    })

    const response: AssetBundleResponse = {
      op: ResponseType.Bundle,
      id,
      target,
      data,
    }

    self.postMessage(
      response,
      R.map((v) => v.data, data)
    )
  } catch (e) {
    if (!(e instanceof PullError)) throw e

    update({
      type: BundleLoadStateType.Failed,
    })
  }
}

self.onmessage = (evt) => {
  const { data } = evt
  const request: AssetRequest = data

  if (request.op === RequestType.Environment) {
    const { assetSources: newSources } = request
    assetSources = newSources
    ;(async () => {
      bundleIndex = await fetchIndices()

      const response: IndexResponse = {
        op: ResponseType.Index,
        index: bundleIndex,
      }

      self.postMessage(response, [])
    })()
  } else if (request.op === RequestType.Load) {
    const { target, id } = request
    processLoad(target, id)
  }
}
