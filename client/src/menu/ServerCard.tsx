import * as React from 'react'
import styled from '@emotion/styled'
import { css } from '@emotion/react'
import { t } from './theme'
import MapThumb from './MapThumb'

const Card = styled.div<{ $full?: boolean }>`
  display: grid;
  grid-template-columns: 72px 1fr auto auto;
  gap: 16px;
  align-items: center;
  background: ${t.bgCard};
  padding: 14px;
  border-radius: 16px;
  cursor: pointer;
  transition: transform 0.12s, box-shadow 0.12s;
  border: 1px solid ${t.line};
  box-shadow: ${t.cardShadow};

  &:hover {
    transform: translateY(-2px);
    box-shadow: 0 6px 0 rgba(40, 28, 8, 0.07);
    border-color: ${t.accent};
  }
`

const Thumb = styled.div`
  width: 72px;
  height: 56px;
  border-radius: 10px;
  overflow: hidden;
  position: relative;
  flex: 0 0 auto;
`

const Info = styled.div`
  min-width: 0;
`

const SName = styled.div`
  font-family: ${t.fontDisplay};
  font-size: 17px;
  font-weight: 700;
  letter-spacing: -0.015em;
  color: ${t.bone};
  display: flex;
  align-items: center;
  gap: 8px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
`

const HotTag = styled.span`
  font-family: ${t.fontDisplay};
  font-size: 10px;
  font-weight: 700;
  background: ${t.accent};
  color: ${t.bone};
  padding: 2px 7px;
  border-radius: 99px;
  letter-spacing: 0.02em;
`

const SLine = styled.div`
  font-family: ${t.fontUi};
  font-size: 13px;
  color: ${t.mute};
  margin-top: 2px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;

  strong { color: ${t.bone2}; font-weight: 600; }
`

const Players = styled.div`
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 4px;
  min-width: 64px;
`

const PCount = styled.div`
  font-family: ${t.fontDisplay};
  font-size: 18px;
  font-weight: 800;
  color: ${t.bone};
  letter-spacing: -0.02em;
  line-height: 1;

  em {
    color: ${t.mute};
    font-style: normal;
    font-weight: 500;
    font-size: 12px;
  }
`

const JoinBtn = styled.button<{ $full?: boolean }>`
  font-family: ${t.fontDisplay};
  font-size: 13px;
  font-weight: 700;
  padding: 9px 16px;
  background: ${t.accent};
  color: ${t.bone};
  border: none;
  border-radius: 10px;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 5px;
  box-shadow: 0 3px 0 ${t.accent3};
  transition: transform 0.08s, box-shadow 0.08s;

  &:hover { background: ${t.accent2}; }
  &:active { transform: translateY(2px); box-shadow: 0 1px 0 ${t.accent3}; }

  ${p => p.$full && css`
    background: ${t.surface2};
    color: ${t.mute};
    box-shadow: 0 3px 0 ${t.dim};
    cursor: default;
  `}
`

export type ServerEntry = {
  name: string
  mapName?: string
  mapImageUrl?: string
  mode?: string
  region?: string
  currentPlayers?: number
  maxPlayers?: number
  hot?: boolean
}

type Props = {
  server: ServerEntry
  onJoin?: () => void
}

export default function ServerCard({ server, onJoin }: Props) {
  const isFull = server.currentPlayers != null &&
    server.maxPlayers != null &&
    server.currentPlayers >= server.maxPlayers

  return (
    <Card onClick={() => !isFull && onJoin?.()}>
      <Thumb>
        <MapThumb imageUrl={server.mapImageUrl} name={server.mapName || ''} />
      </Thumb>
      <Info>
        <SName>
          {server.name}
          {server.hot && <HotTag>HOT</HotTag>}
        </SName>
        <SLine>
          {server.mapName && <strong>{server.mapName}</strong>}
          {server.mode && <> · {server.mode}</>}
          {server.region && <> · {server.region}</>}
        </SLine>
      </Info>
      <Players>
        {server.currentPlayers != null && (
          <PCount>
            {server.currentPlayers}
            <em>/{server.maxPlayers}</em>
          </PCount>
        )}
      </Players>
      <JoinBtn $full={isFull} onClick={e => { e.stopPropagation(); !isFull && onJoin?.() }}>
        {isFull ? 'Full' : 'Join'}
      </JoinBtn>
    </Card>
  )
}
