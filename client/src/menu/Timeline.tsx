import * as React from 'react'
import styled from '@emotion/styled'
import { css } from '@emotion/react'
import { t } from './theme'
import type { BrowseMapEntry } from '../catalog/types'

const Wrap = styled.div`
  flex: 0 0 auto;
  border-top: 1px solid ${t.line};
  background: ${t.surface};
  padding: 14px 32px 16px;
  font-family: ${t.fontMono};
  font-size: 10px;
  color: ${t.mute};
  position: relative;
`

const Ticks = styled.div`
  display: grid;
  grid-template-columns: repeat(16, 1fr);
  gap: 3px;
  align-items: end;
  height: 36px;
`

const Tick = styled.div<{ $active?: boolean; $hot?: boolean; $height: number }>`
  background: ${t.dim};
  border-radius: 2px;
  height: ${p => p.$height}%;
  min-height: 2px;
  cursor: pointer;

  ${p => p.$hot && css`background: ${t.accent2};`}
  ${p => p.$active && css`background: ${t.accent};`}
`

const Years = styled.div`
  display: grid;
  grid-template-columns: repeat(16, 1fr);
  gap: 3px;
  margin-top: 6px;
  font-size: 10px;
  color: ${t.mute};

  span { text-align: center; }
`

const ScrubberHead = styled.div<{ $pos: number }>`
  position: absolute;
  top: 8px;
  bottom: 8px;
  width: 2px;
  background: ${t.bone};
  z-index: 2;
  border-radius: 1px;
  left: ${p => p.$pos}%;

  &::before {
    content: '';
    position: absolute;
    left: -5px;
    top: -5px;
    width: 12px;
    height: 12px;
    background: ${t.accent};
    border: 2px solid ${t.bone};
    border-radius: 50%;
  }
`

type Props = {
  maps: BrowseMapEntry[]
  activeYear?: number
  onYearClick?: (year: number) => void
  startYear?: number
  endYear?: number
}

export default function Timeline({
  maps,
  activeYear,
  onYearClick,
  startYear = 2005,
  endYear = 2020,
}: Props) {
  const yearCount = endYear - startYear + 1

  const counts = React.useMemo(() => {
    const c = new Array(yearCount).fill(0)
    for (const m of maps) {
      if (!m.date) continue
      const y = parseInt(m.date.slice(0, 4), 10)
      if (isNaN(y)) continue
      const idx = y - startYear
      if (idx >= 0 && idx < yearCount) c[idx]++
    }
    return c
  }, [maps, startYear, endYear])

  const max = Math.max(...counts, 1)
  const activeIdx = activeYear != null ? activeYear - startYear : -1
  const scrubberPos = activeIdx >= 0
    ? ((activeIdx / (yearCount - 1)) * (100 - 8)) + 4  // account for padding
    : -1

  return (
    <Wrap>
      <Ticks>
        {counts.map((c, i) => (
          <Tick
            key={i}
            $height={(c / max) * 100}
            $active={i === activeIdx}
            $hot={Math.abs(i - activeIdx) <= 1 && i !== activeIdx}
            onClick={() => onYearClick?.(startYear + i)}
          />
        ))}
      </Ticks>
      <Years>
        {counts.map((_, i) => (
          <span key={i}>{String(startYear + i).slice(2)}</span>
        ))}
      </Years>
      {activeIdx >= 0 && (
        <ScrubberHead
          $pos={(activeIdx / (yearCount - 1)) * 100}
        />
      )}
    </Wrap>
  )
}
