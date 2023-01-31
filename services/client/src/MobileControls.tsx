import styled from '@emotion/styled'
import * as React from 'react'
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

import nipplejs from 'nipplejs'

import PISTOL_ICON from 'url:./static/pistol.png'

const Container = styled.div`
  width: 100%;
  height: 100%;
  position: absolute;
  top: 0;
  bottom: 0;
  left: 0;
  right: 0;
  user-select: none;
`

const MovementPad = styled.div`
  width: 50%;
  height: 100%;
  position: absolute;
  z-index: 0;
`

const DirectionPad = styled.div`
  right: 0;
  width: 50%;
  height: 100%;
  position: absolute;
  z-index: 0;
`

const TopLeftPanel = styled.div`
  left: 0;
  top: 0;
  position: absolute;
  z-index: 1;
`

const BottomRightPanel = styled.div`
  right: 0;
  bottom: 0;
  position: absolute;
  z-index: 1;
`

const BottomLeftPanel = styled.div`
  left: 0;
  bottom: 0;
  position: absolute;
  z-index: 1;
`

const ActionButton = styled.div`
  border-radius: 40px;
  border-width: 1px;
  border-color: white;
  background: var(--chakra-colors-whiteAlpha-300);
  width: 60px;
  height: 60px;
  font-size: 30px;

  display: flex;
  place-items: center;
  justify-content: center;
  flex-direction: column;

  &:active {
    border-color: var(--chakra-colors-red-500);
  }
`

const DIRECTIONS: Array<[nipplejs.JoystickEventTypes, string]> = [
  ['dir:up', 'forward'],
  ['dir:down', 'backward'],
  ['dir:right', 'right'],
  ['dir:left', 'left'],
]

const MOTION_FACTOR = 5

type MotionMachine = Record<number, Touch>

function newTouchMachine(): MotionMachine {
  return {}
}

function handleTouchStart(
  machine: MotionMachine,
  touches: TouchList
): MotionMachine {
  const result: MotionMachine = { ...machine }
  for (let i = 0; i < touches.length; i++) {
    const touch = touches[i]
    const { clientX: x } = touch
    if (x < window.screen.width / 2) {
      continue
    }
    result[touch.identifier] = touch
  }
  return result
}

type Motion = [x: number, y: number]

function handleTouchMove(
  machine: MotionMachine,
  touches: TouchList
): [MotionMachine, Motion[]] {
  const result: MotionMachine = { ...machine }
  const movements: Motion[] = []
  for (let i = 0; i < touches.length; i++) {
    const newTouch = touches[i]
    const oldTouch = result[newTouch.identifier]
    if (oldTouch == null) {
      continue
    }

    movements.push([
      newTouch.clientX - oldTouch.clientX,
      newTouch.clientY - oldTouch.clientY,
    ])

    result[newTouch.identifier] = newTouch
  }

  return [result, movements]
}

function handleTouchEnd(
  machine: MotionMachine,
  touches: TouchList
): MotionMachine {
  const result: MotionMachine = {}

  const removed: string[] = []
  for (let i = 0; i < touches.length; i++) {
    const touch = touches[i]
    removed.push(touch.identifier.toString())
  }

  for (const [id, touch] of Object.entries(machine)) {
    if (removed.includes(id)) {
      continue
    }
    result[touch.identifier] = touch
  }

  return result
}

function useCreateAction(
  isRunning: boolean,
  command: string
): [
  down: (event: React.TouchEvent) => void,
  up: (event: React.TouchEvent) => void
] {
  const down = React.useCallback(
    (event: React.TouchEvent) => {
      event.preventDefault()
      if (!isRunning) return
      console.log(`${command} 1`)
      BananaBread.execute(`${command} 1`)
    },
    [isRunning]
  )
  const up = React.useCallback(
    (event: React.TouchEvent) => {
      event.preventDefault()
      if (!isRunning) return
      console.log(`${command} 0`)
      BananaBread.execute(`${command} 0`)
    },
    [isRunning]
  )

  return [down, up]
}

