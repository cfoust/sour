import styled from '@emotion/styled'
import { useResizeDetector } from 'react-resize-detector'
import start from './unsafe-startup'
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
import { vec3 } from 'gl-matrix'

import type { GameState, EntityState, User } from './types'
import { GameStateType } from './types'
import StatusOverlay from './Loading'
import PeerManager from './spatial/PeerManager'
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

function loadMap(name: string) {
  var js = document.createElement('script')
  js.src = `${ASSET_PREFIX}/preload_${name}.js`
  document.body.appendChild(js)
}

// From https://developer.mozilla.org/en-US/docs/Web/API/PannerNode/orientationX
// this utility converts amount of rotation around the Y axis
// (i.e. rotation in the 'horizontal plane') to an orientation vector
const yRotationToVector = (degrees: number) => {
  // convert degrees to radians and offset the angle so 0 points towards the listener
  const radians = (degrees - 90) * (Math.PI / 180)
  // using cosine and sine here ensures the output values are always normalized
  // i.e. they range between -1 and 1
  const x = Math.cos(radians)
  const z = Math.sin(radians)

  // we hard-code the Y component to 0, as Y is the axis of rotation
  return [x, 0, z]
}

function App() {
  const [state, setState] = React.useState<GameState>({
    type: GameStateType.PageLoading,
  })
  const { width, height, ref: containerRef } = useResizeDetector()

  const entityState = React.useRef<Maybe<EntityState>>(null)

  const [isVoiceEnabled, setVoiceEnabled] = React.useState<boolean>(true)

  const spatialRef = React.useRef<PeerManager>()

  const destroySpatial = React.useCallback(() => {
    spatialRef.current?.terminate()
  }, [])

  const connectToSpatial = React.useCallback((clientId: number) => {
    if (spatialRef.current != null) destroySpatial()
    const peer = new PeerManager(clientId, console.log)
    peer.connect()
    spatialRef.current = peer
  }, [])

  // Initialize everything for voice
  React.useEffect(() => {
    const myID = entityState.current?.me.id
    if (!isVoiceEnabled || myID == null) return
    connectToSpatial(myID)
  }, [connectToSpatial, isVoiceEnabled])

  React.useEffect(() => {
    if (!isVoiceEnabled) return

    let stop = false
    const cb = (delta: number) => {
      if (stop) return
      requestAnimationFrame(cb)
      const { current: state } = entityState
      const { current: peerManager } = spatialRef
      if (state == null || peerManager == null) return

      const {
        me: { position, heading },
        users: rest,
      } = state

      let out = vec3.create()
      for (const client of rest) {
        const { id, position: userPosition, heading: userHeading } = client
        const panner = peerManager.getUser(id)?.panner
        if (panner == null) continue

        const [x, y, z] = vec3.scale(
          vec3.create(),
          vec3.subtract(out, userPosition, position),
          0.25
        )

        const targetHeading = yRotationToVector(userHeading - heading)
        panner.orientationX.value = targetHeading[0]
        panner.orientationY.value = targetHeading[1]
        panner.orientationZ.value = targetHeading[2]

        panner.positionX.value = x
        panner.positionY.value = y
        panner.positionZ.value = z
      }
    }

    requestAnimationFrame(cb)

    return () => {
      stop = true
    }
  }, [isVoiceEnabled])

  const setEntityState = React.useCallback(
    (setter: (state: EntityState) => EntityState) => {
      const { current } = entityState
      entityState.current = setter(
        current == null
          ? {
              me: {
                id: 0,
                name: '',
                position: [0, 0, 0],
                heading: 0,
                speaking: false,
                muted: false,
              },
              users: [],
            }
          : current
      )
    },
    []
  )

  React.useEffect(() => {
    let removeSubscribers: Array<(arg0: string) => boolean> = []

    Module.setStatus = (text) => {
      // Sometimes we get download progress this way, handle it here
      handleDownload(text, (downloadedBytes, totalBytes) =>
        setState({
          type: GameStateType.Downloading,
          downloadedBytes,
          totalBytes,
        })
      )
    }

    Module.postLoadWorld = function () {
      Module.tweakDetail()
      BananaBread.execute('spawnitems')
      BananaBread.execute('clearconsole')
      setState({
        type: GameStateType.Connected,
      })
    }

    Module.postRun.push(() => {
      const _removeRunDependency = Module.removeRunDependency
      Module.removeRunDependency = (file) => {
        let newSubscribers = []
        for (const callback of removeSubscribers) {
          if (!callback(file)) newSubscribers.push(callback)
        }
        removeSubscribers = newSubscribers

        _removeRunDependency(file)
      }
    })

    Module.onPlayerMove = (cn, pos, yaw, mypos, myyaw) => {
      setEntityState((state) => {
        return {
          me: {
            ...state.me,
            position: mypos,
            heading: myyaw,
          },
          users: R.map(
            (user: User) => ({
              ...user,
              position: user.id === cn ? pos : user.position,
              heading: user.id === cn ? yaw : user.heading,
            }),
            state.users
          ),
        }
      })
    }

    Module.onPlayerJoin = (cn) => {
      setEntityState((state) => ({
        ...state,
        users: [
          ...state.users,
          {
            id: cn,
            name: '',
            position: [0, 0, 0],
            heading: 0,
            speaking: false,
            muted: false,
          },
        ],
      }))
    }

    Module.onClientNumber = (cn) => {
      setEntityState((state) => ({
        ...state,
        me: {
          ...state.me,
          id: cn,
        },
      }))
      connectToSpatial(cn)
    }

    Module.onPlayerName = (cn, name) => {
      setEntityState((state) => {
        const { me, users } = state
        if (me.id === cn) {
          return {
            ...state,
            me: {
              ...me,
              name,
            },
          }
        }

        return {
          ...state,
          users: R.map(
            (user: User) => ({
              ...user,
              name: user.id === cn ? name : user.name,
            }),
            state.users
          ),
        }
      })
    }

    Module.onPlayerLeave = (cn) => {
      setEntityState((state) => ({
        ...state,
        users: R.filter((user) => user.id !== cn, state.users),
      }))
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

      if (text.startsWith('load data for world: ')) {
        const map = text.split(': ')[1]
        loadMap(map)

        setState({
          type: GameStateType.MapChange,
          map,
        })

        removeSubscribers.push((file) => {
          if (!file.endsWith(`${map}.data`)) return false
          setTimeout(() => {
            BananaBread.execute(`reallyloadworld ${map}`)
          }, 1000)

          return true
        })
      }

      console.log(text)
    }
  }, [connectToSpatial])

  React.useEffect(() => {
    if (width == null || height == null) return
    Module.desiredWidth = width
    Module.desiredHeight = height
    if (Module.setCanvasSize == null) return
    Module.setCanvasSize(width, height)
  }, [width, height])

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
          style={{ opacity: state.type !== GameStateType.Connected ? 0 : 1 }}
          id="canvas"
          ref={(canvas) => (Module.canvas = canvas)}
          onContextMenu={(event) => event.preventDefault()}
        ></canvas>
      </GameContainer>
      {state.type !== GameStateType.Connected && (
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
