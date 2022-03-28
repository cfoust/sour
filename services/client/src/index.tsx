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

const DOWNLOAD_REGEX = /Downloading data... \((\d+)\/(\d+)\)/

const handleDownload = (
  text: string,
  handler: (downloadedBytes: number, totalBytes: number) => void
) => {
  const result = DOWNLOAD_REGEX.exec(text)
  if (result == null) return
  const [, completedText, totalText] = result
  const downloadedBytes = parseInt(completedText)
  const totalBytes = parseInt(totalText)
  handler(downloadedBytes, totalBytes)
}

const getPreloadName = (name: string) => `preload_${name}.js`
const getDataName = (name: string) => `${name}.data`
const getBaseName = (dataName: string) => dataName.split('.')[1]

function loadData(name: string) {
  const js = document.createElement('script')
  js.src = `${ASSET_PREFIX}${getPreloadName(name)}`
  js.onerror = () => {
    BananaBread.execute(`echo Failed to load data ${name}; disconnect`)
  }
  document.body.appendChild(js)
}

const MAIN_LOOP_REGEX = /main loop blocker "(\w+)" took 1 ms/

const handleBlocker = (text: string, handler: (func: string) => void) => {
  const result = MAIN_LOOP_REGEX.exec(text)
  if (result == null) return
  const [, func] = result
  handler(func)
}

enum NodeType {
  Game,
  Map,
}

function App() {
  const [state, setState] = React.useState<GameState>({
    type: GameStateType.PageLoading,
  })
  const { width, height, ref: containerRef } = useResizeDetector()

  React.useEffect(() => {
    let removeSubscribers: Array<(arg0: string) => boolean> = []

    // All of the files loaded by a map
    let nodes: PreloadNode[] = []
    let lastMap: Maybe<string> = null
    let haveStarted: boolean = false

    Module.registerNode = (node) => {
      nodes.push(node)
    }

    Module.preInit.push(() => {
      const _removeRunDependency = Module.removeRunDependency
      Module.removeRunDependency = (file) => {
        let newSubscribers = []
        for (const callback of removeSubscribers) {
          if (!callback(file)) newSubscribers.push(callback)
        }
        removeSubscribers = newSubscribers

        _removeRunDependency(file)
      }

      const _monitorRunDependencies = Module.monitorRunDependencies
      Module.monitorRunDependencies = (left) => {
        _monitorRunDependencies(left)

        // Wait for it to be ready
        if (nodes.length > 0 && left === 0 && !haveStarted) {
          shouldRunNow = true
          Module.run()
        }
      }
    })

    // Load the basic required data for the game
    loadData('base')

    Module.setStatus = (text) => {
      // Sometimes we get download progress this way, handle it here
      handleDownload(text, (downloadedBytes, totalBytes) => {
        if (BananaBread.renderprogress == null) return
        BananaBread.renderprogress(
          downloadedBytes / totalBytes,
          'loading map data..'
        )
      })
    }

    Module.postLoadWorld = function () {
      Module.tweakDetail()
      BananaBread.execute('spawnitems')
      BananaBread.execute('clearconsole')
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
      }

      // Randomly assign a new name if the user joins without one
      if (text === 'setting name to: unnamed') {
        const name = NAMES[Math.floor(Math.random() * NAMES.length)]
        BananaBread.execute(`name ${name}`)
      }

      if (text.startsWith('load data for world: ')) {
        const map = text.split(': ')[1]

        // Clear out all of the old map files
        const need = ['base', map]
        const [have, dontNeed] = R.partition(
          ({ name }) => need.includes(getBaseName(name)),
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

          Module._free(node.pointer)
          nodes = nodes.filter(({ name }) => name !== node.name)
        }

        const dontHave = R.filter(
          (base) =>
            R.find(({ name }) => name.endsWith(getDataName(base)), nodes) ==
            null,
          need
        )

        const loadMap = () => {
          setTimeout(() => {
            BananaBread.execute(`reallyloadworld ${map}`)
          }, 1000)
        }

        if (dontHave.length === 0) {
          loadMap()
          return
        }

        loadData(map)

        removeSubscribers.push((file) => {
          if (!file.endsWith(`${map}.data`)) return false

          loadMap()

          return true
        })
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
    const { protocol, hostname } = window.location
    const ws = new WebSocket(
      `${protocol === 'https:' ? 'wss://' : 'ws:/'}${hostname}/service/relay/`
    )
    ws.binaryType = 'arraybuffer'
    ws.onmessage = (evt) => {
      const servers = CBOR.decode(evt.data)

      if (
        BananaBread == null ||
        BananaBread.execute == null ||
        BananaBread.injectServer == null
      )
        return

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
