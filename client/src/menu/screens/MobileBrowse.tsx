import * as React from 'react'
import styled from '@emotion/styled'
import { css } from '@emotion/react'
import { t } from '../theme'
import { Marker } from '../styled'
import Icon from '../Icon'
import MapCard from '../MapCard'
import { useSearch } from '../useSearch'
import type { BrowseMapEntry } from '../../catalog/types'
import type { MenuTab } from '../TopBar'

const PAGE_SIZE = 20

const Frame = styled.div`
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  background: ${t.bg};
  color: ${t.bone};
  font-family: ${t.fontUi};
  position: relative;
  overflow: hidden;
`

const TabBar = styled.div`
  display: flex;
  padding: 8px 14px;
  gap: 6px;
  border-bottom: 1px solid ${t.line};
  overflow-x: auto;
  flex: 0 0 auto;

  &::-webkit-scrollbar { display: none; }
`

const MobileTab = styled.button<{ $active?: boolean }>`
  flex: 0 0 auto;
  font-family: ${t.fontDisplay};
  font-size: 13px;
  font-weight: 600;
  letter-spacing: -0.005em;
  color: ${t.mute};
  padding: 8px 14px;
  border: none;
  border-radius: 10px;
  background: transparent;
  cursor: pointer;

  ${p => p.$active && css`
    color: ${t.bone};
    background: ${t.accent};
    box-shadow: 0 2px 0 ${t.accent3};
  `}
`

const Hero = styled.div`
  padding: 22px 18px 12px;
  display: flex;
  flex-direction: column;
  gap: 4px;
  flex: 0 0 auto;
`

const Eyebrow = styled.div`
  font-family: ${t.fontDisplay};
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: ${t.mute};
`

const MobileH1 = styled.h1`
  font-family: ${t.fontDisplay};
  font-size: 36px;
  font-weight: 900;
  letter-spacing: -0.03em;
  line-height: 1;
  margin: 0;
  color: ${t.bone};
`

const MobileMeta = styled.div`
  font-family: ${t.fontUi};
  font-size: 13px;
  color: ${t.mute};
  margin-top: 6px;
`

const SearchBox = styled.div`
  margin: 4px 16px 10px;
  padding: 12px 14px;
  background: ${t.bgCard};
  border: 1px solid ${t.line};
  border-radius: 12px;
  display: flex;
  align-items: center;
  gap: 10px;
  font-family: ${t.fontUi};
  font-size: 14px;
  color: ${t.mute};
  flex: 0 0 auto;
`

const SearchInput = styled.input`
  flex: 1;
  background: none;
  border: none;
  outline: none;
  font-family: inherit;
  font-size: inherit;
  color: ${t.bone};

  &::placeholder { color: ${t.mute}; }
`

const FilterChips = styled.div`
  display: flex;
  gap: 6px;
  padding: 4px 16px 14px;
  overflow-x: auto;
  flex: 0 0 auto;

  &::-webkit-scrollbar { display: none; }
`

const Chip = styled.button<{ $active?: boolean }>`
  flex: 0 0 auto;
  font-family: ${t.fontDisplay};
  font-size: 12px;
  font-weight: 600;
  padding: 6px 12px;
  border: 1px solid ${t.line};
  border-radius: 99px;
  color: ${t.bone2};
  background: ${t.bgCard};
  cursor: pointer;

  ${p => p.$active && css`
    background: ${t.accent};
    border-color: ${t.accent};
    color: ${t.bone};
  `}
`

const Grid = styled.div`
  flex: 1 1 auto;
  overflow-y: auto;
  padding: 0 16px 20px;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 18px 12px;
  align-content: start;
`

const LoadMore = styled.div`
  grid-column: 1 / -1;
  display: flex;
  justify-content: center;
  padding: 12px 0;
`

const LoadMoreBtn = styled.button`
  font-family: ${t.fontDisplay};
  font-size: 13px;
  font-weight: 600;
  padding: 10px 20px;
  border: 1px solid ${t.line2};
  background: ${t.bgCard};
  color: ${t.bone};
  border-radius: 12px;
  cursor: pointer;
`

