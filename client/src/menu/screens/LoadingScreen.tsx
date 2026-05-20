import * as React from 'react'
import styled from '@emotion/styled'
import { keyframes } from '@emotion/react'
import { t } from '../theme'
import type { GameState } from '../../types'
import { GameStateType } from '../../types'
import type { BrowseMapEntry } from '../../catalog/types'

const drift = keyframes`
  0%   { transform: scale(1.04) translate(-1.5%, -1%); }
  100% { transform: scale(1.08) translate( 1.5%,  1%); }
`

const Wrap = styled.div`
  position: absolute;
  inset: 0;
  background: ${t.bone};
  color: #f7eecf;
  overflow: hidden;
  font-family: ${t.fontUi};
`

const BgLayer = styled.div`
  position: absolute;
  inset: 0;
  overflow: hidden;
`

const BgGradient = styled.div`
  position: absolute;
  inset: -8%;
  background-blend-mode: screen, normal;
  animation: ${drift} 22s ease-in-out infinite alternate;
  transform-origin: 50% 50%;
`

const BgScan = styled.div`
  position: absolute;
  inset: 0;
  background: repeating-linear-gradient(0deg, rgba(255,255,255,0.03) 0 1px, transparent 1px 3px);
  mix-blend-mode: overlay;
  pointer-events: none;
`

const BgVignette = styled.div`
  position: absolute;
  inset: 0;
  background:
    linear-gradient(180deg, rgba(20,17,13,0.35) 0%, transparent 25%, transparent 55%, rgba(20,17,13,0.92) 100%),
    radial-gradient(ellipse at 50% 55%, transparent 35%, rgba(20,17,13,0.7) 100%);
  pointer-events: none;
`

const BgGrain = styled.div`
  position: absolute;
  inset: 0;
  background-image:
    repeating-linear-gradient(0deg, rgba(0,0,0,0.06) 0 1px, transparent 1px 4px),
    repeating-linear-gradient(90deg, rgba(255,255,255,0.025) 0 1px, transparent 1px 5px);
  mix-blend-mode: overlay;
  opacity: 0.6;
  pointer-events: none;
`

const Stamps = styled.div`
  position: absolute;
  top: 38px;
  left: 40px;
  display: flex;
  align-items: center;
  gap: 12px;
  font-family: ${t.fontMono};
  font-size: 10.5px;
  letter-spacing: 0.22em;
  text-transform: uppercase;
  color: rgba(247, 238, 207, 0.7);
  z-index: 5;

  &::before {
    content: '';
    display: block;
    width: 32px;
    height: 1px;
    background: ${t.accent};
    margin-right: 4px;
  }
`

const StampSep = styled.span`
  color: rgba(247, 238, 207, 0.25);
`

const DestPanel = styled.div`
  position: absolute;
  top: 32px;
  right: 40px;
  z-index: 5;
  color: rgba(247, 238, 207, 0.95);
  font-family: ${t.fontUi};
  text-align: right;
  max-width: 320px;
`

const DestEyebrow = styled.div`
  font-family: ${t.fontDisplay};
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.22em;
  text-transform: uppercase;
  color: ${t.accent};
  margin-bottom: 6px;
`

const DestLine = styled.div`
  font-family: ${t.fontDisplay};
  font-size: 18px;
  font-weight: 700;
  letter-spacing: -0.015em;
  color: #fff;
`

const DestMeta = styled.div`
  font-family: ${t.fontMono};
  font-size: 11px;
  color: rgba(247, 238, 207, 0.55);
  letter-spacing: 0.06em;
  margin-top: 4px;
`

const TitleBlock = styled.div`
  position: absolute;
  left: 40px;
  bottom: 96px;
  z-index: 5;
  max-width: 70%;
`

const TitleEyebrow = styled.div`
  font-family: ${t.fontDisplay};
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.22em;
  text-transform: uppercase;
  color: ${t.accent};
  margin-bottom: 10px;
`

const TitleH1 = styled.h1`
  font-family: ${t.fontDisplay};
  font-size: 124px;
  font-weight: 900;
  line-height: 0.9;
  letter-spacing: -0.05em;
  color: #fff;
  text-shadow: 0 4px 24px rgba(0, 0, 0, 0.55);
  margin: 0;

  em {
    font-style: normal;
    background: ${t.accent};
    color: ${t.bone};
    padding: 0 12px;
    border-radius: 14px;
    text-shadow: none;
  }

  @media (max-width: 768px) {
    font-size: 64px;
  }
`

const Byline = styled.div`
  font-family: ${t.fontUi};
  font-size: 17px;
  color: rgba(247, 238, 207, 0.7);
  font-weight: 500;
  margin-top: 16px;
  letter-spacing: -0.005em;

  .auth {
    font-style: italic;
    color: #fff;
    font-weight: 600;
  }
`

const NoteBlock = styled.div`
  position: absolute;
  right: 40px;
  bottom: 96px;
  width: 280px;
  z-index: 5;
`

const NoteEyebrow = styled.div`
  font-family: ${t.fontDisplay};
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.22em;
  text-transform: uppercase;
  color: rgba(247, 238, 207, 0.5);
  margin-bottom: 8px;
  padding-bottom: 8px;
  border-bottom: 1px solid rgba(247, 238, 207, 0.12);
`

const NoteText = styled.p`
  font-family: ${t.fontUi};
  font-size: 12.5px;
  line-height: 1.55;
  color: rgba(247, 238, 207, 0.7);
  margin: 0;
  font-style: italic;
`

const ProgressWrap = styled.div`
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  padding: 0 40px 22px;
  z-index: 6;
`

