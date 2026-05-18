// Design tokens — source of truth for the Sour menu visual system.
// Friendly bright: warm cream paper, sunshine yellow accent, chunky cartoon shadows.

export const t = {
  // Backgrounds
  bg: '#fffceb',
  bgDeep: '#fff4c2',
  bgCard: '#ffffff',
  surface: '#fff8d6',
  surface2: '#ffeb88',
  softYel: '#fff3a0',

  // Borders
  line: 'rgba(50, 35, 10, 0.07)',
  line2: 'rgba(50, 35, 10, 0.13)',
  line3: 'rgba(50, 35, 10, 0.22)',

  // Text
  bone: '#2a2010',
  bone2: '#56492d',
  mute: '#8a7d5a',
  mute2: '#b8ad8a',
  dim: '#e6dab0',

  // Primary accent
  accent: '#ffd60a',
  accent2: '#ffe34a',
  accent3: '#ffb300',
  accentDim: 'rgba(255, 214, 10, 0.24)',

  // Secondary pops
  coral: '#ff7a87',
  coral2: '#ff5570',
  sky: '#6fb8ff',
  sky2: '#4a9aff',
  mint: '#7eda9a',
  mint2: '#4ec873',
  grape: '#b88aff',

  // Semantic
  ok: '#4ec873',
  danger: '#ff5570',

  // Typography
  fontDisplay: '"Inter", system-ui, sans-serif',
  fontUi: '"Inter", system-ui, sans-serif',
  fontMono: '"JetBrains Mono", ui-monospace, monospace',

  // Shadows
  cardShadow: '0 2px 0 rgba(40, 28, 8, 0.06)',
  cardHoverShadow: '0 8px 16px rgba(40, 28, 8, 0.1)',
  btnShadow: '0 2px 0 rgba(40, 28, 8, 0.08)',
  btnPrimaryShadow: '0 4px 0 #ffb300',
  modalShadow: '0 12px 0 rgba(40,28,8,0.08), 0 24px 48px rgba(40,28,8,0.15)',
} as const
