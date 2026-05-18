import * as React from 'react'
import styled from '@emotion/styled'
import { t } from '../theme'
import { Btn, BtnIcon, FilterLabel, Mono } from '../styled'
import Icon from '../Icon'
import ModePill from '../ModePill'
import type { BrowseMapEntry } from '../../catalog/types'

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

const HeroWrap = styled.div`
  height: 240px;
  margin: 12px 16px 0;
  border-radius: 18px;
  overflow: hidden;
  position: relative;
  flex: 0 0 auto;
  background:
    radial-gradient(circle at 60% 40%, rgba(180, 100, 60, 0.3), transparent 60%),
    linear-gradient(135deg, #3a2818 0%, #1a1612 100%);
`

const HeroImg = styled.img`
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
`

const HeroOverlay = styled.div`
  position: absolute;
  inset: 0;
  background:
    linear-gradient(180deg, rgba(20, 17, 13, 0.5) 0%, transparent 30%, rgba(20, 17, 13, 0.95) 100%);
  z-index: 1;
`

const HeroScanlines = styled.div`
  position: absolute;
  inset: 0;
  background: repeating-linear-gradient(
    0deg,
    rgba(255, 255, 255, 0.025) 0 1px,
    transparent 1px 3px
  );
  z-index: 2;
`

const HeroNav = styled.div`
  position: absolute;
  top: 16px;
  left: 18px;
  right: 18px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  z-index: 3;
`

const HeroNavBtn = styled.button`
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(20, 17, 13, 0.6);
  backdrop-filter: blur(4px);
  border: none;
  border-radius: 10px;
  color: white;
  cursor: pointer;
`

const HeroMeta = styled.div`
  position: absolute;
  bottom: 14px;
  left: 18px;
  right: 18px;
  display: flex;
  gap: 8px;
  align-items: center;
  font-family: ${t.fontMono};
  font-size: 9px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: rgba(236, 228, 214, 0.7);
  z-index: 3;
`

const Content = styled.div`
  flex: 1 1 auto;
  overflow-y: auto;
  padding: 22px 18px 26px;
`

const MobileH1 = styled.h1`
  font-family: ${t.fontDisplay};
  font-size: 38px;
  font-weight: 900;
  line-height: 0.95;
  letter-spacing: -0.03em;
  margin: 0 0 8px;
  color: ${t.bone};

  em {
    font-style: normal;
    background: ${t.accent};
    color: ${t.bone};
    padding: 0 8px;
    border-radius: 8px;
  }
`

const CreditsLine = styled.div`
  font-family: ${t.fontUi};
  font-size: 14px;
  color: ${t.bone2};
  margin-bottom: 20px;
  font-weight: 500;
`

const AuthSpan = styled.span`
  color: ${t.bone};
  font-weight: 700;
  font-style: italic;
  cursor: pointer;
`

const PlayRow = styled.div`
  display: flex;
  gap: 8px;
  margin-bottom: 22px;

  .play-primary { flex: 1; justify-content: center; }
`

const Descr = styled.div`
  font-family: ${t.fontUi};
  font-size: 14.5px;
  line-height: 1.65;
  color: ${t.bone2};
  margin-bottom: 22px;
`

const MetaList = styled.div`
  border-top: 1px solid ${t.line};
  padding-top: 14px;
`

const MetaRow = styled.div`
  display: flex;
  justify-content: space-between;
  padding: 6px 0;
  font-size: 13.5px;
  color: ${t.bone2};
`

const MetaKey = styled.span`
  font-family: ${t.fontUi};
  font-size: 13px;
  font-weight: 500;
  color: ${t.mute};
`

const MetaVal = styled.span`
  font-family: ${t.fontMono};
  font-size: 12.5px;
  color: ${t.bone};
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
  onBack: () => void
  onPlay: (name: string) => void
  onOpenAuthor: (author: string) => void
}

export default function MobileDetail({ map, onBack, onPlay, onOpenAuthor }: Props) {
  const year = yearFromDate(map.date)
  const [first, rest] = splitName(map.name)

  return (
    <Frame>
      <HeroWrap>
        {map.imageUrl && <HeroImg src={map.imageUrl} alt={map.name} />}
        <HeroOverlay />
        <HeroScanlines />
        <HeroNav>
          <HeroNavBtn onClick={onBack}><Icon name="arrow-l" size={12} /></HeroNavBtn>
          <HeroNavBtn><Icon name="globe" size={12} /></HeroNavBtn>
        </HeroNav>
        <HeroMeta>
          <span>{map.name.toUpperCase()}</span>
          {year && <><span>·</span><span>{year}</span></>}
        </HeroMeta>
      </HeroWrap>

      <Content>
        <MobileH1>
          {first} {rest && <em>{rest}</em>}
        </MobileH1>
        <CreditsLine>
          by{' '}
          {map.author ? (
            <AuthSpan onClick={() => map.author && onOpenAuthor(map.author)}>
              {map.author}
            </AuthSpan>
          ) : 'Unknown'}
          {year && ` · ${year}`}
        </CreditsLine>

        <PlayRow>
          <Btn $primary $large className="play-primary" onClick={() => onPlay(map.name)}>
            <Icon name="play" size={13} /> Play this map
          </Btn>
          <BtnIcon style={{ width: 42, height: 38 }}>
            <Icon name="download" size={13} />
          </BtnIcon>
        </PlayRow>

        {map.description && (
          <Descr>{map.description}</Descr>
        )}

        <MetaList>
          {year && (
            <MetaRow>
              <MetaKey>Released</MetaKey>
              <MetaVal>{map.date}</MetaVal>
            </MetaRow>
          )}
        </MetaList>
      </Content>
    </Frame>
  )
}
