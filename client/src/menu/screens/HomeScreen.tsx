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
import ServerCard from '../ServerCard'
import type { ServerEntry } from '../ServerCard'
import type { BrowseMapEntry } from '../../catalog/types'
import type { ClusterServerInfo } from '../../protocol'

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

const ServerCards = styled.div`
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;

  @media (max-width: 768px) { grid-template-columns: 1fr; }
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
  servers: ClusterServerInfo[]
  onOpenMap: (map: BrowseMapEntry) => void
  onPlayMap: (name: string) => void
  onSeeAll: () => void
  onJoinServer: (serverName: string) => void
}

export default function HomeScreen({ maps, servers, onOpenMap, onPlayMap, onSeeAll, onJoinServer }: Props) {
  const popular = React.useMemo(
    () => [...maps].slice(0, 8),
    [maps]
  )

  const totalPlayers = servers.reduce((sum, s) => sum + s.CurrentPlayers, 0)

  const serverEntries: ServerEntry[] = React.useMemo(
    () => servers.map(s => ({
      name: s.Name,
      mapName: s.Map,
      mode: s.Mode?.toUpperCase(),
      currentPlayers: s.CurrentPlayers,
      maxPlayers: s.MaxPlayers,
      hot: s.MaxPlayers > 0 && s.CurrentPlayers / s.MaxPlayers >= 0.75,
    })),
    [servers]
  )

  return (
    <Scroll>
      {/* Servers section */}
      <SectionHeadWrap>
        <div>
          <SectionTitle>
            Servers
            {totalPlayers > 0 && <CountPill>{totalPlayers} playing</CountPill>}
          </SectionTitle>
          <SectionMeta>Pick one to jump in.</SectionMeta>
        </div>
      </SectionHeadWrap>
      <Stack>
        {serverEntries.length > 0 ? (
          <ServerCards>
            {serverEntries.map((s, i) => (
              <ServerCard key={i} server={s} onJoin={() => onJoinServer(s.name)} />
            ))}
          </ServerCards>
        ) : (
          <EmptyState>
            <EmptyTitle>No servers running</EmptyTitle>
            Browse the archive to find maps and start playing.
          </EmptyState>
        )}
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
