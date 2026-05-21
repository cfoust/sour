import * as React from 'react'
import styled from '@emotion/styled'
import { css } from '@emotion/react'
import { t } from './theme'

export const MODEL_LIST = [
  { id: 'mrfixit',       index: 0, name: 'Mr. Fixit',       hint: 'all-purpose wrecking machine' },
  { id: 'snoutx10k',     index: 1, name: 'IronSnout X10K',  hint: 'part pig, part machine' },
  { id: 'ogro',          index: 2, name: 'Ogro',            hint: 'smaller than a normal ogre' },
  { id: 'inky',          index: 3, name: 'Inky',            hint: 'the lesser evil' },
  { id: 'captaincannon', index: 4, name: 'Captain Cannon',  hint: 'righteous sense of justice' },
]

export const MODEL_PALETTES: Record<string, [string, string]> = {
  mrfixit:        ['#ffd60a', '#3a2810'],
  snoutx10k:      ['#ff7a87', '#3a1818'],
  ogro:           ['#7eda9a', '#2c4030'],
  inky:           ['#6fb8ff', '#0e1a24'],
  captaincannon:  ['#c8a560', '#3a2818'],
}

// Procedural avatar circle with first letter
export function ModelGlyph({ model, size = 28 }: { model: string; size?: number }) {
  const pal = MODEL_PALETTES[model] || MODEL_PALETTES.ogro
  return (
    <GlyphCircle
      style={{
        width: size, height: size,
        background: `radial-gradient(circle at 40% 35%, ${pal[0]} 0%, ${pal[1]} 75%)`,
      }}
    >
      <GlyphLetter style={{ color: pal[0], fontSize: size * 0.42 }}>
        {model[0].toUpperCase()}
      </GlyphLetter>
    </GlyphCircle>
  )
}

const GlyphCircle = styled.span`
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  flex: 0 0 auto;
  box-shadow: inset 0 -3px 6px rgba(0,0,0,0.25), inset 0 2px 3px rgba(255,255,255,0.3);
`

const GlyphLetter = styled.span`
  font-family: ${t.fontDisplay};
  font-weight: 900;
  letter-spacing: -0.02em;
  text-shadow: 0 1px 0 rgba(0,0,0,0.35);
  filter: brightness(1.4);
`

const Wrap = styled.div`
  position: relative;
`

const Chip = styled.button<{ $open?: boolean }>`
  display: inline-flex;
  align-items: center;
  gap: 9px;
  padding: 5px 12px 5px 5px;
  background: ${t.bgCard};
  border: 1px solid ${t.line};
  border-radius: 99px;
  cursor: pointer;
  box-shadow: 0 1px 0 rgba(40, 28, 8, 0.06);
  transition: background 0.12s, border-color 0.12s;

  &:hover { background: ${t.surface}; border-color: ${t.line2}; }

  ${p => p.$open && css`
    border-color: ${t.accent};
    background: ${t.softYel};
    box-shadow: 0 1px 0 ${t.accent3};
  `}
`

const ChipText = styled.span`
  display: flex;
  flex-direction: column;
  line-height: 1.1;
`

const ChipName = styled.span`
  font-family: ${t.fontDisplay};
  font-size: 13px;
  font-weight: 700;
  letter-spacing: -0.005em;
  color: ${t.bone};
`

const ChipModel = styled.span`
  font-family: ${t.fontMono};
  font-size: 10px;
  color: ${t.mute};
  letter-spacing: 0.04em;
  text-transform: lowercase;
`

const Caret = styled.svg<{ $open?: boolean }>`
  color: ${t.mute};
  transition: transform 0.15s;
  ${p => p.$open && 'transform: rotate(180deg);'}
`

const Popover = styled.div`
  position: absolute;
  top: calc(100% + 10px);
  right: 0;
  width: 380px;
  background: ${t.bgCard};
  border: 1px solid ${t.line};
  border-radius: 16px;
  padding: 16px;
  box-shadow: 0 10px 0 rgba(40, 28, 8, 0.06), 0 24px 48px rgba(40, 28, 8, 0.18);
  z-index: 50;
  display: flex;
  flex-direction: column;
  gap: 12px;

  &::before {
    content: '';
    position: absolute;
    top: -7px;
    right: 32px;
    width: 12px;
    height: 12px;
    background: ${t.bgCard};
    border-left: 1px solid ${t.line};
    border-top: 1px solid ${t.line};
    transform: rotate(45deg);
  }
`

const PopSection = styled.div`
  display: flex;
  flex-direction: column;
  gap: 6px;
`

const PopLabel = styled.label`
  font-family: ${t.fontDisplay};
  font-size: 10.5px;
  font-weight: 700;
  letter-spacing: 0.22em;
  text-transform: uppercase;
  color: ${t.mute};
`

