import * as React from 'react'
import * as R from 'ramda'

import type {
  AssetData,
  DownloadingState,
  AssetIndex,
  MountData,
  Response,
  GameMap,
  GameMod,
  AssetSource,
  Bundle,
} from './types'
import {
  ResultType,
  LoadStateType,
  LoadRequestType,
  ResponseType as AssetResponseType,
  RequestType as AssetRequestType,
} from './types'
import * as log from '../logging'

import type { GameState } from '../types'
import { GameStateType, DownloadingType } from '../types'

import type { PromiseSet } from '../utils'
import { breakPromise } from '../utils'
import { getModImage } from './utils'

import { CONFIG } from '../config'

enum NodeType {
  Game,
  Map,
}

export type Layer = {
  type: LoadRequestType
  data: Record<string, AssetData>
}

export type AssetRequest = {
  id: string
  promiseSet: PromiseSet<Maybe<Layer>>
}

function getValidMaps(sources: AssetSource[]): string[] {
  return R.pipe(
    R.chain((source: AssetSource) => source.maps),
    R.chain((map: GameMap) => [map.name, map.id])
  )(sources)
}

function getMods(sources: AssetSource[]): GameMod[] {
  return R.pipe(R.chain((source: AssetSource) => source.mods))(sources)
}

const getDataName = (name: string) => `${name}.data`
const getBaseName = (dataName: string) => dataName.split('.')[1]

function getDirectory(source: string): string {
  const lastSlash = source.lastIndexOf('/')
  if (lastSlash === -1) {
    return ''
  }

  return source.slice(0, lastSlash + 1)
}

function normalizePath(path: string): string {
  return path.startsWith('/') ? path.slice(1) : path
}

export async function mountFile(path: string, data: Uint8Array): Promise<void> {
  const normalizedPath = normalizePath(path)
  const parts = getDirectory(normalizedPath).split('/')
  for (let i = 0; i < parts.length; i++) {
    const first = parts.slice(0, i).join('/')
    const last = parts[i]
    Module.FS_createPath(`/${first}`, last, true, true)
  }
  return new Promise<void>((resolve, reject) => {
    Module.FS_createPreloadedFile(
      `/${normalizedPath}`,
      null,
      data,
      true,
      true,
      () => resolve(),
      () => {
        reject(new Error('Preloading file ' + path + ' failed'))
      },
      false,
      true
    )
  })
}
let layers: Layer[] = []

export async function pushLayer(
  assets: AssetData[],
  type: LoadRequestType
): Promise<Layer> {
  const data: Record<string, AssetData> = {}
  for (const asset of assets) {
    const { path } = asset
    data[normalizePath(path)] = asset
  }

  const newLayer: Layer = {
    type,
    data,
  }

  // Find the index at which we should insert this layer
  let targetIndex: number = R.findIndex(({ type: otherType }) => {
    if (type > otherType) return false
    if (type < otherType) return true
    // New layers of the same type always follow previous ones
    return false
  }, layers)

  if (targetIndex == -1) {
    targetIndex = layers.length
  }

  const before = layers.slice(0, targetIndex)
  const after = layers.slice(targetIndex + 1)

  // Clear out the previous layer's assets
  for (const asset of assets) {
    const { path } = asset

    let shouldMount = true
    // If any layers after this one have this file, don't mount it
    for (const { data: otherData } of after) {
      if (otherData[path] == null) continue
      shouldMount = false
    }

    if (!shouldMount) continue

    // If any layers before this one have this file, remove it
    for (const { data: otherData } of before.reverse()) {
      if (otherData[path] == null) continue
      try {
        FS.unlink(path)
      } catch (e) {}
    }

    await mountFile(path, asset.data)
  }

  layers = [...before, newLayer, ...after]

  return newLayer
}

export async function removeLayer(layer: Layer) {
  const index = R.findIndex((v) => v == layer, layers)
  if (index === -1) return
  const before = layers.slice(0, index)
  const after = layers.slice(index + 1)

  const assets = R.values(layer.data)
  for (const asset of assets) {
    const { path } = asset

    let isMounted: boolean = true
    for (const { data: otherData } of after) {
      if (otherData[path] == null) continue
      isMounted = false
    }

    if (!isMounted) continue

    // We can safely remove this
    try {
      FS.unlink(path)
    } catch (e) {}

    // If any layers before this one have this file, mount it
    for (const { data: otherData } of before.reverse()) {
      const otherAsset = otherData[path]
      if (otherAsset == null) continue
      await mountFile(path, otherAsset.data)
    }
  }
}

