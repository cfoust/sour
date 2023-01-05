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
  IndexResponse,
  Bundle,
} from './types'
import {
  LoadStateType,
  LoadRequestType,
  ResponseType as AssetResponseType,
  RequestType as AssetRequestType,
} from './types'

import type { GameState } from '../types'
import { GameStateType } from '../types'

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

  const loadAsset = React.useCallback(
    async (type: LoadRequestType, target: string) => {
      const { current: assetWorker } = assetWorkerRef
      if (assetWorker == null) return

      const { current: requests } = requestStateRef

      const id = target
      const promiseSet = breakPromise<Maybe<AssetData[]>>()

      requestStateRef.current = [
        ...requests,
        {
          id,
          promiseSet,
        },
      ]

      assetWorker.postMessage({
        op: AssetRequestType.Load,
        type,
        id,
        target,
      })

      return promiseSet.promise
    },
    []
  )

  React.useEffect(() => {
    const worker = new Worker(
      // @ts-ignore
      new URL('./worker.ts', import.meta.url),
      { type: 'module' }
    )

    worker.postMessage({
      op: AssetRequestType.Environment,
      assetSources: CONFIG.assets,
    })

    worker.onmessage = (evt) => {
      const { data } = evt
      const message: Response = data

      if (message.op === AssetResponseType.State) {
        const { overall, type } = message

        // Show progress if maps or mods are downloading
        if (
          (type === LoadRequestType.Map || type === LoadRequestType.Mod) &&
          overall.type === LoadStateType.Downloading
        ) {
          const { downloadedBytes, totalBytes } = overall
          if (!Module.running) {
            setState({
              type: GameStateType.Downloading,
              downloadedBytes,
              totalBytes,
            })
          } else {
            BananaBread.renderprogress(
              downloadedBytes / totalBytes,
              'loading map data..'
            )
          }
        }
      } else if (message.op === AssetResponseType.Data) {
        const { id, data, status } = message

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

          // Mount the data first
          if (data != null) {
            await Promise.all(R.map((v) => mountFile(v.path, v.data), data))
          }

          resolve(data)
        })()
      } else if (message.op === AssetResponseType.Index) {
        const { index } = message
        bundleIndexRef.current = index
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

    // already mapped into packages/base/*.ogz, so it needs to be able to check
    // whether a map is valid before loading it
    const isValidMap = (map: string): number => {
      const maps = getValidMaps(bundleIndexRef.current ?? [])
      return maps.includes(map) ? 1 : 0
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
        console.error(`Failed to load bundle for map ${bundle}`)
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

    Module.assets = {
      isValidMap,
      isMountedFile,
      onConnect: () => {
        targetMap = null
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
