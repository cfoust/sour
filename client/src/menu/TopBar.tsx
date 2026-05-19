import * as React from 'react'
import styled from '@emotion/styled'
import { css } from '@emotion/react'
import { t } from './theme'
import Icon from './Icon'
import { Mono } from './styled'

const Header = styled.header`
  display: flex;
  align-items: center;
  border-bottom: 1px solid ${t.line};
  height: 68px;
  flex: 0 0 auto;
  padding: 0 28px;
  gap: 20px;
`

const Brand = styled.div`
  display: flex;
  align-items: center;
  gap: 10px;
  padding-right: 24px;
  margin-right: 4px;
  border-right: 1px solid ${t.line};
  height: 100%;
`

const BrandName = styled.span`
  font-family: ${t.fontDisplay};
  font-size: 26px;
  font-weight: 900;
  letter-spacing: -0.04em;
  color: ${t.bone};
  display: inline-flex;
  align-items: center;
  gap: 7px;

  &::after {
    content: '';
    width: 14px;
    height: 14px;
    background: ${t.accent};
    border-radius: 50%;
    box-shadow:
      inset -3px -3px 0 rgba(0,0,0,0.06),
      inset 3px 3px 0 rgba(255,255,255,0.4),
      0 2px 0 ${t.accent3};
  }
`

const Tabs = styled.nav`
  display: flex;
  align-items: center;
  gap: 4px;
  flex: 1 1 auto;
`

const Tab = styled.button<{ $active?: boolean }>`
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 9px 18px;
  font-family: ${t.fontDisplay};
  font-size: 14px;
  font-weight: 600;
  letter-spacing: -0.005em;
  color: ${t.mute};
  border: none;
  background: transparent;
  cursor: pointer;
  border-radius: 10px;
  transition: background 0.12s, color 0.12s;

  &:hover { color: ${t.bone}; background: rgba(40, 28, 8, 0.04); }

  ${p => p.$active && css`
    color: ${t.bone};
    background: ${t.accent};
    box-shadow: 0 3px 0 ${t.accent3};
    &:hover { background: ${t.accent}; }
  `}
`

const TabCount = styled.span<{ $active?: boolean }>`
  font-family: ${t.fontUi};
  font-size: 12px;
  font-weight: 500;
  color: ${p => p.$active ? t.bone2 : t.mute2};
  letter-spacing: 0;
`

const Right = styled.div`
  display: flex;
  align-items: center;
  gap: 10px;
  margin-left: auto;
`

const InGameBadge = styled.span`
  display: flex;
  align-items: center;
  gap: 6px;
  font-family: ${t.fontMono};
  font-size: 10px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: ${t.accent2};
`

const InGameDot = styled.span`
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: ${t.accent};
  box-shadow: 0 0 8px ${t.accent};
`

const SearchWrap = styled.div`
  display: flex;
  align-items: center;
  gap: 8px;
  background: ${t.bgCard};
  border: 1px solid ${t.line};
  border-radius: 12px;
  padding: 9px 14px;
  width: 300px;
  font-family: ${t.fontUi};
  font-size: 13px;
  color: ${t.bone};
  box-shadow: 0 1px 0 rgba(40, 28, 8, 0.04);

  &:focus-within { border-color: ${t.accent}; }
`

const SearchInput = styled.input`
  flex: 1;
  background: none;
  border: none;
  outline: none;
  font-family: inherit;
  font-size: inherit;
  color: inherit;

  &::placeholder { color: ${t.mute}; }
`

const Kbd = styled.span`
  font-family: ${t.fontMono};
  font-size: 10px;
  color: ${t.mute};
  background: ${t.surface};
  border: 1px solid ${t.line};
  padding: 2px 6px;
  margin-left: auto;
  border-radius: 5px;
`

export type MenuTab = 'home' | 'browse' | 'mods' | 'servers' | 'settings'

type TabDef = {
  id: MenuTab
  name: string
  count?: string
}

type Props = {
  active: MenuTab
  onTab: (tab: MenuTab) => void
  isInGame?: boolean
  mapCount?: number
  searchQuery?: string
  onSearch?: (query: string) => void
}

const TABS: TabDef[] = [
  { id: 'home', name: 'Home' },
  { id: 'browse', name: 'Maps' },
  { id: 'mods', name: 'Mods' },
  { id: 'servers', name: 'Servers' },
  { id: 'settings', name: 'Settings' },
]

export default function TopBar({ active, onTab, isInGame, mapCount, searchQuery, onSearch }: Props) {
  const searchRef = React.useRef<HTMLInputElement>(null)

  React.useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault()
        searchRef.current?.focus()
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [])

  return (
    <Header>
      <Brand>
        <BrandName>Sour</BrandName>
      </Brand>
      <Tabs>
        {TABS.map(tab => {
          const isActive = active === tab.id
          const count = tab.id === 'browse' && mapCount
            ? mapCount.toLocaleString()
            : tab.count
          return (
            <Tab key={tab.id} $active={isActive} onClick={() => onTab(tab.id)}>
              {tab.name}
              {count && <TabCount $active={isActive}>{count}</TabCount>}
            </Tab>
          )
        })}
      </Tabs>
      <Right>
        {isInGame && (
          <InGameBadge>
            <InGameDot />
            in game
          </InGameBadge>
        )}
        <SearchWrap>
          <Icon name="search" size={12} />
          <SearchInput
            ref={searchRef}
            placeholder={`Search ${mapCount ? mapCount.toLocaleString() + ' ' : ''}maps, authors…`}
            value={searchQuery || ''}
            onChange={e => onSearch?.(e.target.value)}
          />
          <Kbd>⌘K</Kbd>
        </SearchWrap>
      </Right>
    </Header>
  )
}
