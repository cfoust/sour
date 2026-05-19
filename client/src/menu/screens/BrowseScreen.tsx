import * as React from 'react'
import styled from '@emotion/styled'
import { t } from '../theme'
import { Btn, Marker } from '../styled'
import Icon from '../Icon'
import MapCard from '../MapCard'
import FilterRail from '../FilterRail'
import Timeline from '../Timeline'
import { useFilters } from '../useFilters'
import { useSearch, SortField } from '../useSearch'
import type { BrowseMapEntry } from '../../catalog/types'

const PAGE_SIZE = 50

const Layout = styled.div`
  display: grid;
  grid-template-columns: 256px 1fr;
  flex: 1 1 auto;
  min-height: 0;

  @media (max-width: 768px) {
    grid-template-columns: 1fr;
  }
`

const Main = styled.div`
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-width: 0;
`

const Subhead = styled.div`
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  padding: 26px 32px 18px;
  border-bottom: 1px solid ${t.line};
`

const Title = styled.div`
  font-family: ${t.fontDisplay};
  font-size: 36px;
  font-weight: 800;
  letter-spacing: -0.025em;
  line-height: 1;
  color: ${t.bone};
`

const MetaStrip = styled.div`
  font-family: ${t.fontUi};
  font-size: 13px;
  color: ${t.mute};
  margin-top: 8px;
  display: flex;
  gap: 14px;

  span + span::before {
    content: '·';
    margin-right: 14px;
    color: ${t.dim};
  }
`

const Toolbar = styled.div`
  display: flex;
  align-items: center;
  gap: 10px;
`

const SortWrap = styled.div`
  display: flex;
  align-items: center;
  gap: 8px;
  font-family: ${t.fontUi};
  font-size: 13px;
  color: ${t.mute};
  font-weight: 500;
`

const SortSelect = styled.select`
  background: ${t.bgCard};
  color: ${t.bone};
  border: 1px solid ${t.line};
  border-radius: 8px;
  padding: 7px 10px;
  font-family: ${t.fontUi};
  font-size: 13px;
  cursor: pointer;
`

const ViewToggle = styled.div`
  display: inline-flex;
  border: 1px solid ${t.line};
  border-radius: 10px;
  overflow: hidden;
  background: ${t.bgCard};
`

const ViewBtn = styled.button<{ $active?: boolean }>`
  width: 34px;
  height: 32px;
  background: ${p => p.$active ? t.accent : 'transparent'};
  border: none;
  color: ${p => p.$active ? t.bone : t.mute};
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
`

const GridWrap = styled.div`
  overflow-y: auto;
  padding: 24px 32px 36px;
  flex: 1 1 auto;
`

const MapGrid = styled.div<{ $dense?: boolean }>`
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 26px 22px;

  ${p => p.$dense && `
    grid-template-columns: repeat(5, 1fr);
    gap: 18px 14px;
  `}

  @media (max-width: 1200px) { grid-template-columns: repeat(3, 1fr); }
  @media (max-width: 900px) { grid-template-columns: repeat(2, 1fr); }
`

const LoadMore = styled.div`
  display: flex;
  justify-content: center;
  padding: 24px 0 8px;
`

type Props = {
  maps: BrowseMapEntry[]
  searchQuery: string
  onOpenMap: (map: BrowseMapEntry) => void
  onPlayMap: (name: string) => void
}

export default function BrowseScreen({ maps, searchQuery, onOpenMap, onPlayMap }: Props) {
  const { filters, filtered, yearRange, mapsWithScreenshots, modeCounts, setYear, toggleHasScreenshot, toggleMode } = useFilters(maps)
  const { query, setQuery, sortBy, setSortBy, results } = useSearch(filtered)
  const [shown, setShown] = React.useState(PAGE_SIZE)

  // Sync external search query into local search
  React.useEffect(() => {
    if (searchQuery !== query) {
      setQuery(searchQuery)
    }
  }, [searchQuery])

  React.useEffect(() => {
    setShown(PAGE_SIZE)
  }, [query, sortBy, filters])

  const visible = results.slice(0, shown)
  const hasMore = shown < results.length
  const authors = React.useMemo(
    () => new Set(maps.map(m => m.author).filter(Boolean)).size,
    [maps]
  )

  const handleRandom = () => {
    if (results.length === 0) return
    const map = results[Math.floor(Math.random() * results.length)]
    onOpenMap(map)
  }

  return (
    <Layout>
      <FilterRail
        filters={filters}
        onToggleHasScreenshot={toggleHasScreenshot}
        onToggleMode={toggleMode}
        onSetYear={setYear}
        yearRange={yearRange}
        totalMaps={maps.length}
        mapsWithScreenshots={mapsWithScreenshots}
        modeCounts={modeCounts}
      />
      <Main>
        <Subhead>
          <div>
            <Title>All <Marker>Maps</Marker></Title>
            <MetaStrip>
              <span>{results.length.toLocaleString()} maps</span>
              <span>{authors} authors</span>
              <span>{mapsWithScreenshots.toLocaleString()} with screenshots</span>
            </MetaStrip>
          </div>
          <Toolbar>
            <Btn $ghost onClick={handleRandom}>
              <Icon name="shuffle" size={12} /> Random
            </Btn>
            <SortWrap>
              Sort
              <SortSelect
                value={sortBy}
                onChange={e => setSortBy(e.target.value as SortField)}
              >
                <option value="name">A → Z</option>
                <option value="date">Newest</option>
              </SortSelect>
            </SortWrap>
            <ViewToggle>
              <ViewBtn $active><Icon name="grid" size={13} /></ViewBtn>
              <ViewBtn><Icon name="list" size={13} /></ViewBtn>
            </ViewToggle>
          </Toolbar>
        </Subhead>
        <GridWrap>
          <MapGrid>
            {visible.map(m => (
              <MapCard
                key={m.name}
                map={m}
                onOpen={onOpenMap}
                onPlay={onPlayMap}
              />
            ))}
          </MapGrid>
          {hasMore && (
            <LoadMore>
              <Btn onClick={() => setShown(s => s + PAGE_SIZE)}>
                Load more ({(results.length - shown).toLocaleString()} remaining)
              </Btn>
            </LoadMore>
          )}
        </GridWrap>
        <Timeline
          maps={maps}
          activeYear={filters.year < yearRange[1] ? filters.year : undefined}
          onYearClick={setYear}
          startYear={yearRange[0]}
          endYear={yearRange[1]}
        />
      </Main>
    </Layout>
  )
}