const PopInputWrap = styled.div`
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 9px 12px;
  background: ${t.bg};
  border: 1px solid ${t.line};
  border-radius: 10px;

  &:focus-within { border-color: ${t.accent}; }
`

const PopInput = styled.input`
  flex: 1;
  background: none;
  border: none;
  outline: none;
  font-family: ${t.fontDisplay};
  font-size: 14px;
  font-weight: 700;
  letter-spacing: -0.01em;
  color: ${t.bone};
`

const CharCount = styled.span`
  font-family: ${t.fontMono};
  font-size: 10px;
  color: ${t.mute2};
  letter-spacing: 0;
`

const PopDivider = styled.div`
  height: 1px;
  background: ${t.line};
  margin: 2px -16px;
`

const ModelGrid = styled.div`
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  gap: 6px;
`

const ModelBtn = styled.button<{ $active?: boolean }>`
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 10px 6px 8px;
  background: ${t.bg};
  border: 1px solid ${t.line};
  border-radius: 10px;
  cursor: pointer;
  text-align: center;
  transition: background 0.12s;

  &:hover { background: ${t.surface}; border-color: ${t.line2}; }

  ${p => p.$active && css`
    background: ${t.softYel};
    border-color: ${t.accent};
    box-shadow: 0 2px 0 ${t.accent3};
  `}
`

const ModelName = styled.span`
  font-family: ${t.fontDisplay};
  font-size: 12.5px;
  font-weight: 700;
  letter-spacing: -0.005em;
  color: ${t.bone};
`

const ModelHint = styled.span`
  font-family: ${t.fontMono};
  font-size: 9.5px;
  color: ${t.mute};
  letter-spacing: 0.04em;
`

type Props = {
  name: string
  model: string
  onNameChange?: (name: string) => void
  onModelChange?: (model: string) => void
}

export default function PlayerChip({ name, model, onNameChange, onModelChange }: Props) {
  const [open, setOpen] = React.useState(false)
  const [localName, setLocalName] = React.useState(name)
  const wrapRef = React.useRef<HTMLDivElement>(null)

  // Sync from parent
  React.useEffect(() => { setLocalName(name) }, [name])

  // Close on click outside
  React.useEffect(() => {
    if (!open) return
    const handler = (e: MouseEvent) => {
      if (wrapRef.current && !wrapRef.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [open])

  const handleNameBlur = () => {
    const trimmed = localName.trim()
    if (trimmed && trimmed !== name) {
      onNameChange?.(trimmed)
    }
  }

  const handleNameKeyDown = (e: React.KeyboardEvent) => {
    // Prevent game engine from consuming keypresses while typing
    e.stopPropagation()
    if (e.key === 'Enter') {
      handleNameBlur()
      ;(e.target as HTMLInputElement).blur()
    }
  }

  const currentModel = MODEL_LIST.find(m => m.id === model) || MODEL_LIST[2]

  return (
    <Wrap ref={wrapRef}>
      <Chip $open={open} onClick={() => setOpen(!open)}>
        <ModelGlyph model={model} size={30} />
        <ChipText>
          <ChipName>{name}</ChipName>
          <ChipModel>{currentModel.name.toLowerCase()}</ChipModel>
        </ChipText>
        <Caret
          $open={open}
          width="10" height="10" viewBox="0 0 10 10"
          fill="none" stroke="currentColor" strokeWidth="1.5"
          strokeLinecap="round" strokeLinejoin="round"
        >
          <path d="M2.5 4l2.5 2.5L7.5 4" />
        </Caret>
      </Chip>

      {open && (
        <Popover>
          <PopSection>
            <PopLabel>Name</PopLabel>
            <PopInputWrap>
              <PopInput
                value={localName}
                maxLength={16}
                onChange={e => setLocalName(e.target.value)}
                onBlur={handleNameBlur}
                onKeyDown={handleNameKeyDown}
                onKeyUp={e => e.stopPropagation()}
                onKeyPress={e => e.stopPropagation()}
                onClick={e => e.stopPropagation()}
              />
              <CharCount>{localName.length} / 16</CharCount>
            </PopInputWrap>
          </PopSection>

          <PopDivider />

          <PopSection>
            <PopLabel>Player model</PopLabel>
            <ModelGrid>
              {MODEL_LIST.map(m => (
                <ModelBtn
                  key={m.id}
                  $active={m.id === model}
                  onClick={() => onModelChange?.(m.id)}
                >
                  <ModelGlyph model={m.id} size={42} />
                  <ModelName>{m.name}</ModelName>
                  <ModelHint>{m.hint}</ModelHint>
                </ModelBtn>
              ))}
            </ModelGrid>
          </PopSection>
        </Popover>
      )}
    </Wrap>
  )
}
