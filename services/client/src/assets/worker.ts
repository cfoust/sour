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

class PullError extends Error {}

let ASSET_PREFIX: string = ''
let pullState: BundleState[] = []

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

async function fetchBundle(
  target: string,
  progress: (bundle: BundleLoadState) => void
): Promise<Bundle> {
  const request = new XMLHttpRequest()
  const packageName = `${ASSET_PREFIX}${target}.sour`
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
        resolve(unpackBundle(request.response))
      } else {
        throw new PullError(request.statusText + ' : ' + request.responseURL)
      }
    }
    request.send(null)
  })
}

async function loadBundle(
  target: string,
  progress: (bundle: BundleLoadState) => void
): Promise<Bundle> {
  const bundle = await fetchBundle(target, progress)
  return bundle
}

async function processLoad(target: string, id: symbol) {
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
    const bundle = await loadBundle(target, update)

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