export default function MobileControls(props: { isRunning: boolean }) {
  const { isRunning } = props
  const containerRef = React.useRef<HTMLDivElement>(null)
  const leftRef = React.useRef<HTMLDivElement>(null)
  const rightRef = React.useRef<HTMLDivElement>(null)

  const toggleMenu = React.useCallback(
    (event: React.MouseEvent) => {
      event.preventDefault()
      if (!isRunning) return
      BananaBread.execute('togglemainmenu')
    },
    [isRunning]
  )

  const [jumpDown, jumpUp] = useCreateAction(isRunning, '_jump')
  const [attackDown, attackUp] = useCreateAction(isRunning, '_attack')

  React.useEffect(() => {
    if (!isRunning) return
    const { current: container } = containerRef
    const { current: leftPad } = leftRef
    const { current: rightPad } = rightRef
    if (container == null || leftPad == null || rightPad == null) return

    let isInMenu: boolean = false

    let movement: Maybe<nipplejs.FixedJoystickManager> = null

    const registerJoysticks = () => {
      movement = nipplejs.create({
        zone: leftPad,
        fadeTime: 0,
      })

      movement.on('added', function (evt, nipple) {
        for (const [dir, command] of DIRECTIONS) {
          nipple.on(dir, (evt) => {
            for (const [otherDir, otherCommand] of DIRECTIONS) {
              BananaBread.execute(
                `_${otherCommand} ${dir === otherDir ? 1 : 0}`
              )
            }
          })
        }
      })

      movement.on('removed', function (evt, nipple) {
        for (const [dir, command] of DIRECTIONS) {
          nipple.off(dir, () => {})
          BananaBread.execute(`_${command} 0`)
        }
      })
    }

    const unregisterJoysticks = () => {
      if (movement != null) {
        movement.destroy()
        movement = null
      }
    }

    const cb = () => {
      window.requestAnimationFrame(cb)
      if (!Module.running) return
      const newInMenu = BananaBread.isInMenu() === 1
      if (!isInMenu && newInMenu) {
        unregisterJoysticks()
      } else if (isInMenu && !newInMenu) {
        registerJoysticks()
      }
      if (!newInMenu && movement == null) {
        registerJoysticks()
      }
      isInMenu = newInMenu
    }
    window.requestAnimationFrame(cb)

    container.onpointerup = (evt) => {
      const { x, y, width, height } = container.getBoundingClientRect()
      const { x: mouseX, y: mouseY } = evt

      BananaBread.click((mouseX - x) / width, (mouseY - y) / height)
    }

    let machine: MotionMachine = newTouchMachine()
    container.ontouchstart = (evt) => {
      machine = handleTouchStart(machine, evt.changedTouches)
    }
    container.ontouchmove = (evt) => {
      const [newMachine, motions] = handleTouchMove(machine, evt.changedTouches)
      machine = newMachine
      if (motions.length == 0) return
      const [motion] = motions
      const [dx, dy] = motion
      if (isInMenu || (dx === 0 && dy === 0)) return
      BananaBread.mousemove(dx * MOTION_FACTOR, dy * MOTION_FACTOR)
    }
    container.ontouchend = (evt) => {
      machine = handleTouchEnd(machine, evt.changedTouches)
    }
  }, [isRunning])

  return (
    <Container ref={containerRef}>
      <TopLeftPanel>
        <Button onMouseDown={toggleMenu}>☰</Button>
      </TopLeftPanel>
      <BottomLeftPanel>
        <ActionButton
          onTouchStart={attackDown}
          style={{
            position: 'absolute',
              bottom: 150,
              left: 60,
          }}
        >
          <span style={{ marginTop: -5 }}>⌖</span>
        </ActionButton>
      </BottomLeftPanel>
      <BottomRightPanel>
        <ActionButton
          onTouchStart={jumpDown}
          style={{
            position: 'absolute',
            bottom: 30,
            right: 100,
          }}
        >
          <span style={{ marginTop: -5 }}>▲</span>
        </ActionButton>
        <ActionButton
          onTouchStart={attackDown}
          style={{
            position: 'absolute',
            width: 80,
            height: 80,
            bottom: 100,
            right: 100,
            fontSize: 40,
          }}
        >
          <span style={{ marginTop: -10 }}>⌖</span>
        </ActionButton>
        <Button leftIcon={<img src={PISTOL_ICON} width={16} height={16} />}>
          16
        </Button>
      </BottomRightPanel>
      <MovementPad ref={leftRef} />
      <DirectionPad ref={rightRef} />
    </Container>
  )
}
