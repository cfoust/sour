import * as React from 'react'
import styled from '@emotion/styled'
import { t } from '../theme'
import { Btn, Marker, SectionHeadWrap, SectionTitle, SectionMeta } from '../styled'
import Icon from '../Icon'
import MapCard from '../MapCard'
import Timeline from '../Timeline'
import type { BrowseMapEntry } from '../../catalog/types'

const Wrap = styled.div`
  display: flex;
  flex-direction: column;
  min-height: 0;
  flex: 1 1 auto;
  overflow-y: auto;
`

const BackBar = styled.div`
  font-family: ${t.fontDisplay};
  font-size: 13px;
  font-weight: 600;
  color: ${t.mute};
  padding: 14px 32px;
  border-bottom: 1px solid ${t.line};
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;

  &:hover { color: ${t.bone}; }
`

const Hero = styled.div`
  padding: 36px 40px 28px;
  border-bottom: 1px solid ${t.line};
  display: flex;
  gap: 32px;
  align-items: flex-end;
`

const Avatar = styled.div`
  width: 120px;
  height: 120px;
  background: ${t.accent};
  font-family: ${t.fontDisplay};
  font-weight: 900;
  font-size: 60px;
  color: ${t.bone};
  display: flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 auto;
  text-transform: uppercase;
  letter-spacing: -0.03em;
  border-radius: 28px;
  box-shadow: 0 6px 0 ${t.accent3};
`

const TextBlock = styled.div`
  display: flex;
  flex-direction: column;
  gap: 6px;
`

const Eyebrow = styled.div`
  font-family: ${t.fontDisplay};
  font-size: 13px;
  font-weight: 600;
  color: ${t.mute};
`

const AuthorH1 = styled.h1`
  font-family: ${t.fontDisplay};
  font-size: 64px;
  font-weight: 900;
  line-height: 1;
  letter-spacing: -0.04em;
  margin: 0;
  color: ${t.bone};
`

const YearsLine = styled.div`
  font-family: ${t.fontUi};
  font-size: 13px;
  color: ${t.bone2};
  margin-top: 4px;
`

const Stats = styled.div`
  display: flex;
  gap: 36px;
  margin-top: 10px;
`

const StatBlock = styled.div``

const StatValue = styled.div`
  font-family: ${t.fontDisplay};
  font-size: 26px;
  font-weight: 800;
  color: ${t.bone};
  letter-spacing: -0.02em;
`

const StatLabel = styled.div`
  font-family: ${t.fontUi};
  font-size: 12px;
  font-weight: 500;
  color: ${t.mute};
`

const GridPad = styled.div`
  padding: 0 40px 40px;
`

const MapGrid = styled.div`
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 26px 22px;

  @media (max-width: 900px) { grid-template-columns: repeat(3, 1fr); }
  @media (max-width: 600px) { grid-template-columns: repeat(2, 1fr); }
`

const TimelinePad = styled.div`
  padding: 0 40px 56px;
`

function yearFromDate(date?: string): number | null {
  if (!date) return null
  const y = parseInt(date.slice(0, 4), 10)
  return isNaN(y) ? null : y
}

type Props = {
  authorName: string
  allMaps: BrowseMapEntry[]
  onBack: () => void
  onOpenMap: (map: BrowseMapEntry) => void
  onPlayMap: (name: string) => void
}

export default function AuthorScreen({ authorName, allMaps, onBack, onOpenMap, onPlayMap }: Props) {
  const works = React.useMemo(
    () => allMaps.filter(m => m.author === authorName),
    [allMaps, authorName]
  )

  const years = React.useMemo(() => {
    const yrs = works
      .map(m => yearFromDate(m.date))
      .filter((y): y is number => y !== null)
      .sort()
    return [...new Set(yrs)]
  }, [works])

  const firstYear = years.length > 0 ? years[0] : null
  const lastYear = years.length > 0 ? years[years.length - 1] : null

  return (
    <Wrap>
      <BackBar onClick={onBack}>
        <Icon name="arrow-l" size={11} /> Back to archive
      </BackBar>

      <Hero>
        <Avatar>{authorName[0]?.toLowerCase() || '?'}</Avatar>
        <TextBlock>
          <Eyebrow>Cartographer</Eyebrow>
          <AuthorH1>{authorName}</AuthorH1>
          {firstYear && lastYear && (
            <YearsLine>
              Active · {firstYear} → {lastYear} · {years.length} year{years.length !== 1 ? 's' : ''}
            </YearsLine>
          )}
          <Stats>
            <StatBlock>
              <StatValue>{works.length}</StatValue>
              <StatLabel>Maps in archive</StatLabel>
            </StatBlock>
          </Stats>
        </TextBlock>
      </Hero>

      <SectionHeadWrap>
        <div>
          <SectionTitle>Complete works · chronological</SectionTitle>
          <SectionMeta>{works.length} maps</SectionMeta>
        </div>
      </SectionHeadWrap>
      <GridPad>
        <MapGrid>
          {works.map(m => (
            <MapCard
              key={m.name}
              map={m}
              onOpen={onOpenMap}
              onPlay={onPlayMap}
            />
          ))}
        </MapGrid>
      </GridPad>

      {years.length > 0 && (
        <>
          <SectionHeadWrap>
            <div>
              <SectionTitle>Career timeline</SectionTitle>
              <SectionMeta>Maps per year</SectionMeta>
            </div>
          </SectionHeadWrap>
          <TimelinePad>
            <Timeline
              maps={works}
              startYear={firstYear!}
              endYear={lastYear!}
            />
          </TimelinePad>
        </>
      )}
    </Wrap>
  )
}
