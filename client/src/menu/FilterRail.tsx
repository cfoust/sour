import * as React from 'react'
import styled from '@emotion/styled'
import { css } from '@emotion/react'
import { t } from './theme'
import { FilterLabel, Mono } from './styled'
import type { Filters, ModeCount } from './useFilters'

const Rail = styled.aside`
  border-right: 1px solid ${t.line};
  padding: 24px 22px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 24px;
  background: rgba(255, 255, 255, 0.4);
  width: 256px;
  flex: 0 0 auto;
`

const Group = styled.div`
  display: flex;
  flex-direction: column;
  gap: 6px;
`

const Row = styled.div<{ $active?: boolean }>`
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 8px;
  font-size: 13.5px;
  color: ${t.bone2};
  cursor: pointer;
  user-select: none;
  border-radius: 8px;

  &:hover { background: rgba(40, 28, 8, 0.04); color: ${t.bone}; }

  ${p => p.$active && css`
    color: ${t.bone};
    background: ${t.softYel};
  `}
`

const RowLeft = styled.div`
  display: flex;
  align-items: center;
`

const Check = styled.span<{ $active?: boolean }>`
  width: 16px;
  height: 16px;
  border: 1.5px solid ${t.line3};
  border-radius: 5px;
  margin-right: 10px;
  flex: 0 0 auto;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: ${t.bgCard};

  ${p => p.$active && css`
    background: ${t.accent};
    border-color: ${t.accent};

    &::after {
      content: '';
      width: 7px;
      height: 3.5px;
      border-left: 2px solid ${t.bone};
      border-bottom: 2px solid ${t.bone};
      transform: rotate(-45deg) translate(0, -1px);
    }
  `}
`

const Num = styled.span`
  font-family: ${t.fontMono};
  font-size: 11px;
  color: ${t.mute};
`

// Year slider
const SliderWrap = styled.div`
  position: relative;
  height: 36px;
  margin: 8px 0 0;
`

const Track = styled.div`
  position: absolute;
  left: 0;
  right: 0;
  top: 14px;
  height: 4px;
  background: ${t.surface2};
  border-radius: 2px;
`

const ActiveTrack = styled.div<{ $left: number; $right: number }>`
  position: absolute;
  top: 14px;
  height: 4px;
  background: ${t.accent};
  border-radius: 2px;
  left: ${p => p.$left}%;
  right: ${p => p.$right}%;
`

const Handle = styled.div<{ $pos: number }>`
  position: absolute;
  top: 8px;
  width: 16px;
  height: 16px;
  background: ${t.accent};
  border: 2px solid ${t.bone};
  border-radius: 50%;
  transform: translateX(-50%);
  box-shadow: 0 2px 0 ${t.accent3};
  left: ${p => p.$pos}%;
  cursor: grab;
  z-index: 1;
`

const YearLabels = styled.div`
  display: flex;
  justify-content: space-between;
  font-family: ${t.fontMono};
  font-size: 11px;
  color: ${t.mute};
  margin-top: 10px;
`

const YearValue = styled.span`
  font-family: ${t.fontMono};
  font-size: 11px;
  color: ${t.bone};
`

const MODE_LABELS: Record<string, string> = {
  ffa: 'Free for All',
  tdm: 'Team DM',
  ctf: 'Capture the Flag',
  capture: 'Capture',
  insta: 'Instagib',
  effic: 'Efficiency',
  tac: 'Tactics',
  coop: 'Co-op',
}

type Props = {
  filters: Filters
  onToggleHasScreenshot: () => void
  onToggleMode: (mode: string) => void
  onSetYear: (year: number) => void
  yearRange: [number, number]
  totalMaps: number
  mapsWithScreenshots: number
  modeCounts: ModeCount[]
}

export default function FilterRail({
  filters,
  onToggleHasScreenshot,
  onToggleMode,
  onSetYear,
  yearRange,
  totalMaps,
  mapsWithScreenshots,
  modeCounts,
}: Props) {
  const [minYear, maxYear] = yearRange
  const yearSpan = maxYear - minYear || 1
  const yearPos = ((filters.year - minYear) / yearSpan) * 100

  const handleSliderClick = (e: React.MouseEvent<HTMLDivElement>) => {
    const rect = e.currentTarget.getBoundingClientRect()
    const pct = (e.clientX - rect.left) / rect.width
    const year = Math.round(minYear + pct * yearSpan)
    onSetYear(Math.max(minYear, Math.min(maxYear, year)))
  }

  return (
    <Rail>
      {modeCounts.length > 0 && (
        <Group>
          <FilterLabel>Game mode</FilterLabel>
          {modeCounts.map(mc => {
            const active = filters.activeModes.has(mc.mode)
            return (
              <Row key={mc.mode} $active={active} onClick={() => onToggleMode(mc.mode)}>
                <RowLeft>
                  <Check $active={active} />
                  {MODE_LABELS[mc.mode] || mc.mode}
                </RowLeft>
                <Num>{mc.count.toLocaleString()}</Num>
              </Row>
            )
          })}
        </Group>
      )}

      <Group>
        <FilterLabel>Year</FilterLabel>
        <SliderWrap onClick={handleSliderClick}>
          <Track />
          <ActiveTrack $left={0} $right={100 - yearPos} />
          <Handle $pos={yearPos} />
        </SliderWrap>
        <YearLabels>
          <span>{minYear}</span>
          <YearValue>{filters.year}</YearValue>
          <span>{maxYear}</span>
        </YearLabels>
      </Group>

      <Group>
        <FilterLabel>Has preview</FilterLabel>
        <Row
          $active={filters.hasScreenshot}
          onClick={onToggleHasScreenshot}
        >
          <RowLeft>
            <Check $active={filters.hasScreenshot} />
            Screenshot
          </RowLeft>
          <Num>{mapsWithScreenshots.toLocaleString()}</Num>
        </Row>
      </Group>
    </Rail>
  )
}
