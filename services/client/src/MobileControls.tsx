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
  min-height: 100vh;
  position: absolute;
  top: 0;
  bottom: 0;
  left: 0;
  right: 0;
`

const MovementPad = styled.div`
  width: 50%;
  height: 100%;
  position: absolute;
`

const DirectionPad = styled.div`
  right: 0;
  width: 50%;
  height: 100%;
  position: absolute;
`

export default function MobileControls() {
  const leftRef = React.useRef<HTMLDivElement>(null)
  const rightRef = React.useRef<HTMLDivElement>(null)

  React.useLayoutEffect(() => {
    const { current: leftPad } = leftRef
    const { current: rightPad } = rightRef
    if (leftPad == null || rightPad == null) return

    const movement = nipplejs.create({
      zone: leftPad,
      fadeTime: 0,
    })

    const directions = [
      ['up', 'forward'],
      ['down', 'backward'],
      ['right', 'right'],
      ['left', 'left'],
    ]

    movement
      .on('added', function (evt, nipple) {
        for (const [dir, command] of directions) {
          nipple.on(`dir:${dir}`, (evt) => {
            for (const [otherDir, otherCommand] of directions) {
              BananaBread.execute(
                `_${otherCommand} ${dir === otherDir ? 1 : 0}`
              )
            }
          })
        }
      })
      .on('removed', function (evt, nipple) {
        for (const [dir, command] of directions) {
          nipple.off(`dir:${dir}`)
          BananaBread.execute(`_${command} 0`)
        }
      })

    let dx: number = 0
    let dy: number = 0
    const direction = nipplejs.create({
      zone: rightPad,
      fadeTime: 0,
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

    const cb = () => {
      window.requestAnimationFrame(cb)
      if (dx === 0 && dy === 0) return
      BananaBread.mousemove(dx, dy)
    }
    window.requestAnimationFrame(cb)

  }, [])

  return (
    <Container>
      <MovementPad ref={leftRef} />
      <DirectionPad ref={rightRef} />
    </Container>
  )
}
