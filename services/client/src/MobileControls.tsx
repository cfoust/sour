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

import SAW_ICON from 'url:./static/saw.png'
import SHOTGUN_ICON from 'url:./static/shotgun.png'
import CHAINGUN_ICON from 'url:./static/machinegun.png'
import ROCKET_ICON from 'url:./static/rocket.png'
import RIFLE_ICON from 'url:./static/rifle.png'
import GRENADE_ICON from 'url:./static/grenade.png'
import PISTOL_ICON from 'url:./static/pistol.png'

import type { PlayerState } from './types'

import { WeaponType } from './types'

type Weapon = {
  type: WeaponType
  icon: string
}

const WEAPON_INFO: Weapon[] = [
  {
    type: WeaponType.Saw,
    icon: SAW_ICON,
  },
  {
    type: WeaponType.Shotgun,
    icon: SHOTGUN_ICON,
  },
  {
    type: WeaponType.Chaingun,
    icon: CHAINGUN_ICON,
  },
  {
    type: WeaponType.Rocket,
    icon: ROCKET_ICON,
  },
  {
    type: WeaponType.Rifle,
    icon: RIFLE_ICON,
  },
  {
    type: WeaponType.Grenade,
    icon: GRENADE_ICON,
  },
  {
    type: WeaponType.Pistol,
    icon: PISTOL_ICON,
  },
]

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

  display: flex;
  flex-direction: column;
  align-items: flex-end;
`

const BottomLeftPanel = styled.div`
  left: 0;
  bottom: 0;
  position: absolute;
  z-index: 1;
`

const GunElement = styled.div<{ selected: boolean; disabled: boolean }>`
  border-radius: 5px;
  border-width: 1px;
  border-color: white;
  padding: 3px;
  width: 70px;

  ${(p) => (p.selected ? 'border-color: var(--chakra-colors-red-500);' : '')}
  ${(p) => (p.disabled ? 'opacity: 0.4;' : '')}

  & > span {
    margin-left: 5px;
    margin-right: 5px;
    width: 30px;
    text-align: right;
  }

  display: flex;
  flex-direction: row-reverse;
  align-items: center;
`

const ActionButton = styled.div<{ active?: boolean }>`
  border-radius: 40px;
  border-width: 1px;
  border-color: white;
  background: var(--chakra-colors-whiteAlpha-300);
  width: 60px;
  height: 60px;
  font-size: 30px;

  ${(p) =>
    p.active === true ? 'border-color: var(--chakra-colors-red-500);' : ''}

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

type TouchAction = () => void
type TrackedTouch = {
  touch: React.Touch
  endAction: Maybe<TouchAction>
}

function newTrackedTouch(
  touch: React.Touch,
  endAction?: TouchAction
): TrackedTouch {
  return { touch, endAction }
}

type TouchMachine = Record<number, TrackedTouch>

function newTouchMachine(): TouchMachine {
  return {}
}

function handleTouchStart(
  machine: TouchMachine,
  touches: React.TouchList,
  endAction?: TouchAction
): TouchMachine {
  const result: TouchMachine = { ...machine }
  for (let i = 0; i < touches.length; i++) {
    const touch = touches[i]
    const { clientX: x } = touch
    if (result[touch.identifier] != null) {
      continue
    }
    result[touch.identifier] = newTrackedTouch(touch, endAction)
  }
  return result
}

type Motion = [x: number, y: number]

function handleTouchMove(
  machine: TouchMachine,
  touches: React.TouchList
): [TouchMachine, Motion[]] {
  const result: TouchMachine = { ...machine }
  const movements: Motion[] = []
  for (let i = 0; i < touches.length; i++) {
    const newTouch = touches[i]
    const oldTouch = result[newTouch.identifier]
    if (oldTouch == null) {
      continue
    }

    movements.push([
      newTouch.clientX - oldTouch.touch.clientX,
      newTouch.clientY - oldTouch.touch.clientY,
    ])

    result[newTouch.identifier] = { ...oldTouch, touch: newTouch }
  }

  return [result, movements]
}

function handleTouchEnd(
  machine: TouchMachine,
  touches: React.TouchList
): TouchMachine {
  const result: TouchMachine = {}

  const removed: string[] = []
  for (let i = 0; i < touches.length; i++) {
    const touch = touches[i]
    removed.push(touch.identifier.toString())
  }

  for (const [id, touch] of Object.entries(machine)) {
    if (removed.includes(id)) {
      const { endAction } = touch
      if (endAction != null) {
        endAction()
      }
      continue
    }
    result[touch.touch.identifier] = touch
  }

  return result
}

function GunButton(props: {
  type: WeaponType
  icon: string
  playerState: PlayerState
  selectWeapon: () => void
}) {
  const { type, icon, playerState, selectWeapon } = props
  const { ammo, weapon } = playerState
  const weaponAmmo = ammo[type]

  return (
    <GunElement
      disabled={weaponAmmo === 0}
      selected={weapon === type}
      onClick={selectWeapon}
    >
      <img src={icon} width={32} height={32} />
      <span>{weaponAmmo}</span>
    </GunElement>
  )
}

