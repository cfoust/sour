import * as R from 'ramda'
import type {
  IndexAsset,
  MountData,
  Asset,
  BundleData,
  AssetData,
  Bundle,
  AssetLoadState,
  GameMod,
  GameMap,
  AssetState,
  AssetIndex,
  AssetSource,
  AssetRequest,
  AssetResponse,
  AssetStateResponse,
  IndexResponse,
  AssetDataResponse,
} from './types'
import {
  ResponseType,
  RequestType,
  AssetLoadStateType,
  AssetLoadType,
  LoadRequestType,
} from './types'

import { getBlob as getSavedBlob, saveBlob, haveBlob } from './storage'

class PullError extends Error {}

let assetSources: string[] = []
let bundleIndex: Maybe<AssetIndex> = null
let pullState: Record<string, AssetState[]> = {}

async function fetchIndex(source: string): Promise<AssetSource> {
  const response = await fetch(source)
  const index: AssetSource = await response.json()
  index.source = source
  return index
}

async function fetchIndices(): Promise<AssetIndex> {
  const indices = await Promise.all(R.map((v) => fetchIndex(v), assetSources))
  return indices
}

const updatePull = (pullId: string, newState: AssetState[]) => {
  pullState[pullId] = newState

  const update: AssetStateResponse = {
    op: ResponseType.State,
    id: pullId,
    status: AssetLoadStateType.Waiting,
    state: newState,
  }

  self.postMessage(update)
}

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
  source: string,
  id: string,
  progress: (bundle: AssetLoadState) => void
): Promise<ArrayBuffer> {
  const request = new XMLHttpRequest()
  const packageName = `${cleanPath(source)}${id}`
  request.open('GET', packageName, true)
  request.responseType = 'arraybuffer'
  request.onprogress = (event) => {
    if (!event.lengthComputable) return
    progress({
      type: AssetLoadStateType.Downloading,
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
        resolve(request.response)
      } else {
        throw new PullError(request.statusText + ' : ' + request.responseURL)
      }
    }
    request.send(null)
  })
}

async function loadBlob(
  source: string,
  id: string,
  url: string,
  progress: (bundle: AssetLoadState) => void
): Promise<ArrayBuffer> {
  if (await haveBlob(id)) {
    const buffer = await getSavedBlob(id)
    if (buffer == null) {
      throw new PullError(`Asset ${id} did not exist`)
    }
    return buffer
  }

  const buffer = await fetchData(source, url, progress)
  await saveBlob(id, buffer)
  return buffer
}

async function loadAsset(
  source: string,
  asset: Asset,
  progress: (bundle: AssetLoadState) => void
): Promise<MountData> {
  const { id, path } = asset

  const buffer = await loadBlob(source, id, id, progress)

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
  source: string,
  bundle: string,
  progress: (bundle: AssetLoadState) => void
): Promise<MountData> {
  const buffer = await loadBlob(source, bundle, `${bundle}.sour`, progress)
  const data = unpackBundle(buffer)

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

type FoundBundle = {
  source: string
  assets: Asset[]
  bundles: Bundle[]
}

const resolveBundles = (source: AssetSource, bundles: string[]): Bundle[] =>
  R.chain((id) => {
    const bundle = R.find((v) => v.id === id, source.bundles)
    if (bundle == null) return []
    return [bundle]
  }, bundles)

const resolveAssets = (source: AssetSource, assets: IndexAsset[]): Asset[] =>
  R.chain(([id, path]) => {
    return [
      {
        id: source.assets[id],
        path,
      },
    ]
  }, assets)

function resolveRequest(
  type: LoadRequestType,
  target: string
): Maybe<FoundBundle> {
  if (bundleIndex == null) return null

  switch (type) {
    case LoadRequestType.Map:
      for (const source of bundleIndex) {
        for (const map of source.maps) {
          if (map.name !== target && map.id !== target) continue
          const { bundle, assets } = map

          if (bundle != null) {
            return {
              source: source.source,
              assets: [],
              bundles: resolveBundles(source, [bundle]),
            }
          }

          return {
            source: source.source,
            assets: resolveAssets(source, assets),
            bundles: [],
          }
        }
      }
      break
    case LoadRequestType.Model:
      for (const source of bundleIndex) {
        for (const model of source.models) {
          if (model.name !== target && model.id !== target) continue
          const { id } = model

          return {
            source: source.source,
            assets: [],
            bundles: resolveBundles(source, [id]),
          }
        }
      }
      break
    case LoadRequestType.Texture:
      for (const source of bundleIndex) {
        for (const texture of source.textures) {
          const [index, path] = texture
          const asset = source.assets[index]
          if (path !== target && asset !== target) continue

          return {
            source: source.source,
            assets: [
              {
                id: asset,
                path,
              },
            ],
            bundles: [],
          }
        }
      }
      break
    case LoadRequestType.Mod:
      for (const source of bundleIndex) {
        for (const mod of source.mods) {
          if (mod.name !== target && mod.id !== target) continue

          return {
            source: source.source,
            assets: [],
            bundles: resolveBundles(source, [mod.id]),
          }
        }
      }
      break
  }

  return null
}

async function processLoad(
  pullId: string,
  type: LoadRequestType,
  target: string
) {
  if (bundleIndex == null) {
    bundleIndex = await fetchIndices()
  }

  const found = resolveRequest(type, target)

  if (found == null) {
    throw new Error(`Could not resolve ${LoadRequestType[type]} ${target}`)
  }

  const { source, bundles, assets } = found

  let state: AssetState[] = [
    ...R.map(
      ({ id }): AssetState => ({
        type: AssetLoadType.Bundle,
        state: {
          type: AssetLoadStateType.Waiting,
        },
        id,
      }),
      bundles
    ),
    ...R.map(
      ({ id }): AssetState => ({
        type: AssetLoadType.Asset,
        state: {
          type: AssetLoadStateType.Waiting,
        },
        id,
      }),
      assets
    ),
  ]

  const update =
    (type: AssetLoadType, id: string) => (loadState: AssetLoadState) => {
      state = R.map((item) => {
        if (item.type !== type || item.id !== id) return item
        return {
          ...item,
          state: loadState,
        }
      }, state)

      const response: AssetStateResponse = {
        op: ResponseType.State,
        // TODO
        status: AssetLoadStateType.Waiting,
        id: pullId,
        state,
      }

      self.postMessage(response)
    }

  try {
    const data: MountData[] = await Promise.all([
      ...R.map(
        ({ id }) => loadBundle(source, id, update(AssetLoadType.Bundle, id)),
        bundles
      ),
      ...R.map(
        (asset) =>
          loadAsset(source, asset, update(AssetLoadType.Asset, asset.id)),
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

    const response: AssetDataResponse = {
      op: ResponseType.Data,
      id: pullId,
      data: aggregated.files,
    }

    self.postMessage(response, aggregated.buffers)
  } catch (e) {
    if (!(e instanceof PullError)) throw e
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
    const { target, type, id } = request
    processLoad(id, type, target)
  }
}