const BottomBar = styled.div`
  height: 64px;
  border-top: 1px solid ${t.line};
  display: flex;
  align-items: stretch;
  justify-content: space-around;
  padding-bottom: 4px;
  background: ${t.bg};
  flex: 0 0 auto;
`

const BottomTab = styled.button<{ $active?: boolean }>`
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  font-family: ${t.fontDisplay};
  font-size: 10.5px;
  font-weight: 600;
  color: ${p => p.$active ? t.bone : t.mute};
  border: none;
  background: transparent;
  cursor: pointer;
  padding: 0;
`

const TABS: { id: MenuTab; name: string }[] = [
  { id: 'browse', name: 'Archive' },
  { id: 'home', name: 'Home' },
  { id: 'servers', name: 'Servers' },
  { id: 'settings', name: 'Settings' },
]

const TAB_ICONS: Record<string, 'archive' | 'home' | 'globe' | 'settings'> = {
  browse: 'archive',
  home: 'home',
  servers: 'globe',
  settings: 'settings',
}

type Props = {
  maps: BrowseMapEntry[]
  activeTab: MenuTab
  onTab: (tab: MenuTab) => void
  onOpenMap: (map: BrowseMapEntry) => void
  onPlayMap: (name: string) => void
  searchQuery: string
  onSearch: (q: string) => void
}

export default function MobileBrowse({
  maps, activeTab, onTab, onOpenMap, onPlayMap, searchQuery, onSearch,
}: Props) {
  const { query, setQuery, results } = useSearch(maps)
  const [shown, setShown] = React.useState(PAGE_SIZE)
  const mapsWithScreenshots = React.useMemo(
    () => maps.filter(m => !!m.imageUrl).length,
    [maps]
  )
  const authors = React.useMemo(
    () => new Set(maps.map(m => m.author).filter(Boolean)).size,
    [maps]
  )

  React.useEffect(() => {
    if (searchQuery !== query) setQuery(searchQuery)
  }, [searchQuery])

  React.useEffect(() => {
    setShown(PAGE_SIZE)
  }, [query])

  const visible = results.slice(0, shown)
  const hasMore = shown < results.length

  return (
    <Frame>
      <TabBar>
        {TABS.map(tab => (
          <MobileTab key={tab.id} $active={activeTab === tab.id} onClick={() => onTab(tab.id)}>
            {tab.name}
          </MobileTab>
        ))}
      </TabBar>

      <Hero>
        <Eyebrow>The Archive</Eyebrow>
        <MobileH1>{maps.length.toLocaleString()} <Marker>maps</Marker></MobileH1>
        <MobileMeta>{authors} authors · {mapsWithScreenshots.toLocaleString()} with screenshots</MobileMeta>
      </Hero>

      <SearchBox>
        <Icon name="search" size={12} />
        <SearchInput
          placeholder="Search maps, authors…"
          value={query}
          onChange={e => {
            setQuery(e.target.value)
            onSearch(e.target.value)
          }}
        />
      </SearchBox>

      <FilterChips>
        <Chip $active>All</Chip>
      </FilterChips>

      <Grid>
        {visible.map(m => (
          <MapCard
            key={m.name}
            map={m}
            onOpen={onOpenMap}
            onPlay={onPlayMap}
          />
        ))}
        {hasMore && (
          <LoadMore>
            <LoadMoreBtn onClick={() => setShown(s => s + PAGE_SIZE)}>
              Load more
            </LoadMoreBtn>
          </LoadMore>
        )}
      </Grid>

      <BottomBar>
        {TABS.map(tab => (
          <BottomTab key={tab.id} $active={activeTab === tab.id} onClick={() => onTab(tab.id)}>
            <Icon name={TAB_ICONS[tab.id]} size={18} />
            {tab.name}
          </BottomTab>
        ))}
      </BottomBar>
    </Frame>
  )
}
