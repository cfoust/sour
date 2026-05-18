import * as React from 'react'
import styled from '@emotion/styled'
import { t } from './theme'
import Icon from './Icon'
import MapThumb from './MapThumb'
import ModePill from './ModePill'
import { Mono } from './styled'
import type { BrowseMapEntry } from '../catalog/types'

const Card = styled.div`
  display: flex;
  flex-direction: column;
  cursor: pointer;
  gap: 12px;

  &:hover .map-card-thumb {
    transform: translateY(-3px);
    box-shadow: ${t.cardHoverShadow};
  }

  &:hover .map-card-play {
    opacity: 1;
    transform: scale(1);
  }
`

const Thumb = styled.div`
  aspect-ratio: 4 / 3;
  background: ${t.surface};
  overflow: hidden;
  position: relative;
  border-radius: 14px;
  box-shadow: ${t.cardShadow};
  transition: transform 0.12s, box-shadow 0.12s;
`

const Scrim = styled.div`
  position: absolute;
  inset: 0;
  background: linear-gradient(180deg, transparent 55%, rgba(0, 0, 0, 0.55) 100%);
  z-index: 1;
`

const YearBadge = styled.span`
  position: absolute;
  top: 10px;
  right: 10px;
  font-family: ${t.fontMono};
  font-size: 11px;
  color: ${t.bone};
  background: rgba(255, 255, 255, 0.9);
  padding: 3px 8px;
  border-radius: 8px;
  z-index: 2;
`

const Modes = styled.div`
  position: absolute;
  left: 10px;
  bottom: 10px;
  display: flex;
  gap: 5px;
  z-index: 2;
`

const PlayBtn = styled.button`
  position: absolute;
  right: 12px;
  bottom: 12px;
  width: 44px;
  height: 44px;
  border-radius: 50%;
  background: ${t.accent};
  border: none;
  color: ${t.bone};
  display: flex;
  align-items: center;
  justify-content: center;
  opacity: 0;
  transform: scale(0.7);
  transition: opacity 0.15s, transform 0.15s;
  box-shadow: ${t.btnPrimaryShadow};
  z-index: 4;
  cursor: pointer;
`

const Meta = styled.div`
  display: flex;
  flex-direction: column;
  gap: 3px;
`

const Title = styled.div`
  font-family: ${t.fontDisplay};
  font-size: 16px;
  font-weight: 700;
  letter-spacing: -0.01em;
  color: ${t.bone};
  line-height: 1.2;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
`

const Author = styled.div`
  font-family: ${t.fontUi};
  font-size: 13px;
  color: ${t.mute};

  em {
    font-style: normal;
    color: ${t.bone2};
    font-weight: 500;
  }
`

function yearFromDate(date?: string): string | null {
  if (!date) return null
  const y = date.slice(0, 4)
  return /^\d{4}$/.test(y) ? y : null
}

type Props = {
  map: BrowseMapEntry
  onOpen?: (map: BrowseMapEntry) => void
  onPlay?: (mapName: string) => void
}

export default function MapCard({ map, onOpen, onPlay }: Props) {
  const year = yearFromDate(map.date)

  return (
    <Card onClick={() => onOpen?.(map)}>
      <Thumb className="map-card-thumb">
        <MapThumb imageUrl={map.imageUrl} name={map.name} />
        <Scrim />
        {year && <YearBadge>{year}</YearBadge>}
        <PlayBtn
          className="map-card-play"
          onClick={e => {
            e.stopPropagation()
            onPlay?.(map.name)
          }}
        >
          <Icon name="play" size={12} />
        </PlayBtn>
      </Thumb>
      <Meta>
        <Title>{map.name}</Title>
        <Author>
          {map.author && <>by <em>{map.author}</em></>}
        </Author>
      </Meta>
    </Card>
  )
}
