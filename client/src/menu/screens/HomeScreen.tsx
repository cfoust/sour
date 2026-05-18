import * as React from 'react'
import styled from '@emotion/styled'
import { t } from '../theme'
import {
  SectionHeadWrap,
  SectionTitle,
  CountPill,
  SectionMeta,
  SectionLink,
} from '../styled'
import MapCard from '../MapCard'
import Icon from '../Icon'
import type { BrowseMapEntry } from '../../catalog/types'

const Scroll = styled.div`
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  overflow-y: auto;
`

const Stack = styled.div`
  padding: 0 32px 40px;
`

const MapGrid = styled.div`
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 26px 22px;

  @media (max-width: 900px) { grid-template-columns: repeat(3, 1fr); }
  @media (max-width: 600px) { grid-template-columns: repeat(2, 1fr); }
`

const EmptyState = styled.div`
  padding: 48px 32px;
  text-align: center;
  color: ${t.mute};
  font-size: 14px;
  background: ${t.surface};
  border-radius: 16px;
  border: 1px dashed ${t.line2};
`

const EmptyTitle = styled.div`
  font-family: ${t.fontDisplay};
  font-size: 16px;
  font-weight: 700;
  color: ${t.bone2};
  margin-bottom: 8px;
`

type Props = {
  maps: BrowseMapEntry[]
  onOpenMap: (map: BrowseMapEntry) => void
  onPlayMap: (name: string) => void
  onSeeAll: () => void
}

export default function HomeScreen({ maps, onOpenMap, onPlayMap, onSeeAll }: Props) {
  const popular = React.useMemo(
    () => [...maps].slice(0, 8),
    [maps]
  )

  return (
    <Scroll>
      {/* Servers section — stubbed */}
      <SectionHeadWrap>
        <div>
          <SectionTitle>Servers</SectionTitle>
          <SectionMeta>Pick one to jump in.</SectionMeta>
        </div>
      </SectionHeadWrap>
      <Stack>
        <EmptyState>
          <EmptyTitle>Server list coming soon</EmptyTitle>
          Browse the archive to find maps and start playing.
        </EmptyState>
      </Stack>

      {/* Popular maps */}
      <SectionHeadWrap>
        <div>
          <SectionTitle>Popular maps</SectionTitle>
          <SectionMeta>From the archive.</SectionMeta>
        </div>
        <SectionLink onClick={onSeeAll}>
          See all {maps.length.toLocaleString()} →
        </SectionLink>
      </SectionHeadWrap>
      <Stack>
        <MapGrid>
          {popular.map(m => (
            <MapCard
              key={m.name}
              map={m}
              onOpen={onOpenMap}
              onPlay={onPlayMap}
            />
          ))}
        </MapGrid>
      </Stack>
    </Scroll>
  )
}