const CHUNK_SIZE = 17
function buildModMenu(index: AssetIndex): string {
  const mods: GameMod[] = R.sort(
    ({ name: nameA }, { name: nameB }): number => {
      if (nameA.length < nameB.length) return -1
      return R.ascend((v) => v)(nameA, nameB)
    },
    R.chain((source) => source.mods, index)
  )
  const chunks = R.splitEvery(CHUNK_SIZE, mods)

  const header: string = R.join(
    '\n',
    R.map(([i, chunk]: [number, GameMod[]]) => {
      const list = R.map((v) => v.name, chunk)
      return `gamemods${i + 1} = "${R.join(' ', list)}"`
    }, R.zip(R.range(0, chunks.length), chunks))
  )

  const tabGroups = R.splitEvery(3, R.range(0, chunks.length))

  const tabs: string = R.join(
    '',
    R.map(([i, ids]: [number, number[]]): string => {
      const entries = R.join(
        '',
        R.map(
          (id) => `
      guilist [ guistrut 15 1; genmoditems $gamemods${id + 1} ]`,
          ids
        )
      )
      return `
    ${i > 0 ? `guitab ${i + 1}` : ''}
    guilist [
      guistrut 17 1
      ${entries}
      showmodshot
    ]`
    }, R.zip(R.range(0, tabGroups.length), tabGroups))
  )

  return `
${header}

loadmod = [
  js (concatword "Module.assets.installMod('" $arg1) "')")
]

showmodshot = [
    guibar
    mname = (checkrolloveraction "loadmod " [if (> $numargs 0) [result $arg1] [at $guirollovername 0]])
    guilist [
        guiimage (js (concatword "Module.assets.getModImage('" $mname "')")) (checkrolloveraction "loadmod ") 4 1 "data/cube.png" $mname
    ]
]

genmoditems = [
    looplist curmod $arg1 [
        guibutton $curmod (concat loadmod $curmod) "cube"
    ]
]

newgui mods [
${tabs}
]`
}

