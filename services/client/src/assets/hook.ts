import * as React from 'react'
import * as R from 'ramda'

import type {
  AssetData,
  LoadState,
  StateResponse,
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

// A semaphore for ensuring we don't render frames when assets are loading
let loadingCount: number = 0

function safeSetLoading(value: boolean) {
  if (BananaBread == null) return
  const { setLoading } = BananaBread
  if (setLoading == null) return
  setLoading(value)
}

function setLoading(value: boolean) {
  loadingCount = R.max(0, loadingCount + (value ? 1 : -1))
  safeSetLoading(loadingCount !== 0)
}

export async function pushLayer(
  assets: AssetData[],
  type: LoadRequestType
): Promise<Layer> {
  setLoading(true)
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

  setLoading(false)
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
    R.chain((source) => source.mods, index.sources)
  )
  const chunks = R.splitEvery(CHUNK_SIZE, mods)

  const header: string = R.join(
    '\n',
    R.map(([i, chunk]: [number, GameMod[]]) => {
      const list = R.map((v) => v.id, chunk)
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
  js (concatword "Module.assets.installMod('" $arg1 "')")
  showgui content
]

getmodinfo = [
  result (js (concatword "Module.assets.getModProperty('" $arg1 "','" $arg2 "')"))
]

showmodshot = [
    guibar
    mid = (checkrolloveraction "loadmod " [if (> $numargs 0) [result $arg1] [at $guirolloveraction 1]])
    guilist [
        guiimage (getmodinfo $mid image) (checkrolloveraction "loadmod ") 4 1 "data/cube.png" (getmodinfo $mid name)
        guitext (getmodinfo $mid description)
    ]
]

genmoditems = [
    looplist curmod $arg1 [
        guibutton (getmodinfo $curmod name) (concat loadmod $curmod) "cube"
    ]
]

newgui mods [
${tabs}
]`
}

const wrap = (length: number, s: string): string =>
  R.join('\n', R.splitEvery(length, s))

const TITLE_REGEX = /Blurb\n?\s+([^\n]+)/
function formatQuadropolis(description: string): string {
  const lines = description.split('\n')
  const title = TITLE_REGEX.exec(description)
  if (title == null) {
    return ''
  }
  return log.colors.blue(wrap(15, title[1].trim()))
}

function formatDescription(description: string): string {
  if (description.includes('Blurb')) {
    return formatQuadropolis(description)
  }

  return description
}

type ModLookup = Record<string, GameMod>

const MODS_KEY = 'sour-mods'
export function getInstalledMods(): string[] {
  const modString = localStorage.getItem(MODS_KEY)
  if (modString == null) return []
  return modString.split(',')
}

function setInstalledMods(mods: string[]) {
  if (mods.length === 0) {
    localStorage.removeItem(MODS_KEY)
    return
  }
  localStorage.setItem(MODS_KEY, R.join(',', mods))
}

function installMod(id: string) {
  setInstalledMods([...getInstalledMods(), id])
}

function removeMod(id: string) {
  setInstalledMods(R.filter((v: string) => v !== id, getInstalledMods()))
}

function getProgress(state: LoadState): number {
  const { type } = state
  switch (type) {
    case LoadStateType.Waiting:
    case LoadStateType.Missing:
    case LoadStateType.Failed:
      return 0
    case LoadStateType.Ok:
      return 1
    case LoadStateType.Downloading:
      if (state.type === LoadStateType.Downloading) {
        const { downloadedBytes, totalBytes } = state
        if (totalBytes === 0) {
          return 0
        }
        return downloadedBytes / totalBytes
      }
  }

  return 0
}

function getLoadProgress(
  downloadType: DownloadingType,
  state: StateResponse
): {
  message: string
  progress: number
} {
  const { individual } = state
  const total = individual.length
  const factor = 1 / total
  const progress = R.reduce(
    (a, v) => {
      const { state } = v
      return a + getProgress(state) * factor
    },
    0,
    individual
  )
  const done = R.filter(
    (v) => v.state.type === LoadStateType.Ok,
    individual
  ).length

  return {
    message: `Loading ${DownloadingType[
      downloadType
    ].toLowerCase()} data.. (${done}/${total})`,
    progress,
  }
}

export default function useAssets(
  setState: React.Dispatch<React.SetStateAction<GameState>>
): {
  loadAsset: (type: LoadRequestType, target: string) => Promise<Maybe<Layer>>
  getMod: (id: string) => Maybe<GameMod>
  onReady: () => void
} {
  const assetWorkerRef = React.useRef<Worker>()
  const requestStateRef = React.useRef<AssetRequest[]>([])
  const bundleIndexRef = React.useRef<AssetIndex>()
  const modLookupRef = React.useRef<ModLookup>()

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

  const getMod = React.useCallback((id: string): Maybe<GameMod> => {
    const { current: mods } = modLookupRef
    if (mods == null) return null
    const lookup = mods[id]
    if (lookup != null) return lookup

    // Now look by prefix
    const results = R.filter(
      (v: GameMod) => v.id.startsWith(id),
      R.values(mods)
    )
    if (results.length !== 1) return null
    return results[0]
  }, [])

  const loadAsset = React.useCallback(
    async (type: LoadRequestType, target: string): Promise<Maybe<Layer>> => {
      const { current: assetWorker } = assetWorkerRef
      if (assetWorker == null) return null

      const request = addRequest(target)
      if (target === 'environment') {
        assetWorker.postMessage({
          op: AssetRequestType.Environment,
          assetSources: CONFIG.assets,
        })
        return request.promiseSet.promise
      }

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

  const onReady = React.useCallback(() => {
    const { current: index } = bundleIndexRef
    if (index == null) return
    BananaBread.execute(buildModMenu(index))
  }, [])

  React.useEffect(() => {
    const worker = new Worker(
      // @ts-ignore
      new URL('./worker.ts', import.meta.url),
      { type: 'module' }
    )

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
          type === LoadRequestType.Map ||
          type === LoadRequestType.Mod ||
          type == null
        ) {
          const { message: text, progress } = getLoadProgress(
            downloadType,
            message
          )

          if (!Module.running) {
            setState({
              type: GameStateType.Downloading,
              progress,
              text,
            })
          } else {
            BananaBread.renderprogress(progress, text.toLowerCase())
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
            bundleIndexRef.current = index

            const lookup: ModLookup = {}
            for (const source of index.sources) {
              for (const mod of source.mods) {
                lookup[mod.id] = mod
              }
            }
            modLookupRef.current = lookup

            resolve(null)
            return
          }

          if (type == null) {
            return
          }

          const { data } = result
          try {
            const layer = await pushLayer(
              data.map((v) => ({ ...v, path: normalizePath(v.path) }), data),
              type
            )
            resolve(layer)
          } catch (e) {
            reject(e)
          }
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
      setLoading(true)
      if (loadingMap === map) return
      loadingMap = map

      if (mapLayer != null) {
        await removeLayer(mapLayer)
      }
      mapLayer = null

      const layer = await loadAsset(LoadRequestType.Map, map)
      if (layer == null) {
        console.error(`failed to load data for map ${map}`)
        BananaBread.execute('disconnect')
        return
      }

      Module.loadedMap(map)

      const loadMap = (realMap: string) => {
        mapLayer = layer
        loadingMap = null
        setLoading(false)
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
      modsToURL: () => {
        const mods = getInstalledMods()
        const {
          location: { search: params },
        } = window
        const parsedParams = new URLSearchParams(params)

        const { current: lookup } = modLookupRef
        if (lookup == null) return

        const allMods: GameMod[] = R.values(lookup)

        // Find the shortest unique identifier for every mod
        const minimal: string[] = R.chain((id: string): string[] => {
          const mod = getMod(id)
          if (mod == null) return []

          const { id: fullId } = mod
          const shortest = R.find(
            (shortId: string) => {
              for (const { id: otherId } of allMods) {
                if (fullId !== otherId && otherId.startsWith(shortId))
                  return false
              }
              return true
            },
            R.map((index) => fullId.slice(0, index), R.range(1, fullId.length))
          )

          if (shortest == null) return [fullId]
          return [shortest]
        }, mods)

        parsedParams.set('mods', R.join(',', minimal))
        window.location.search = parsedParams.toString()
      },
      getModProperty: (id: string, property: string): string => {
        const found = getMod(id)
        if (found == null) return ''

        switch (property) {
          case 'image':
            return getModImage(found)
          case 'description':
            return formatDescription(found.description ?? '')
          case 'name':
            const { name } = found
            if (R.includes(id, getInstalledMods())) {
              return log.colors.green(name)
            }
            return name
        }

        return ''
      },
      installMod: (target: string) => {
        const mod = getMod(target)
        if (mod == null) {
          log.error(`failed to find mod with id ${target}`)
          return
        }
        const { name, id } = mod
        if (R.includes(id, getInstalledMods())) {
          removeMod(id)
          log.success(
            `uninstalled mod ${name}. you must refresh the page for this to take effect.`
          )
          return
        }
        installMod(id)
        log.success(
          `installed mod ${name}. you must refresh the page for this to take effect.`
        )
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
            console.error(
              `texture ${name} not found anywhere or failed to load`
            )
          }
        })()
      },
      missingModel: (name: string, msg: number) => {
        if (models.has(name)) return
        models.add(name)
        ;(async () => {
          try {
            const layer = await loadAsset(LoadRequestType.Model, name)
            if (layer == null) {
              console.error(`model ${name} was null`)
            }
          } catch (e) {
            console.error(`model ${name} not found anywhere`)
          }
        })()
      },
      loadRandomMap: () => {
        const maps = getValidMaps(bundleIndexRef.current?.sources ?? [])
        const map = maps[Math.floor(maps.length * Math.random())]
        setTimeout(() => BananaBread.execute(`map ${map}`), 0)
      },
      loadWorld: (target: string) => loadMapData(target),
      receiveMap: (map: string, oldMap: string) => {
        if (
          oldMap != null &&
          oldMap.length > 0 &&
          !oldMap.startsWith('getmap_')
        ) {
          targetMap = map
          loadMapData(oldMap)
        } else {
          BananaBread.loadWorld(map)
        }
      },
    }
  }, [])

  return { loadAsset, getMod, onReady }
}
