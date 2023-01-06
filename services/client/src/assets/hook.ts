import * as React from 'react'
import * as R from 'ramda'

import type {
  AssetData,
  DownloadingState,
  AssetIndex,
  MountData,
  Response,
  GameMap,
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

import { CONFIG } from '../config'

enum NodeType {
  Game,
  Map,
}

export type AssetRequest = {
  id: string
  promiseSet: PromiseSet<Maybe<AssetData[]>>
}

function getValidMaps(sources: AssetSource[]): string[] {
  return R.pipe(
    R.chain((source: AssetSource) => source.maps),
    R.chain((map: GameMap) => [map.name, map.id])
  )(sources)
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

export async function mountFile(path: string, data: Uint8Array): Promise<void> {
  const normalizedPath = path.startsWith('/') ? path.slice(1) : path
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

export default function useAssets(
  setState: React.Dispatch<React.SetStateAction<GameState>>
): {
  loadAsset: (
    type: LoadRequestType,
    target: string
  ) => Promise<Maybe<AssetData[]>>
} {
  const assetWorkerRef = React.useRef<Worker>()
  const requestStateRef = React.useRef<AssetRequest[]>([])
  const bundleIndexRef = React.useRef<AssetIndex>()

  const addRequest = React.useCallback((id: string): AssetRequest => {
    const { current: requests } = requestStateRef
    const promiseSet = breakPromise<Maybe<AssetData[]>>()
    const request: AssetRequest = {
      id,
      promiseSet,
    }

    requestStateRef.current = [...requests, request]
    return request
  }, [])

  const loadAsset = React.useCallback(
    async (
      type: LoadRequestType,
      target: string
    ): Promise<Maybe<AssetData[]>> => {
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
        const { id, result, status } = message

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
            resolve(null)
            return
          }

          const { data } = result
          // Mount the data first
          await Promise.all(R.map((v) => mountFile(v.path, v.data), data))
          resolve(data)
        })()
      }
    }

    assetWorkerRef.current = worker
  }, [])

  React.useEffect(() => {
    // All of the files loaded by a map
    let nodes: PreloadNode[] = []
    let lastMap: Maybe<string> = null
    let loadingMap: Maybe<string> = null
    let targetMap: Maybe<string> = null

    Module.registerNode = (node) => {
      nodes.push(node)
    }

    const isMountedFile = (filename: string): number => {
      const found = R.pipe(
        R.chain((node: PreloadNode) => node.files),
        R.find(
          (file) => file.filename == filename || file.filename == `/${filename}`
        )
      )(nodes)
      return found != null ? 1 : 0
    }

    const loadMapData = async (map: string) => {
      if (loadingMap === map) return
      loadingMap = map
      const need = ['base', map]

      // Clear out all of the old map files
      const [have, dontNeed] = R.partition(
        ({ name }) => need.includes(name),
        nodes
      )
      for (const node of dontNeed) {
        for (const file of node.files) {
          try {
            FS.unlink(file.filename)
          } catch (e) {
            console.error(`Failed to remove old map file: ${file}`)
          }
        }

        nodes = nodes.filter(({ name }) => name !== node.name)
      }

      const dontHave = R.filter(
        (base) =>
          R.find(({ name }) => name.endsWith(getDataName(base)), nodes) == null,
        need
      )

      const loadMap = (realMap: string) => {
        setTimeout(() => {
          loadingMap = null
          if (targetMap == null) {
            BananaBread.loadWorld(realMap)
          } else {
            BananaBread.loadWorld(targetMap, realMap)
            targetMap = null
          }
        }, 1000)
      }

      if (dontHave.length === 0) {
        loadMap(map)
        return
      }

      const bundle = await loadAsset(LoadRequestType.Map, map)
      if (bundle == null) {
        console.error(`Failed to load bundle for map ${map}`)
        return
      }

      const mapFile = R.find((file) => file.path.endsWith('.ogz'), bundle)
      if (mapFile == null) {
        console.error('Could not find map file in bundle')
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

      console.error(`Map file was not in base ${mapFile.path}`)
    }

    const textures = new Set<string>()
    const models = new Set<string>()
    Module.assets = {
      isMountedFile,
      onConnect: () => {
        targetMap = null
      },
      missingTexture: (name: string, msg: number) => {
        if (textures.has(name)) return
        textures.add(name)
        ;(async () => {
          try {
            const texture = await loadAsset(LoadRequestType.Texture, name)
            if (texture == null) {
              if (msg === 1) {
                log.vanillaError(`could not load texture ${name}`)
              }
              return
            }
            const [asset] = texture

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
            const assets = await loadAsset(LoadRequestType.Model, name)
            if (assets == null) {
              return
            }

            await Promise.all(R.map((v) => mountFile(v.path, v.data), assets))
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
