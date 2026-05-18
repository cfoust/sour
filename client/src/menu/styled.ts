import styled from '@emotion/styled'
import { css } from '@emotion/react'
import { t } from './theme'

// Full-screen menu overlay with backdrop blur and cream gradient
export const MenuOverlay = styled.div`
  position: absolute;
  inset: 0;
  backdrop-filter: blur(10px) saturate(0.9);
  -webkit-backdrop-filter: blur(10px) saturate(0.9);
  background:
    radial-gradient(circle at 88% 8%, rgba(255, 214, 10, 0.16), transparent 40%),
    radial-gradient(circle at 8% 95%, rgba(255, 122, 135, 0.10), transparent 38%),
    linear-gradient(180deg, rgba(255, 252, 235, 0.97) 0%, rgba(255, 250, 225, 0.98) 100%);
  display: flex;
  flex-direction: column;
  color: ${t.bone};
  font-family: ${t.fontUi};
  font-size: 14.5px;
  line-height: 1.5;
  letter-spacing: -0.005em;
  -webkit-font-smoothing: antialiased;
  text-rendering: optimizeLegibility;
`

// Chunky cartoon button
export const Btn = styled.button<{
  $primary?: boolean
  $ghost?: boolean
  $large?: boolean
  $full?: boolean
  $danger?: boolean
}>`
  font-family: ${t.fontDisplay};
  font-size: 13.5px;
  font-weight: 600;
  letter-spacing: -0.005em;
  padding: 10px 18px;
  border: 1px solid ${t.line2};
  background: ${t.bgCard};
  color: ${t.bone};
  border-radius: 12px;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  transition: transform 0.08s, box-shadow 0.08s, background 0.08s;
  box-shadow: ${t.btnShadow};

  &:hover { background: ${t.surface}; }
  &:active { transform: translateY(1px); box-shadow: 0 1px 0 rgba(40, 28, 8, 0.08); }

  ${p => p.$primary && css`
    background: ${t.accent};
    border-color: ${t.accent};
    font-weight: 700;
    box-shadow: ${t.btnPrimaryShadow};
    &:hover { background: ${t.accent2}; }
    &:active { transform: translateY(2px); box-shadow: 0 2px 0 ${t.accent3}; }
  `}

  ${p => p.$ghost && css`
    background: transparent;
    box-shadow: none;
  `}

  ${p => p.$large && css`
    font-size: 15px;
    padding: 13px 24px;
    border-radius: 14px;
  `}

  ${p => p.$full && css`
    grid-column: 1 / -1;
  `}

  ${p => p.$danger && css`
    color: ${t.coral2};
    border-color: ${t.accentDim};
  `}
`

// Small square icon button
export const BtnIcon = styled.button`
  width: 36px;
  height: 36px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: ${t.bgCard};
  border: 1px solid ${t.line};
  color: ${t.bone2};
  cursor: pointer;
  border-radius: 10px;
  box-shadow: 0 1px 0 rgba(40, 28, 8, 0.06);

  &:hover { background: ${t.surface}; color: ${t.bone}; }
`

// Monospace span
export const Mono = styled.span`
  font-family: ${t.fontMono};
`

// Yellow highlighter marker for headline emphasis
export const Marker = styled.em`
  font-style: normal;
  background: ${t.accent};
  color: ${t.bone};
  padding: 0 8px;
  border-radius: 6px;
`

// Section header
export const SectionHeadWrap = styled.div`
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  padding: 32px 32px 14px;
`

export const SectionTitle = styled.h3`
  font-family: ${t.fontDisplay};
  font-size: 28px;
  font-weight: 800;
  letter-spacing: -0.025em;
  margin: 0;
  color: ${t.bone};
  display: inline-flex;
  align-items: center;
  gap: 12px;
`

export const CountPill = styled.span`
  font-family: ${t.fontUi};
  font-size: 13px;
  font-weight: 600;
  background: ${t.accent};
  color: ${t.bone};
  padding: 3px 10px;
  border-radius: 99px;
  letter-spacing: 0;
`

export const SectionMeta = styled.div`
  font-family: ${t.fontUi};
  font-size: 13px;
  color: ${t.mute};
  margin-top: 6px;
`

export const SectionLink = styled.button`
  font-family: ${t.fontDisplay};
  font-size: 13px;
  font-weight: 600;
  color: ${t.bone};
  text-decoration: none;
  background: ${t.bgCard};
  padding: 8px 14px;
  border-radius: 10px;
  border: 1px solid ${t.line};
  box-shadow: 0 2px 0 rgba(40, 28, 8, 0.06);
  cursor: pointer;

  &:hover { background: ${t.surface}; }
`

// Filter group label
export const FilterLabel = styled.h4`
  font-family: ${t.fontDisplay};
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: ${t.mute};
  margin: 0 0 8px;
`

// Meta sidebar section
export const MetaSection = styled.div`
  padding: 16px 0;
  border-bottom: 1px solid ${t.line};

  &:first-of-type { padding-top: 0; }
  &:last-of-type { border-bottom: none; }
`