const ProgressRail = styled.div`
  position: relative;
  height: 2px;
  background: rgba(247, 238, 207, 0.12);
  border-radius: 1px;
  overflow: hidden;
`

const ProgressFill = styled.div`
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  background: ${t.accent};
  box-shadow: 0 0 12px rgba(255, 214, 10, 0.55);
  transition: width 0.4s ease-out;

  &::after {
    content: '';
    position: absolute;
    right: 0;
    top: 0;
    bottom: 0;
    width: 60px;
    background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.6));
  }
`

const ProgressMeta = styled.div`
  display: flex;
  align-items: center;
  gap: 10px;
  font-family: ${t.fontMono};
  font-size: 10.5px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: rgba(247, 238, 207, 0.6);
  margin-top: 10px;
`

const ProgressDot = styled.span`
  color: rgba(247, 238, 207, 0.25);
`

// Deterministic color palette from map name
function mapPalette(name: string): string[] {
  let h = 2166136261
  for (let i = 0; i < name.length; i++) {
    h ^= name.charCodeAt(i)
    h = (h * 16777619) >>> 0
  }
  const hue1 = h % 360
  const hue2 = (hue1 + 40) % 360
  return [
    `hsl(${hue1}, 25%, 12%)`,
    `hsl(${hue1}, 30%, 18%)`,
    `hsl(${hue2}, 35%, 28%)`,
    `hsl(${hue2}, 20%, 15%)`,
  ]
}

function getProgress(state: GameState): number {
  switch (state.type) {
    case GameStateType.PageLoading: return 0
    case GameStateType.Downloading: return state.progress * 100
    case GameStateType.Running: return 100
    case GameStateType.MapChange: return 100
    case GameStateType.Ready: return 100
    case GameStateType.GameError: return 0
  }
}

function getStatusText(state: GameState): string {
  switch (state.type) {
    case GameStateType.PageLoading: return 'Initializing'
    case GameStateType.Downloading: return 'Loading assets'
    case GameStateType.Running: return 'Starting game'
    case GameStateType.MapChange: return 'Loading map'
    case GameStateType.Ready: return 'Ready'
    case GameStateType.GameError: return 'Error'
  }
}

type Props = {
  state: GameState
  mapName?: string
  mapEntry?: BrowseMapEntry
}

export default function LoadingScreen({ state, mapName, mapEntry }: Props) {
  const progress = getProgress(state)
  const statusText = getStatusText(state)
  const name = mapEntry?.name || mapName || ''
  const pal = mapPalette(name || 'sour')

  const titleWords = name.split(/(?=[A-Z_-])/).join(' ').split(' ').filter(Boolean)
  const first = titleWords[0] || ''
  const rest = titleWords.slice(1).join(' ')

  return (
    <Wrap>
      <BgLayer>
        <BgGradient style={{
          background: `
            radial-gradient(70% 60% at 35% 35%, ${pal[2]}cc 0%, transparent 60%),
            radial-gradient(80% 70% at 75% 80%, ${pal[3]}55 0%, transparent 55%),
            linear-gradient(135deg, ${pal[1]} 0%, ${pal[0]} 100%)
          `,
        }} />
        <BgScan />
        <BgVignette />
        <BgGrain />
      </BgLayer>

      <Stamps>
        {name && (
          <>
            <span>{name.toUpperCase()}</span>
            <StampSep>·</StampSep>
          </>
        )}
        {mapEntry?.date && (
          <>
            <span>{mapEntry.date.slice(0, 4)}</span>
            <StampSep>·</StampSep>
          </>
        )}
        {mapEntry?.modes && mapEntry.modes.length > 0 && (
          <>
            <span>{mapEntry.modes.map(m => m.toUpperCase()).join(' / ')}</span>
            <StampSep>·</StampSep>
          </>
        )}
        {mapEntry?.author && (
          <span>by {mapEntry.author}</span>
        )}
        {!name && <span>{statusText}</span>}
      </Stamps>

      {name && (
        <DestPanel>
          <DestEyebrow>Loading map</DestEyebrow>
          <DestLine>
            <strong>{name}</strong>
            {mapEntry?.modes?.[0] && ` · ${mapEntry.modes[0].toUpperCase()}`}
          </DestLine>
          <DestMeta>.ogz</DestMeta>
        </DestPanel>
      )}

      {name && (
        <TitleBlock>
          <TitleEyebrow>Now loading</TitleEyebrow>
          <TitleH1>
            {first}
            {rest && <> <em>{rest}</em></>}
          </TitleH1>
          {mapEntry?.author && (
            <Byline>
              by <span className="auth">{mapEntry.author}</span>
              {mapEntry.date && ` · ${mapEntry.date.slice(0, 4)}`}
            </Byline>
          )}
        </TitleBlock>
      )}

      {!name && (
        <TitleBlock>
          <TitleEyebrow>{statusText}</TitleEyebrow>
          <TitleH1 style={{ fontSize: 72 }}>Sour</TitleH1>
        </TitleBlock>
      )}

      {mapEntry?.description && (
        <NoteBlock>
          <NoteEyebrow>From the archive note</NoteEyebrow>
          <NoteText>{mapEntry.description}</NoteText>
        </NoteBlock>
      )}

      <ProgressWrap>
        <ProgressRail>
          <ProgressFill style={{ width: `${progress}%` }} />
        </ProgressRail>
        <ProgressMeta>
          <span>{statusText}</span>
          <ProgressDot>·</ProgressDot>
          <span>{Math.round(progress)}%</span>
          {state.type === GameStateType.Downloading && state.text && (
            <>
              <ProgressDot>·</ProgressDot>
              <span>{state.text}</span>
            </>
          )}
        </ProgressMeta>
      </ProgressWrap>
    </Wrap>
  )
}
