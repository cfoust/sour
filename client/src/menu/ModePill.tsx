import * as React from 'react'
import styled from '@emotion/styled'
import { css } from '@emotion/react'
import { t } from './theme'

const Pill = styled.span<{ $primary?: boolean }>`
  font-family: ${t.fontDisplay};
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0;
  text-transform: uppercase;
  border: 1px solid ${t.line2};
  color: ${t.bone2};
  padding: 4px 10px;
  border-radius: 99px;
  background: ${t.bgCard};

  ${p => p.$primary && css`
    background: ${t.accent};
    border-color: ${t.accent};
    color: ${t.bone};
    font-weight: 700;
  `}
`

const MODE_NAMES: Record<string, string> = {
  dm: 'DM',
  tdm: 'TDM',
  ctf: 'CTF',
  capture: 'CAP',
  insta: 'INST',
  efficiency: 'EFF',
  tactics: 'TAC',
  ffa: 'FFA',
  coop: 'COOP',
}

type Props = {
  mode: string
  primary?: boolean
}

export default function ModePill({ mode, primary }: Props) {
  return <Pill $primary={primary}>{MODE_NAMES[mode] || mode.toUpperCase()}</Pill>
}
