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

const DIRECTIONS = [
  ['up', 'forward'],
  ['down', 'backward'],
  ['right', 'right'],
  ['left', 'left'],
]

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

  React.useEffect(() => {
    if (!isRunning) return
    const { current: container } = containerRef
    const { current: leftPad } = leftRef
    const { current: rightPad } = rightRef
    if (container == null || leftPad == null || rightPad == null) return

    let isInMenu: boolean = false

    let dx: number = 0
    let dy: number = 0

    let movement: Maybe<nipplejs.JoystickManager> = null
    let direction: Maybe<nipplejs.JoystickManager> = null

    const registerJoysticks = () => {
      movement = nipplejs.create({
        zone: leftPad,
        fadeTime: 0,
      })
      direction = nipplejs.create({
        zone: rightPad,
        fadeTime: 0,
      })

      movement
        .on('added', function (evt, nipple) {
          for (const [dir, command] of DIRECTIONS) {
            nipple.on(`dir:${dir}`, (evt) => {
              for (const [otherDir, otherCommand] of DIRECTIONS) {
                BananaBread.execute(
                  `_${otherCommand} ${dir === otherDir ? 1 : 0}`
                )
              }
            })
          }
        })
        .on('removed', function (evt, nipple) {
          for (const [dir, command] of DIRECTIONS) {
            nipple.off(`dir:${dir}`)
            BananaBread.execute(`_${command} 0`)
          }
        })

      direction
        .on('added', function (evt, nipple) {
          nipple.on('move', (_, data) => {
            const {
              distance,
              angle: { radian },
            } = data

            const factor = distance
            dx = Math.cos(radian) * factor
            dy = Math.sin(radian) * factor * -1
          })
        })
        .on('removed', function (evt, nipple) {
          dx = 0
          dy = 0
          nipple.off('move')
        })
    }

    const unregisterJoysticks = () => {
      if (movement != null) {
        movement.destroy()
        movement = null
      }
      if (direction != null) {
        direction.destroy()
        direction = null
      }
      dx = 0
      dy = 0
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
      if (isInMenu || (dx === 0 && dy === 0)) return
      BananaBread.mousemove(dx, dy)
    }
    window.requestAnimationFrame(cb)

    container.onpointerup = (evt) => {
      const { x, y, width, height } = container.getBoundingClientRect()
      const { x: mouseX, y: mouseY } = evt

      BananaBread.click((mouseX - x) / width, (mouseY - y) / height)
    }
  }, [isRunning])

  return (
    <Container ref={containerRef}>
      <TopLeftPanel>
        <Button onMouseDown={toggleMenu}>☰</Button>
      </TopLeftPanel>
      <MovementPad ref={leftRef} />
      <DirectionPad ref={rightRef} />
    </Container>
  )
}
