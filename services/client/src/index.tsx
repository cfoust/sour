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
import type {
  ServerMessage,
  SocketMessage,
  CommandMessage,
  PacketMessage,
} from './protocol'
import { GameStateType } from './types'
import { MessageType, ENetEventType } from './protocol'
import StatusOverlay from './Loading'
import NAMES from './names'
import useAssets from './assets/hook'
import { CubeMessageType } from './game'
import * as cube from './game'

import type { PromiseSet } from './utils'
import { CONFIG } from './config'
import { breakPromise } from './utils'
import * as log from './logging'

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

const pushURLState = (url: string) => {
  window.history.pushState({}, '', url)
}

const clearURLState = () => pushURLState('/')

export type CommandRequest = {
  id: number
  promiseSet: PromiseSet<string>
}

const SERVER_URL_REGEX = /\/server\/([\w.]+)\/?(\d+)?/

function App() {
  const [state, setState] = React.useState<GameState>({
    type: GameStateType.PageLoading,
  })
  const { width, height, ref: containerRef } = useResizeDetector()

  const { loadBundle } = useAssets(setState)

  React.useEffect(() => {
    // Load the basic required data for the game
    ;(async () => {
      await loadBundle('base')

      shouldRunNow = true
      calledRun = false
      Module.calledRun = false
      Module.run()
    })()

    Module.socket = (addr, port) => {
      const { protocol, host } = window.location
      const prefix = `${
        protocol === 'https:' ? 'wss://' : 'ws:/'
      }${host}/service/proxy/`

      return new WebSocket(
        addr === 'sour' ? prefix : `${prefix}u/${addr}:${port}`,
        ['binary']
      )
    }

    Module.print = (text) => {
      if (text === 'init: sdl') {
        setState({
          type: GameStateType.Running,
        })
      }

      // Randomly assign a new name if the user joins without one
      if (text === 'setting name to: unnamed') {
        const name = NAMES[Math.floor(Math.random() * NAMES.length)]
        BananaBread.execute(`name ${name}`)
      }

      if (text.startsWith('main loop blocker')) {
        return
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
    // All commands in flight
    let commands: CommandRequest[] = []

    const [clusterURL] = CONFIG.clusters

    const { protocol, host } = window.location
    const ws = new WebSocket(
      `${protocol === 'https:' ? 'wss://' : 'ws:/'}${clusterURL}`
    )
    ws.binaryType = 'arraybuffer'

    const runCommand = async (command: string) => {
      const generate = (): number => Math.floor(Math.random() * 2048)

      let id: number = generate()

      // We don't want collisions and can't use a Symbol
      while (R.find((v) => v.id === id, commands) != null) {
        id = generate()
      }

      const promiseSet = breakPromise<string>()

      commands = [
        ...commands,
        {
          id,
          promiseSet,
        },
      ]

      const message: CommandMessage = {
        Op: MessageType.Command,
        Command: command,
        Id: id,
      }

      ws.send(CBOR.encode(message))

      return promiseSet.promise
    }

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

    // This is only used when we make a match and are waiting for players.
    let waitingForPlayers: boolean = false
    let lastSour: Maybe<string> = null
    let warningTimer: Maybe<NodeJS.Timer> = null
    const waitWarning = () => {
      log.info(
        `This is your private server. Invite other players using the link in your URL bar (we copied it to your clipboard) or having them type /join ${lastSour} at sourga.me.`
      )
    }

    let serverEvents: SocketMessage[] = []
    let queuedEvents: SocketMessage[] = []
    let loadingWorld = false

    Module.postLoadWorld = function () {
      loadingWorld = false
      serverEvents = queuedEvents
      if (waitingForPlayers && lastSour) {
        navigator.clipboard.writeText(location.href)
        waitWarning()
        warningTimer = setInterval(waitWarning, 30000)
      }
    }

    Module.onClientJoin = () => {
      if (waitingForPlayers) {
        waitingForPlayers = false
      }
      if (warningTimer != null) {
        clearInterval(warningTimer)
      }
    }

    let cachedServers: Maybe<any> = null
    Module.onGameReady = () => {
      setState({
        type: GameStateType.Ready,
      })

      if (cachedServers != null) {
        injectServers(cachedServers)
      }

      const {
        location: { search: params, pathname },
      } = window

      const serverDestination = SERVER_URL_REGEX.exec(pathname)
      if (serverDestination != null) {
        const [, hostname, port] = serverDestination
        if (port == null) {
          BananaBread.execute(`join ${hostname}`)
        } else {
          BananaBread.execute(`connect ${hostname} ${port}`)
        }
      } else {
        // It should not be anything else
        pushURLState('/')
      }

      if (params.length == 0) return
      const parsedParams = new URLSearchParams(params)
      if (!parsedParams.has('cmd')) return
      const cmd = parsedParams.get('cmd')
      if (cmd == null) return
      setTimeout(() => BananaBread.execute(cmd), 0)
    }

    Module.onConnect = (name: string, port: number) => {
      // Sour server
      if (port === 0) {
        // what is even going on?
        if (name.length != 0) {
          pushURLState(`/server/${name}`)
          return
        }

        if (lastSour == null) return
        pushURLState(`/server/${lastSour}`)
        return
      }

      pushURLState(`/server/${name}/${port}`)
    }
    Module.onDisconnect = () => {
      waitingForPlayers = false
      if (warningTimer != null) {
        clearInterval(warningTimer)
      }
      lastSour = null
      pushURLState(`/`)
    }

    let lastPointer: number = 0
    let lastPointerLength: number = 0

    // Only allocate memory if we really need to
    const malloc = (size: number) => {
      // reduce, reuse, recycle
      if (size <= lastPointerLength) {
        return lastPointer
      }

      if (lastPointer !== 0) {
        Module._free(lastPointer)
      }

      const pointer = Module._malloc(size)
      lastPointer = pointer
      lastPointerLength = size
      return pointer
    }

    Module.cluster = {
      createGame: (preset: string) => {
        log.info('creating private game...')
        ;(async () => {
          try {
            const result = await runCommand('creategame')
            log.success('created game!')
            BananaBread.execute(`join ${result}`)
            waitingForPlayers = true
          } catch (e) {
            log.error(`failed to create private game: ${e}`)
          }
        })()
      },
      connect: (name: string, password: string) => {
        const Target = name.length === 0 ? 'lobby' : name
        lastSour = Target
        ws.send(
          CBOR.encode({
            Op: MessageType.Connect,
            Target,
          })
        )
      },
      send: (channel: number, dataPtr: number, dataLength: number) => {
        const packet = new Uint8Array(dataLength)
        packet.set(new Uint8Array(Module.HEAPU8.buffer, dataPtr, dataLength))
        ws.send(
          CBOR.encode({
            Op: MessageType.Packet,
            Channel: channel,
            Data: packet,
            Length: dataLength,
          })
        )
      },
      receive: (dataPtr: number, dataLengthPtr: number) => {
        const view = new DataView(Module.HEAPU8.buffer)

        const message = serverEvents.shift()
        if (message == null) {
          return 0
        }

        if (message.Op === MessageType.ServerConnected) {
          // Layout:
          // 2: Event
          const frameLength = 2
          const pointer = malloc(frameLength)
          view.setUint16(pointer, ENetEventType.Connect, true)
          return pointer
        }

        if (message.Op === MessageType.ServerDisconnected) {
          const { Reason } = message

          // Layout:
          // 2: Event
          // 2: Reason
          const frameLength = 2 + 2
          const pointer = malloc(frameLength)
          view.setUint16(pointer, ENetEventType.Disconnect, true)
          view.setUint16(pointer + 2, Reason, true)
          return pointer
        }

        const { Channel, Data, Length } = message

        // Layout:
        // 2: Event
        // 2: Channel
        // 4: Length
        // N: Data
        const frameLength = 2 + 2 + 4 + Length
        const pointer = malloc(frameLength)

        // sourEvent
        view.setUint16(pointer, ENetEventType.Receive, true)
        // sourChannel
        view.setUint16(pointer + 2, Channel, true)
        // dataLength
        view.setUint32(pointer + 4, Length, true)

        // Copy in from data
        const dataHeap = new Uint8Array(
          Module.HEAPU8.buffer,
          pointer + 8,
          Length
        )
        dataHeap.set(new Uint8Array(Data.buffer, Data.byteOffset, Length))

        return pointer
      },
      disconnect: () => {
        ws.send(
          CBOR.encode({
            Op: MessageType.Disconnect,
          })
        )
      },
    }

    ws.onmessage = (evt) => {
      const serverMessage: ServerMessage = CBOR.decode(evt.data)

      if (serverMessage.Op === MessageType.Info) {
        const { Cluster, Master } = serverMessage

        if (
          BananaBread == null ||
          BananaBread.execute == null ||
          BananaBread.injectServer == null
        ) {
          cachedServers = Master
          return
        }

        injectServers(Master)
        return
      }

      if (serverMessage.Op === MessageType.ServerResponse) {
        const { Id, Response, Success } = serverMessage
        const request = R.find(({ id: otherId }) => Id === otherId, commands)
        if (request == null) return

        const {
          promiseSet: { resolve, reject },
        } = request

        if (Success) {
          resolve(Response)
        } else {
          reject(new Error(Response))
        }

        commands = R.filter(({ id: otherId }) => Id !== otherId, commands)
        return
      }

      if (serverMessage.Op === MessageType.Packet) {
        const packet = cube.newPacket(serverMessage.Data)
        const msgType = cube.getInt(packet)
        if (msgType === CubeMessageType.N_MAPCHANGE) {
          loadingWorld = true
          serverEvents.push(serverMessage)
          return
        }
        if (
          loadingWorld &&
          (msgType === CubeMessageType.N_CLIENTPING ||
            msgType === CubeMessageType.N_PONG)
        ) {
          serverEvents.push(serverMessage)
          return
        }
        //if (msgType != null && loadingWorld) {
        //console.log(`queueing ${CubeMessageType[msgType]}`)
        //}
      }

      if (loadingWorld) {
        queuedEvents.push(serverMessage)
      } else {
        serverEvents.push(serverMessage)
      }
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
