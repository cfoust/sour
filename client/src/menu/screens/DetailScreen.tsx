import * as React from 'react'
import styled from '@emotion/styled'
import { t } from '../theme'
import { Btn, BtnIcon, Marker, FilterLabel, MetaSection, Mono } from '../styled'
import Icon from '../Icon'
import MapThumb from '../MapThumb'
import ModePill from '../ModePill'
import MapCard from '../MapCard'
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

const HeroImg = styled.div`
  height: 420px;
  position: relative;
  margin: 24px 32px;
  border-radius: 18px;
  overflow: hidden;
  flex: 0 0 auto;
  box-shadow: 0 4px 0 rgba(40, 28, 8, 0.06);
  background:
    radial-gradient(circle at 60% 40%, rgba(180, 100, 60, 0.3), transparent 60%),
    linear-gradient(135deg, #3a2818 0%, #1a1612 100%);
`

const HeroOverlay = styled.div`
  position: absolute;
  inset: 0;
  background:
    repeating-linear-gradient(0deg, rgba(255, 255, 255, 0.025) 0 1px, transparent 1px 3px),
    linear-gradient(135deg, rgba(0, 0, 0, 0) 0%, rgba(0, 0, 0, 0.35) 100%);
`

const HeroLabel = styled.div`
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  font-family: ${t.fontMono};
  font-size: 11px;
  letter-spacing: 0.08em;
  color: rgba(255, 255, 255, 0.5);
  text-transform: lowercase;
  z-index: 1;
`

const HeroImage = styled.img`
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
  z-index: 0;
`

const HeroStamps = styled.div`
  position: absolute;
  top: 20px;
  left: 32px;
  display: flex;
  gap: 18px;
  align-items: center;
  font-family: ${t.fontMono};
  font-size: 10px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: rgba(236, 228, 214, 0.55);
  z-index: 2;
`

const Body = styled.div`
  display: grid;
  grid-template-columns: 1fr 320px;
  gap: 48px;
  padding: 16px 40px 60px;

  @media (max-width: 900px) {
    grid-template-columns: 1fr;
  }
`

const Left = styled.div``

const DetailH1 = styled.h1`
  font-family: ${t.fontDisplay};
  font-size: 72px;
  font-weight: 900;
  line-height: 0.95;
  letter-spacing: -0.045em;
  margin: 0 0 14px;
  color: ${t.bone};

  em {
    font-style: normal;
    background: ${t.accent};
    color: ${t.bone};
    padding: 0 12px;
    border-radius: 12px;
  }

  @media (max-width: 768px) {
    font-size: 38px;
  }
`

const Credits = styled.div`
  font-family: ${t.fontUi};
  font-size: 16px;
  color: ${t.bone2};
  margin-bottom: 28px;
  display: flex;
  align-items: center;
  gap: 10px;
  font-weight: 500;
`

const AuthName = styled.span`
  color: ${t.bone};
  font-weight: 700;
  cursor: pointer;
  &:hover { text-decoration: underline; }
`

const Year = styled.span`
  font-family: ${t.fontMono};
  font-size: 13px;
  color: ${t.mute};
`

const PlayRow = styled.div`
  display: flex;
  gap: 10px;
  align-items: center;
  margin-bottom: 32px;
  flex-wrap: wrap;
`

const Descr = styled.div`
  font-family: ${t.fontUi};
  font-size: 15.5px;
  line-height: 1.7;
  color: ${t.bone2};
  margin-bottom: 30px;
  max-width: 64ch;

  p { margin: 0 0 1em; }
  p:last-child { margin-bottom: 0; }
`

const Sidebar = styled.aside`
  font-family: ${t.fontUi};
  font-size: 13px;
`

const MetaRow = styled.div`
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  padding: 5px 0;
  color: ${t.bone2};
`

const MetaKey = styled.span`
  color: ${t.mute};
  font-weight: 500;
`

const MetaVal = styled.span`
  font-family: ${t.fontMono};
  font-size: 12.5px;
  color: ${t.bone};
`

const SameAuthorItem = styled.div`
  display: flex;
  gap: 10px;
  padding: 7px 0;
  cursor: pointer;
`

const SameAuthorThumb = styled.div`
  width: 50px;
  height: 36px;
  border-radius: 2px;
  overflow: hidden;
  position: relative;
  flex: 0 0 auto;
`

const SameAuthorInfo = styled.div`
  flex: 1;
  min-width: 0;
`

const SameAuthorName = styled.div`
  font-family: ${t.fontDisplay};
  font-size: 14px;
  color: ${t.bone};
`

const SameAuthorMeta = styled.div`
  font-family: ${t.fontMono};
  font-size: 10px;
  color: ${t.mute};
`

const SeeAllLink = styled.button`
  font-family: ${t.fontMono};
  font-size: 10px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: ${t.accent2};
  background: none;
  border: none;
  cursor: pointer;
  margin-top: 8px;
  padding: 0;
`

const Permalink = styled.div`
  font-family: ${t.fontMono};
  font-size: 11px;
  color: ${t.bone2};
  background: ${t.surface};
  padding: 8px 10px;
  border-radius: 3px;
  border: 1px solid ${t.line};
  overflow-x: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
`

const PermalinkNote = styled.div`
  font-family: ${t.fontMono};
  font-size: 10px;
  color: ${t.mute};
  margin-top: 6px;
  line-height: 1.5;
`

const ModesList = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
`

const ModeNote = styled.div`
  font-family: ${t.fontMono};
  font-size: 10px;
  color: ${t.mute};
  margin-top: 10px;
  line-height: 1.5;
