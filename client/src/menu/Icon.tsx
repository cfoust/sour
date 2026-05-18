import * as React from 'react'

export type IconName =
  | 'search' | 'play' | 'shuffle' | 'grid' | 'list'
  | 'arrow-l' | 'arrow-r' | 'star' | 'download' | 'settings'
  | 'globe' | 'users' | 'archive' | 'home' | 'x'
  | 'chev-r' | 'plus' | 'check' | 'expand' | 'crosshair'
  | 'flag' | 'wifi' | 'battery' | 'signal'

type Props = {
  name: IconName
  size?: number
}

export default function Icon({ name, size = 14 }: Props) {
  const s: React.SVGAttributes<SVGSVGElement> = {
    width: size,
    height: size,
    fill: 'none',
    stroke: 'currentColor',
    strokeWidth: 1.5,
    strokeLinecap: 'round',
    strokeLinejoin: 'round',
  }

  switch (name) {
    case 'search':
      return <svg {...s} viewBox="0 0 16 16"><circle cx="7" cy="7" r="4.5"/><path d="M10.5 10.5 14 14"/></svg>
    case 'play':
      return <svg {...s} viewBox="0 0 16 16" fill="currentColor" stroke="none"><path d="M5 3.5v9l8-4.5z"/></svg>
    case 'shuffle':
      return <svg {...s} viewBox="0 0 16 16"><path d="M2 4h2l8 8h2M2 12h2l8-8h2M12 2l2 2-2 2M12 10l2 2-2 2"/></svg>
    case 'grid':
      return <svg {...s} viewBox="0 0 16 16"><rect x="2" y="2" width="5" height="5"/><rect x="9" y="2" width="5" height="5"/><rect x="2" y="9" width="5" height="5"/><rect x="9" y="9" width="5" height="5"/></svg>
    case 'list':
      return <svg {...s} viewBox="0 0 16 16"><path d="M2 4h12M2 8h12M2 12h12"/></svg>
    case 'arrow-l':
      return <svg {...s} viewBox="0 0 16 16"><path d="M10 3 5 8l5 5"/></svg>
    case 'arrow-r':
      return <svg {...s} viewBox="0 0 16 16"><path d="M6 3l5 5-5 5"/></svg>
    case 'star':
      return <svg {...s} viewBox="0 0 16 16" fill="currentColor" stroke="none"><path d="M8 1.5l1.8 4 4.4.5-3.3 3 .9 4.3L8 11.3 4.2 13.3l.9-4.3-3.3-3 4.4-.5z"/></svg>
    case 'download':
      return <svg {...s} viewBox="0 0 16 16"><path d="M8 2v9M4 7l4 4 4-4M3 14h10"/></svg>
    case 'settings':
      return <svg {...s} viewBox="0 0 16 16"><circle cx="8" cy="8" r="2"/><path d="M8 1v2M8 13v2M1 8h2M13 8h2M3 3l1.4 1.4M11.6 11.6 13 13M3 13l1.4-1.4M11.6 4.4 13 3"/></svg>
    case 'globe':
      return <svg {...s} viewBox="0 0 16 16"><circle cx="8" cy="8" r="6"/><path d="M2 8h12M8 2c2 2 2 10 0 12M8 2c-2 2-2 10 0 12"/></svg>
    case 'users':
      return <svg {...s} viewBox="0 0 16 16"><circle cx="6" cy="6" r="2.5"/><path d="M1.5 14c0-2.5 2-4.5 4.5-4.5s4.5 2 4.5 4.5M10.5 5a2 2 0 0 1 0 4M14 14c0-1.5-.6-2.8-1.5-3.5"/></svg>
    case 'archive':
      return <svg {...s} viewBox="0 0 16 16"><rect x="2" y="3" width="12" height="3"/><path d="M3 6v8h10V6M6 9h4"/></svg>
    case 'home':
      return <svg {...s} viewBox="0 0 16 16"><path d="M2 7l6-5 6 5v7H2z"/><path d="M6 14V9h4v5"/></svg>
    case 'x':
      return <svg {...s} viewBox="0 0 16 16"><path d="M3 3l10 10M13 3 3 13"/></svg>
    case 'chev-r':
      return <svg {...s} viewBox="0 0 16 16"><path d="M6 4l4 4-4 4"/></svg>
    case 'plus':
      return <svg {...s} viewBox="0 0 16 16"><path d="M8 3v10M3 8h10"/></svg>
    case 'check':
      return <svg {...s} viewBox="0 0 16 16"><path d="M3 8l3 3 7-7"/></svg>
    case 'expand':
      return <svg {...s} viewBox="0 0 16 16"><path d="M3 7V3h4M13 9v4H9M3 9v4h4M13 7V3H9"/></svg>
    case 'crosshair':
      return <svg {...s} viewBox="0 0 16 16"><circle cx="8" cy="8" r="5"/><path d="M8 1v3M8 12v3M1 8h3M12 8h3"/></svg>
    case 'flag':
      return <svg {...s} viewBox="0 0 16 16"><path d="M3 14V2M3 2h9l-2 3 2 3H3"/></svg>
    case 'wifi':
      return <svg {...s} viewBox="0 0 16 16"><path d="M1 5a12 12 0 0 1 14 0M3 8a8 8 0 0 1 10 0M5 11a4 4 0 0 1 6 0"/><circle cx="8" cy="13.5" r=".5" fill="currentColor"/></svg>
    case 'battery':
      return <svg {...s} viewBox="0 0 24 16"><rect x="1" y="2" width="20" height="12" rx="2"/><rect x="3" y="4" width="14" height="8" fill="currentColor" stroke="none"/><rect x="22" y="6" width="1.5" height="4" fill="currentColor" stroke="none"/></svg>
    case 'signal':
      return <svg {...s} viewBox="0 0 16 16"><rect x="1" y="10" width="3" height="4" fill="currentColor" stroke="none"/><rect x="5" y="7" width="3" height="7" fill="currentColor" stroke="none"/><rect x="9" y="4" width="3" height="10" fill="currentColor" stroke="none"/><rect x="13" y="2" width="2.5" height="12" fill="currentColor" stroke="none"/></svg>
    default:
      return null
  }
}