export default function useAssets(
  setState: React.Dispatch<React.SetStateAction<GameState>>
): {
  loadAsset: (type: LoadRequestType, target: string) => Promise<Maybe<Layer>>
} {
  const assetWorkerRef = React.useRef<Worker>()
  const requestStateRef = React.useRef<AssetRequest[]>([])
  const bundleIndexRef = React.useRef<AssetIndex>()

  const addRequest = React.useCallback((id: string): AssetRequest => {
    const { current: requests } = requestStateRef
    const promiseSet = breakPromise<Maybe<Layer>>()
    const request: AssetRequest = {
      id,
      promiseSet,
    }

    requestStateRef.current = [...requests, request]
    return request
  }, [])

  const loadAsset = React.useCallback(
    async (type: LoadRequestType, target: string): Promise<Maybe<Layer>> => {
      const { current: assetWorker } = assetWorkerRef
      if (assetWorker == null) return null

      const request = addRequest(target)
      assetWorker.postMessage({
        op: AssetRequestType.Load,
        type,
        id: target,
        target,
      })

      return request.promiseSet.promise
    },
    []
  )

  React.useEffect(() => {
    const worker = new Worker(
      // @ts-ignore
      new URL('./worker.ts', import.meta.url),
      { type: 'module' }
    )

    addRequest('environment')
    worker.postMessage({
      op: AssetRequestType.Environment,
      assetSources: CONFIG.assets,
    })

    worker.onmessage = (evt) => {
      const { data } = evt
      const message: Response = data

      if (message.op === AssetResponseType.State) {
        const { overall, type } = message

        const downloadType =
          type === LoadRequestType.Map
            ? DownloadingType.Map
            : type === LoadRequestType.Mod
            ? DownloadingType.Mod
            : DownloadingType.Index

        // Show progress if maps or mods are downloading
        if (
          (type === LoadRequestType.Map ||
            type === LoadRequestType.Mod ||
            type == null) &&
          overall.type === LoadStateType.Downloading
        ) {
          const { downloadedBytes, totalBytes } = overall
          if (!Module.running) {
            setState({
              type: GameStateType.Downloading,
              downloadType,
              downloadedBytes,
              totalBytes,
            })
          } else {
            BananaBread.renderprogress(
              downloadedBytes / totalBytes,
              `loading ${DownloadingType[downloadType].toLowerCase()} data..`
            )
          }
        }
      } else if (message.op === AssetResponseType.Data) {
        const { id, result, status, type } = message

        ;(async () => {
          const { current: requests } = requestStateRef
          const request = R.find(({ id: otherId }) => id === otherId, requests)
          if (request == null) return

          const {
            promiseSet: { resolve, reject },
          } = request

          requestStateRef.current = R.filter(
            ({ id: otherId }) => id !== otherId,
            requestStateRef.current
          )

          if (status === LoadStateType.Failed) {
            reject()
            return
          }

          if (result == null) {
            resolve(null)
            return
          }

          if (result.type === ResultType.Index) {
            const { index } = result
            const modMenu = buildModMenu(index)
            BananaBread.execute(modMenu)
            bundleIndexRef.current = index
            resolve(null)
            return
          }

          if (type == null) {
            return
          }

          const { data } = result
          const layer = await pushLayer(
            data.map((v) => ({ ...v, path: normalizePath(v.path) }), data),
            type
          )
          resolve(layer)
        })()
      }
    }

    assetWorkerRef.current = worker
  }, [])

  React.useEffect(() => {
    // All of the files loaded by a map
    let loadingMap: Maybe<string> = null
    let targetMap: Maybe<string> = null
    let mapLayer: Maybe<Layer> = null

    const loadMapData = async (map: string) => {
      if (loadingMap === map) return
      loadingMap = map

      if (mapLayer != null) {
        await removeLayer(mapLayer)
      }
      mapLayer = null

      const layer = await loadAsset(LoadRequestType.Map, map)
      if (layer == null) {
        console.error(`failed to load data for map ${map}`)
        return
      }

      const loadMap = (realMap: string) => {
        mapLayer = layer
        loadingMap = null
        if (targetMap == null) {
          BananaBread.loadWorld(realMap)
        } else {
          BananaBread.loadWorld(targetMap, realMap)
          targetMap = null
        }
      }

      const mapFile = R.find(
        ({ path }) => path.endsWith('.ogz'),
        R.values(layer.data)
      )
      if (mapFile == null) {
        await removeLayer(layer)
        console.error('could not find map file in bundle')
        return
      }

      const { path } = mapFile
      const match = path.match(/packages\/base\/(.+).ogz/)
      if (match != null) {
        loadMap(match[1])
        return
      }

      const PACKAGES_PREFIX = '/packages/'
      if (path.startsWith(PACKAGES_PREFIX)) {
        loadMap(path.slice(PACKAGES_PREFIX.length))
        return
      }

      console.error(`map file was not in base ${mapFile.path}`)
      await removeLayer(layer)
    }

    const textures = new Set<string>()
    const models = new Set<string>()
    Module.assets = {
      getModImage: (name: string) => {
        const found = R.find(
          (v) => v.name === name,
          getMods(bundleIndexRef.current ?? [])
        )
        if (found == null) return ''
        return getModImage(found)
      },
      installMod: (name: string) => {
        console.log(name)
        const modName = name.trim()
        log.info(`installing mod ${modName}...`)
        ;(async () => {
          try {
            const layer = await loadAsset(LoadRequestType.Mod, modName)
            if (layer == null) {
              log.error(`mod ${modName} does not exist`)
              return
            }
            log.success(`installed ${modName}`)
          } catch (e) {
            log.error(`failed to install ${modName}`)
          }
        })()
      },
      onConnect: () => {
        targetMap = null
      },
      missingTexture: (name: string, msg: number) => {
        if (textures.has(name)) return
        textures.add(name)
        ;(async () => {
          try {
            const layer = await loadAsset(LoadRequestType.Texture, name)
            if (layer == null) {
              if (msg === 1) {
                log.vanillaError(`could not load texture ${name}`)
              }
              return
            }
            const [asset] = R.values(layer.data)

            mountFile(asset.path, asset.data)
            BananaBread.execute(`reloadtex ${name}`)
            // Sauer strips the packages/ from combined textures
            if (name.startsWith('packages/')) {
              BananaBread.execute(`reloadtex ${name.slice('packages/'.length)}`)
            }
          } catch (e) {
            console.error(`texture ${name} not found anywhere`)
          }
        })()
      },
      missingModel: (name: string, msg: number) => {
        if (models.has(name)) return
        models.add(name)
        ;(async () => {
          try {
            await loadAsset(LoadRequestType.Model, name)
          } catch (e) {
            console.error(`model ${name} not found anywhere`)
          }
        })()
      },
      loadRandomMap: () => {
        const maps = getValidMaps(bundleIndexRef.current ?? [])
        const map = maps[Math.floor(maps.length * Math.random())]
        setTimeout(() => BananaBread.execute(`map ${map}`), 0)
      },
      loadWorld: (target: string) => loadMapData(target),
      receivedMap: (map: string, oldMap: string) => {
        if (oldMap != null && !oldMap.startsWith('getmap_')) {
          targetMap = map
          loadMapData(oldMap)
        } else {
          BananaBread.loadWorld(map)
        }
      },
    }
  }, [])

  return { loadAsset }
}