`

const Gallery = styled.div`
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 10px;
  margin-bottom: 32px;
`

const GalleryShot = styled.div`
  aspect-ratio: 4 / 3;
  background: ${t.surface};
  border-radius: 12px;
  position: relative;
  overflow: hidden;
`

const GalleryImg = styled.img`
  width: 100%;
  height: 100%;
  object-fit: cover;
`

function yearFromDate(date?: string): string | null {
  if (!date) return null
  const y = date.slice(0, 4)
  return /^\d{4}$/.test(y) ? y : null
}

function splitName(name: string): [string, string] {
  const parts = name.split(' ')
  if (parts.length <= 1) return [name, '']
  return [parts[0], parts.slice(1).join(' ')]
}

type Props = {
  map: BrowseMapEntry
  allMaps: BrowseMapEntry[]
  onBack: () => void
  onPlay: (name: string) => void
  onOpenAuthor: (author: string) => void
  onOpenMap: (map: BrowseMapEntry) => void
}

export default function DetailScreen({ map, allMaps, onBack, onPlay, onOpenAuthor, onOpenMap }: Props) {
  const year = yearFromDate(map.date)
  const [first, rest] = splitName(map.name)

  const sameAuthor = React.useMemo(
    () => allMaps.filter(m => m.author === map.author && m.name !== map.name).slice(0, 3),
    [allMaps, map]
  )

  return (
    <Wrap>
      <BackBar onClick={onBack}>
        <Icon name="arrow-l" size={11} /> Back to archive
      </BackBar>

      <HeroImg>
        {map.imageUrl ? (
          <HeroImage src={map.imageUrl} alt={map.name} />
        ) : (
          <HeroLabel>{map.name}.ogz</HeroLabel>
        )}
        <HeroOverlay />
        <HeroStamps>
          <span>{map.name.toUpperCase()}</span>
          {year && <><span>·</span><span>{year}</span></>}
          {map.modes && map.modes.length > 0 && (
            <><span>·</span><span>{map.modes.map(m => m.toUpperCase()).join(' / ')}</span></>
          )}
        </HeroStamps>
      </HeroImg>

      <Body>
        <Left>
          <DetailH1>
            {first} {rest && <em>{rest}</em>}
          </DetailH1>
          <Credits>
            by{' '}
            {map.author ? (
              <AuthName onClick={() => map.author && onOpenAuthor(map.author)}>
                {map.author}
              </AuthName>
            ) : (
              'Unknown'
            )}
            {year && <Year>{year}</Year>}
          </Credits>
          <PlayRow>
            <Btn $primary $large onClick={() => onPlay(map.name)}>
              <Icon name="play" size={13} /> Play this map
            </Btn>
            <Btn $large $ghost>
              <Icon name="download" size={12} /> Download .ogz
            </Btn>
          </PlayRow>
          {map.description && (
            <Descr>
              <p>{map.description}</p>
            </Descr>
          )}
          {map.imageUrls && map.imageUrls.length > 1 && (
            <>
              <FilterLabel>Screenshots · {map.imageUrls.length}</FilterLabel>
              <Gallery>
                {map.imageUrls.map((url, i) => (
                  <GalleryShot key={i}>
                    <GalleryImg src={url} alt={`${map.name} screenshot ${i + 1}`} loading="lazy" />
                  </GalleryShot>
                ))}
              </Gallery>
            </>
          )}
        </Left>

        <Sidebar>
          {/* Game modes */}
          {map.modes && map.modes.length > 0 && (
            <MetaSection>
              <FilterLabel>Game modes</FilterLabel>
              <ModesList>
                {map.modes.map((m, i) => (
                  <ModePill key={m} mode={m} primary={i === 0} />
                ))}
              </ModesList>
              <ModeNote>
                Designed primarily for <span style={{ color: t.bone2 }}>{map.modes[0].toUpperCase()}</span>.
              </ModeNote>
            </MetaSection>
          )}

          {/* Particulars */}
          <MetaSection>
            <FilterLabel>Particulars</FilterLabel>
            {year && (
              <MetaRow>
                <MetaKey>Released</MetaKey>
                <MetaVal>{map.date}</MetaVal>
              </MetaRow>
            )}
            {map.players && (
              <MetaRow>
                <MetaKey>Players</MetaKey>
                <MetaVal>{map.players}</MetaVal>
              </MetaRow>
            )}
          </MetaSection>

          {/* By the same author */}
          {sameAuthor.length > 0 && (
            <MetaSection>
              <FilterLabel>By the same author</FilterLabel>
              {sameAuthor.map(other => (
                <SameAuthorItem key={other.name} onClick={() => onOpenMap(other)}>
                  <SameAuthorThumb>
                    <MapThumb imageUrl={other.imageUrl} name={other.name} />
                  </SameAuthorThumb>
                  <SameAuthorInfo>
                    <SameAuthorName>{other.name}</SameAuthorName>
                    <SameAuthorMeta>
                      {yearFromDate(other.date) || ''}
                    </SameAuthorMeta>
                  </SameAuthorInfo>
                </SameAuthorItem>
              ))}
              {map.author && (
                <SeeAllLink onClick={() => onOpenAuthor(map.author!)}>
                  See all by {map.author} →
                </SeeAllLink>
              )}
            </MetaSection>
          )}

          {/* Permalink */}
          <MetaSection>
            <FilterLabel>Permalink</FilterLabel>
            <Permalink>sour.app/m/{map.name}</Permalink>
            <PermalinkNote>
              Static catalog page — sharable without loading the game.
            </PermalinkNote>
          </MetaSection>
        </Sidebar>
      </Body>
    </Wrap>
  )
}
