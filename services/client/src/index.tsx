import styled from '@emotion/styled'
import { useResizeDetector } from 'react-resize-detector'
import start from './unsafe-startup'
import CBOR from 'cbor-js'
import * as React from 'react'
import * as R from 'ramda'
import ReactDOM from 'react-dom'
import {
  Center,
  ChakraProvider,
  Button,
  extendTheme,
  Flex,
  Box,
  VStack,
  Heading,
  Spacer,
} from '@chakra-ui/react'

import type { GameState } from './types'
import { GameStateType } from './types'
import type {
  AssetResponse,
  IndexResponse,
  Bundle,
  BundleState,
  BundleDownloadingState,
} from './assets/types'
import {
  ResponseType as AssetResponseType,
  RequestType as AssetRequestType,
  BundleLoadStateType,
} from './assets/types'
import StatusOverlay from './Loading'
import NAMES from './names'

start()

const colors = {
  brand: {
    900: '#1a365d',
    800: '#153e75',
    700: '#2a69ac',
  },
}

const theme = extendTheme({ colors })

const OuterContainer = styled.div`
  width: 100%;
  height: 100%;
  position: absolute;
  background-color: var(--chakra-colors-yellow-400);
`

const GameContainer = styled.div`
  width: 100%;
  height: 100%;
  position: absolute;
  z-index: 0;
`

const LoadingContainer = styled.div`
  backdrop-filter: blur(5px);
  width: 100%;
  height: 100%;
  position: absolute;
  z-index: 1;
`

const getDataName = (name: string) => `${name}.data`
const getBaseName = (dataName: string) => dataName.split('.')[1]

enum NodeType {
  Game,
  Map,
}

type PromiseSet<T> = {
  promise: Promise<T>
  resolve: (value: T) => void
  reject: (reason?: Error) => void
}

// Break up a promise into its resolve and reject functions for ease of use.
function breakPromise<T>(): PromiseSet<T> {
  let resolve: (value: T) => void = () => {}
  let reject: (reason?: Error) => void = () => {}
  const promise = new Promise<T>((_resolve, _reject) => {
    resolve = _resolve
    reject = _reject
  })

  return {
    promise,
    resolve,
    reject,
  }
}

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

type BundleRequest = {
  id: string
  promiseSet: PromiseSet<void>
}