export default function MobileControls(props: {
  isRunning: boolean
  playerState: PlayerState
}) {
  const { isRunning, playerState } = props

  const containerRef = React.useRef<HTMLDivElement>(null)
  const leftRef = React.useRef<HTMLDivElement>(null)
  const machineRef = React.useRef<TouchMachine>(newTouchMachine())
  const [isInMenu, setIsInMenu] = React.useState<boolean>(true)
  const [shooting, setShooting] = React.useState<boolean>(false)
  const [zooming, setZooming] = React.useState<boolean>(false)

  const toggleMenu = React.useCallback(
    (event: React.MouseEvent) => {
      event.preventDefault()
      if (!isRunning) return
      BananaBread.execute('togglemainmenu')
    },
    [isRunning]
  )

  const trackTouch = React.useCallback(
    (event: React.TouchEvent, onEnd?: TouchAction) => {
      event.preventDefault()
      machineRef.current = handleTouchStart(
        machineRef.current,
        event.changedTouches,
        onEnd
      )
    },
    []
  )

  const trackTouchMove = React.useCallback(
    (event: React.TouchEvent) => {
      const [newMachine, motions] = handleTouchMove(
        machineRef.current,
        event.changedTouches
      )
      machineRef.current = newMachine
      if (motions.length == 0) return
      const [motion] = motions
      const [dx, dy] = motion
      if (isInMenu || (dx === 0 && dy === 0)) return
      BananaBread.mousemove(dx * MOTION_FACTOR, dy * MOTION_FACTOR)
    },
    [isInMenu]
  )

  const trackTouchEnd = React.useCallback((event: React.TouchEvent) => {
    machineRef.current = handleTouchEnd(
      machineRef.current,
      event.changedTouches
    )
  }, [])

  const selectWeapon = React.useCallback(
    (index: number) => {
      return () => {
        if (!isRunning) return
        BananaBread.execute(`weapon ${index}`)
      }
    },
    [isRunning]
  )

  const createAction = React.useCallback(
    (onStart: TouchAction, onEnd: TouchAction) => {
      return (event: React.TouchEvent) => {
        if (!isRunning) return
        trackTouch(event, onEnd)
        onStart()
      }
    },
    [isRunning]
  )

  const startShoot = React.useMemo(() => {
    return createAction(
      () => {
        BananaBread.execute(`_attack 1`)
        setShooting(true)
      },
      () => {
        BananaBread.execute(`_attack 0`)
        setShooting(false)
      }
    )
  }, [createAction])

  const startZoom = React.useMemo(() => {
    return createAction(
      () => {
        BananaBread.execute(`zoom 1`)
        setZooming(true)
      },
      () => {
        BananaBread.execute(`zoom -1`)
        setZooming(false)
      }
    )
  }, [createAction])

  const startJump = React.useMemo(() => {
    return createAction(
      () => BananaBread.execute(`_jump 1`),
      () => BananaBread.execute(`_jump 0`)
    )
  }, [createAction])

  React.useEffect(() => {
    if (!isRunning) return
    const { current: container } = containerRef
    const { current: leftPad } = leftRef
    if (container == null || leftPad == null) return

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

    let _isInMenu: boolean = true
    const cb = () => {
      window.requestAnimationFrame(cb)
      if (!Module.running) return
      const newInMenu = BananaBread.isInMenu() === 1
      if (!_isInMenu && newInMenu) {
        unregisterJoysticks()
        setIsInMenu(true)
      } else if (_isInMenu && !newInMenu) {
        registerJoysticks()
        setIsInMenu(false)
      }
      if (!newInMenu && movement == null) {
        registerJoysticks()
      }
      _isInMenu = newInMenu
    }
    window.requestAnimationFrame(cb)

    container.onpointerup = (evt) => {
      const { x, y, width, height } = container.getBoundingClientRect()
      const { x: mouseX, y: mouseY } = evt

      BananaBread.click((mouseX - x) / width, (mouseY - y) / height)
    }
  }, [isRunning])

  const controlsStyles = { opacity: isInMenu ? 0 : 1 }

  return (
    <Container
      ref={containerRef}
      onTouchMove={trackTouchMove}
      onTouchEnd={trackTouchEnd}
    >
      <TopLeftPanel>
        <Button onMouseDown={toggleMenu}>☰</Button>
      </TopLeftPanel>
      <BottomLeftPanel style={controlsStyles}>
        <ActionButton
          onTouchStart={startShoot}
          active={shooting}
          style={{
            position: 'absolute',
            bottom: 150,
            left: 60,
          }}
        >
          <span style={{ marginTop: -5 }}>⦿</span>
        </ActionButton>
      </BottomLeftPanel>
      <BottomRightPanel style={controlsStyles}>
        <ActionButton
          onTouchStart={startJump}
          style={{
            position: 'absolute',
            bottom: 30,
            right: 100,
          }}
        >
          <span style={{ marginTop: -5 }}>▲</span>
        </ActionButton>
        <ActionButton
          onTouchStart={startShoot}
          active={shooting}
          style={{
            position: 'absolute',
            width: 80,
            height: 80,
            bottom: 100,
            right: 100,
            fontSize: 40,
          }}
        >
          <span style={{ marginTop: -5 }}>⦿</span>
        </ActionButton>
        <ActionButton
          onTouchStart={startZoom}
          active={zooming}
          style={{
            position: 'absolute',
            bottom: 240,
            right: 100,
          }}
        >
          <span style={{ marginTop: -7 }}>⌖</span>
        </ActionButton>
        {WEAPON_INFO.map((v) => (
          <GunButton
            key={v.type}
            type={v.type}
            icon={v.icon}
            playerState={playerState}
            selectWeapon={selectWeapon(v.type)}
          />
        ))}
      </BottomRightPanel>
      <MovementPad ref={leftRef} />
      <DirectionPad onTouchStart={trackTouch} />
    </Container>
  )
}
