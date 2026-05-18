import * as React from 'react'
import styled from '@emotion/styled'
import { css } from '@emotion/react'
import { t } from './theme'
import { Mono } from './styled'

const Wrap = styled.div`
  display: flex;
  flex-direction: column;
  gap: 4px;
`

const ScoreRow = styled.div<{ $you?: boolean }>`
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 14px;
  background: ${t.surface};
  border-left: 3px solid transparent;
  border-radius: 10px;

  ${p => p.$you && css`
    border-left-color: ${t.accent};
    background: ${t.softYel};
  `}
`

const Frag = styled.span`
  font-family: ${t.fontMono};
  font-size: 15px;
  color: ${t.bone};
  font-weight: 600;
  width: 32px;
  text-align: right;
`

const Name = styled.span`
  flex: 1;
  font-family: ${t.fontDisplay};
  font-size: 14.5px;
  font-weight: 600;
  letter-spacing: -0.005em;
  color: ${t.bone};
`

export type PlayerEntry = {
  name: string
  frags: number
  isYou?: boolean
}

type Props = {
  players: PlayerEntry[]
}

export default function Scoreboard({ players }: Props) {
  return (
    <Wrap>
      {players.map((p, i) => (
        <ScoreRow key={i} $you={p.isYou}>
          <Frag>{p.frags}</Frag>
          <Name>{p.name}</Name>
        </ScoreRow>
      ))}
    </Wrap>
  )
}