function App() {
  const [state, setState] = React.useState<GameState>({
    type: GameStateType.PageLoading,
  })
  const { width, height, ref: containerRef } = useResizeDetector()

  const [bundleState, setBundleState] = React.useState<BundleState[]>([])

  const assetWorkerRef = React.useRef<Worker>()
  const requestStateRef = React.useRef<BundleRequest[]>([])

  const loadData = React.useCallback(async (target: string) => {
    const { current: assetWorker } = assetWorkerRef
    if (assetWorker == null) return

    const { current: requests } = requestStateRef

    const id = target
    const promiseSet = breakPromise<void>()

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
      new URL('./assets/worker.ts', import.meta.url),
      { type: 'module' }
    )

    worker.postMessage({
      op: AssetRequestType.Environment,
      ASSET_PREFIX: process.env.ASSET_PREFIX,
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

          resolve()

          requestStateRef.current = R.filter(
            ({ id: otherId }) => id !== otherId,
            requestStateRef.current
          )
        })()
      } else if (message.op === AssetResponseType.Index) {
        const { index } = message

        console.log(index);
        Module.FS_createPath("/", "packages", true, true)
        Module.FS_createPath("/packages/", "base", true, true)
        for (const [name, hash] of Object.entries(index)) {
          Module.FS_createDataFile("packages/base/", `${name}.stub`, '', true, true, true)
        }
      }
    }

    assetWorkerRef.current = worker
  }, [])

  React.useEffect(() => {
    let haveStarted: boolean = false
    let removeSubscribers: Array<(arg0: string) => boolean> = []

    // All of the files loaded by a map
    let nodes: PreloadNode[] = []
    let lastMap: Maybe<string> = null
    let loadingMap: Maybe<string> = null
    let targetMap: Maybe<string> = null

    Module.registerNode = (node) => {
      nodes.push(node)
    }

    // Load the basic required data for the game
    ;(async () => {
      await loadData('base')

      shouldRunNow = true
      calledRun = false
      Module.calledRun = false
      Module.run()
    })()

    Module.postLoadWorld = function () {
      BananaBread.execute('spawnitems')
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

      const loadMap = () => {
        setTimeout(() => {
          loadingMap = null
          if (targetMap == null) {
            BananaBread.loadWorld(map)
          } else {
            BananaBread.loadWorld(targetMap, map)
            targetMap = null
          }
        }, 1000)
      }

      if (dontHave.length === 0) {
        loadMap()
        return
      }

      await loadData(map)
      loadMap()
    }

    Module.print = (text) => {
      if (text === 'init: sdl') {
        setState({
          type: GameStateType.Running,
        })
      }

      if (text === 'init: mainloop') {
        setState({
          type: GameStateType.Ready,
        })
        Module.onGameReady()
      }

      if (text === 'connected to server') {
        targetMap = null
      }

      // Randomly assign a new name if the user joins without one
      if (text === 'setting name to: unnamed') {
        const name = NAMES[Math.floor(Math.random() * NAMES.length)]
        BananaBread.execute(`name ${name}`)
      }

      if (text.startsWith('received map')) {
        const [, , map, oldMap] = text.split(' ')
        if (oldMap != null && !oldMap.startsWith('getmap_')) {
          targetMap = map
          loadMapData(oldMap)
        } else {
          BananaBread.loadWorld(map)
        }
      }

      if (text.startsWith('load data for world: ')) {
        const map = text.split(': ')[1]
        loadMapData(map)
      }

      console.log(text)
    }
  }, [])

  React.useEffect(() => {
    if (width == null || height == null) return
    Module.desiredWidth = width
    Module.desiredHeight = height
    if (Module.setCanvasSize == null) return
    Module.setCanvasSize(width, height)
    if (BananaBread == null || BananaBread.execute == null) return
    BananaBread.execute(`screenres ${width} ${height}`)
  }, [width, height])

  React.useEffect(() => {
    const { protocol, host } = window.location
    const ws = new WebSocket(
      `${protocol === 'https:' ? 'wss://' : 'ws:/'}${host}/service/relay/`
    )
    ws.binaryType = 'arraybuffer'

    const injectServers = (servers: any) => {
      R.map((server) => {
        const { Host, Port, Info, Length } = server

        // Get data byte size, allocate memory on Emscripten heap, and get pointer
        const pointer = Module._malloc(Length)

        // Copy data to Emscripten heap (directly accessed from Module.HEAPU8)
        const dataHeap = new Uint8Array(Module.HEAPU8.buffer, pointer, Length)
        dataHeap.set(new Uint8Array(Info.buffer, Info.byteOffset, Length))

        // Call function and get result
        BananaBread.injectServer(Host, Port, pointer, Length)

        // Free memory
        Module._free(pointer)
      }, servers)
      BananaBread.execute('sortservers')
    }

    let cachedServers: Maybe<any> = null
    Module.onGameReady = () => {
      if (cachedServers == null) return
      injectServers(cachedServers)
    }

    ws.onmessage = (evt) => {
      const servers = CBOR.decode(evt.data)

      if (
        BananaBread == null ||
        BananaBread.execute == null ||
        BananaBread.injectServer == null
      ) {
        cachedServers = servers
        return
      }

      injectServers(servers)
    }
  }, [])

  React.useLayoutEffect(() => {
    const canvas = document.getElementById('canvas')
    if (canvas == null) return

    // As a default initial behavior, pop up an alert when webgl context is lost. To make your
    // application robust, you may want to override this behavior before shipping!
    // See http://www.khronos.org/registry/webgl/specs/latest/1.0/#5.15.2
    canvas.addEventListener(
      'webglcontextlost',
      function (e) {
        alert('WebGL context lost. You will need to reload the page.')
        e.preventDefault()
      },
      false
    )

    canvas.addEventListener('click', function () {
      canvas.requestPointerLock()
    })

    return
  }, [])

  return (
    <OuterContainer>
      <GameContainer ref={containerRef}>
        <canvas
          className="game"
          style={{ opacity: state.type !== GameStateType.Ready ? 0 : 1 }}
          id="canvas"
          ref={(canvas) => (Module.canvas = canvas)}
          onContextMenu={(event) => event.preventDefault()}
        ></canvas>
      </GameContainer>
      {state.type !== GameStateType.Ready && (
        <LoadingContainer>
          <Box w="100%" h="100%">
            <Heading>🍋Sour</Heading>
            <StatusOverlay state={state} />
          </Box>
        </LoadingContainer>
      )}
    </OuterContainer>
  )
}

ReactDOM.render(
  <ChakraProvider theme={theme}>
    <App />
  </ChakraProvider>,
  document.getElementById('root')
)
