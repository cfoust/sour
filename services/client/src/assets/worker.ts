import * as R from 'ramda'
import type {
  Bundle,
  BundleState,
  BundleLoadState,
  AssetRequest,
  AssetResponse,
  AssetStateResponse,
  AssetBundleResponse,
} from './types'
import { ResponseType, RequestType, BundleLoadStateType } from './types'

import { getBundle as getSavedBundle, saveBundle, haveBundle } from './storage'

class PullError extends Error {}

let ASSET_PREFIX: string = ''
let bundleIndex: Maybe<Record<string, string>> = null
let pullState: BundleState[] = []

async function fetchIndex(): Promise<Record<string, string>> {
  const response = await fetch(`${ASSET_PREFIX}index`)
  const index = await response.text()

  const newIndex: Record<string, string> = {}
  for (const line of index.split('\n')) {
    const [name, hash] = line.split(' ')
    newIndex[name] = hash
  }

  return newIndex
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

function cleanPath(): string {
  const lastSlash = ASSET_PREFIX.lastIndexOf('/')
  if (lastSlash === -1) {
    return ''
  }

  return ASSET_PREFIX.slice(0, lastSlash + 1)
}

async function fetchBundle(
  hash: string,
  progress: (bundle: BundleLoadState) => void
): Promise<ArrayBuffer> {
  const request = new XMLHttpRequest()
  const packageName = `${cleanPath()}${hash}.sour`
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
  hash: string,
  progress: (bundle: BundleLoadState) => void
): Promise<Bundle> {
  if (await haveBundle(hash)) {
    const buffer = await getSavedBundle(hash)
    if (buffer == null) {
      throw new PullError(`Bundle ${hash} did not exist`)
    }
    return unpackBundle(buffer)
  }

  const buffer = await fetchBundle(hash, progress)
  await saveBundle(hash, buffer)
  return unpackBundle(buffer)
}

async function processLoad(target: string, id: string) {
  if (bundleIndex == null) {
    bundleIndex = await fetchIndex()
  }

  const hash = bundleIndex[target]

  if (hash == null) {
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
    const bundle = await loadBundle(hash, update)

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
    const { ASSET_PREFIX: newPrefix } = request
    ASSET_PREFIX = newPrefix
  } else if (request.op === RequestType.Load) {
    const { target, id } = request
    processLoad(target, id)
  }
}
