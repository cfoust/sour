import * as React from 'react'
import styled from '@emotion/styled'
import { t } from './theme'

const Wrap = styled.span`
  display: flex;
  align-items: center;
  gap: 8px;
  font-family: ${t.fontUi};
  font-size: 14px;
  font-weight: 500;
  color: ${t.bone};
`

const Stars = styled.span`
  letter-spacing: 0.04em;
  color: ${t.accent3};
`

const Empty = styled.span`
  color: ${t.dim};
`

const Value = styled.span`
  font-family: ${t.fontMono};
`

type Props = {
  value: number
}

export default function Rating({ value }: Props) {
  const full = Math.round(value)
  return (
    <Wrap>
      <Stars>
        {'★'.repeat(full)}
        <Empty>{'★'.repeat(5 - full)}</Empty>
      </Stars>
      <Value>{value.toFixed(1)}</Value>
    </Wrap>
  )
}
