import * as React from 'react'
import * as R from 'ramda'

import type {
  AssetResponse,
  GameMap,
  AssetSource,
  IndexResponse,
  Bundle,
  BundleIndex,
  BundleState,
  BundleDownloadingState,
} from './types'
import {
  ResponseType as AssetResponseType,
  RequestType as AssetRequestType,
  BundleLoadStateType,
} from './types'

import type { GameState } from '../types'
import { GameStateType } from '../types'

import type { PromiseSet } from '../utils'
import { breakPromise } from '../utils'

enum NodeType {
  Game,
  Map,
}

export type BundleRequest = {
  id: string
  promiseSet: PromiseSet<Bundle>
}

function getValidMaps(sources: AssetSource[]): string[] {
  return R.pipe(
    R.chain((source: AssetSource) => source.maps),
    R.chain((map: GameMap) => [map.name, ...map.aliases])
  )(sources)
}

const getDataName = (name: string) => `${name}.data`
const getBaseName = (dataName: string) => dataName.split('.')[1]

async function mountBundle(target: string, bundle: Bundle): Promise<void> {
  const { directories, files, buffer, dataOffset } = bundle

  Module.registerNode({
    name: target,
    files,
  })

  for (const directory of directories) {
    Module.FS_createPath(...directory, true, true)
  }

  await Promise.all(
    R.map(({ filename, start, end, audio }) => {
      const offset = dataOffset + start
      const ref = `fp ${filename}`
      return new Promise<void>((resolve, reject) => {
        Module.FS_createPreloadedFile(
          filename,
          null,
          new Uint8Array(buffer, offset, end - start),
          true,
          true,
          () => resolve(),
          () => {
            reject(new Error('Preloading file ' + filename + ' failed'))
          },
          false,
          true
        )
      })
    }, files)
  )
}

export default function useAssets(
  setState: React.Dispatch<React.SetStateAction<GameState>>
): {
  loadBundle: (target: string) => Promise<Bundle | undefined>
} {
  const [bundleState, setBundleState] = React.useState<BundleState[]>([])
  const assetWorkerRef = React.useRef<Worker>()
  const requestStateRef = React.useRef<BundleRequest[]>([])
  const bundleIndexRef = React.useRef<BundleIndex>()

  const loadBundle = React.useCallback(async (target: string) => {
    const { current: assetWorker } = assetWorkerRef
    if (assetWorker == null) return

    const { current: requests } = requestStateRef

    const id = target
    const promiseSet = breakPromise<Bundle>()

    requestStateRef.current = [
      ...requests,
      {
        id,
        promiseSet,
      },
    ]

    assetWorker.postMessage({
      op: AssetRequestType.Load,
      id,
      target,
    })

    return promiseSet.promise
  }, [])

  React.useEffect(() => {
    const worker = new Worker(
      // @ts-ignore
      new URL('./worker.ts', import.meta.url),
      { type: 'module' }
    )

    worker.postMessage({
      op: AssetRequestType.Environment,
      ASSET_SOURCE: process.env.ASSET_SOURCE,
    })

    worker.onmessage = (evt) => {
      const { data } = evt
      const message: AssetResponse = data

      if (message.op === AssetResponseType.State) {
        const { state } = message

        setBundleState(state)

        const downloading: BundleDownloadingState[] = R.chain(({ state }) => {
          if (state.type !== BundleLoadStateType.Downloading) return []
          return [state]
        }, state)

        // Show progress if any bundles are downloading.
        if (downloading.length > 0) {
          const { downloadedBytes, totalBytes } = R.reduce(
            (
              { downloadedBytes: currentDownload, totalBytes: currentTotal },
              { downloadedBytes: newDownload, totalBytes: newTotal }
            ) => ({
              downloadedBytes: currentDownload + newDownload,
              totalBytes: currentTotal + newTotal,
            }),
            {
              downloadedBytes: 0,
              totalBytes: 0,
            },
            downloading
          )

          if (BananaBread.renderprogress == null) {
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
      } else if (message.op === AssetResponseType.Bundle) {
        const { target, id, bundle } = message

        ;(async () => {
          const { current: requests } = requestStateRef
          const request = R.find(({ id: otherId }) => id === otherId, requests)
          if (request == null) return

          await mountBundle(target, bundle)

          const {
            promiseSet: { resolve },
          } = request

          resolve(bundle)

          requestStateRef.current = R.filter(
            ({ id: otherId }) => id !== otherId,
            requestStateRef.current
          )
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

    // We want Sauerbraten to behave as though all of the available maps were
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

      const bundle = await loadBundle(map)
      if (bundle == null) {
        console.error(`Failed to load bundle for map ${bundle}`)
        return
      }

      const mapFile = R.find(
        (file) => file.filename.endsWith('.ogz'),
        bundle.files
      )
      if (mapFile == null) {
        console.error('Could not find map file in bundle')
        return
      }

      const { filename } = mapFile
      const match = filename.match(/packages\/base\/(.+).ogz/)
      if (match != null) {
        loadMap(match[1])
        return
      }

      const PACKAGES_PREFIX = '/packages/'
      if (filename.startsWith(PACKAGES_PREFIX)) {
        loadMap(filename.slice(PACKAGES_PREFIX.length))
        return
      }

      console.error(`Map file was not in base ${mapFile.filename}`)
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

  return { loadBundle }
}
