import * as R from 'ramda'
import type {
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

import { getBundle as getSavedBundle, saveBundle, haveBundle } from './storage'

class PullError extends Error {}

let ASSET_SOURCE: string = ''
let bundleIndex: Maybe<BundleIndex> = null
let pullState: BundleState[] = []

async function fetchIndex(source: string): Promise<AssetSource> {
  const response = await fetch(source)
  const index: AssetSource = await response.json()
  index.source = source
  for (const map of index.maps) {
    map.name = map.name.replace('.ogz', '')
  }
  return index
}

async function fetchIndices(): Promise<BundleIndex> {
  const sources = ASSET_SOURCE.split(';')
  const indices = await Promise.all(R.map((v) => fetchIndex(v), sources))
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

const INT_SIZE = 4
function unpackBundle(data: ArrayBuffer): Bundle {
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

function cleanPath(source: string): string {
  const lastSlash = source.lastIndexOf('/')
  if (lastSlash === -1) {
    return ''
  }

  return source.slice(0, lastSlash + 1)
}

async function fetchBundle(
  source: string,
  bundle: string,
  progress: (bundle: BundleLoadState) => void
): Promise<ArrayBuffer> {
  const request = new XMLHttpRequest()
  const packageName = `${cleanPath(source)}${bundle}.sour`
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

async function loadBundle(
  source: string,
  bundle: string,
  progress: (bundle: BundleLoadState) => void
): Promise<Bundle> {
  if (await haveBundle(bundle)) {
    const buffer = await getSavedBundle(bundle)
    if (buffer == null) {
      throw new PullError(`Bundle ${bundle} did not exist`)
    }
    return unpackBundle(buffer)
  }

  const buffer = await fetchBundle(source, bundle, progress)
  await saveBundle(bundle, buffer)
  return unpackBundle(buffer)
}

type FoundBundle = {
  source: string
  bundle: string
}

function findBundle(target: string): Maybe<FoundBundle> {
  if (bundleIndex == null) return null

  for (const source of bundleIndex) {
    for (const mod of source.mods) {
      if (mod.name !== target) continue
      return {
        source: source.source,
        bundle: mod.bundle,
      }
    }

    for (const map of source.maps) {
      if (map.name !== target && !map.aliases.includes(target)) continue
      return {
        source: source.source,
        bundle: map.bundle,
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
    const bundle = await loadBundle(found.source, found.bundle, update)

    update({
      type: BundleLoadStateType.Ok,
    })

    const response: AssetBundleResponse = {
      op: ResponseType.Bundle,
      id,
      target,
      bundle,
    }

    self.postMessage(response, [bundle.buffer])
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
    const { ASSET_SOURCE: newPrefix } = request
    ASSET_SOURCE = newPrefix
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
