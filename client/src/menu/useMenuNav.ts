import * as React from 'react'
import type { MenuTab } from './TopBar'

export type MenuView =
  | { screen: 'tab'; tab: MenuTab }
  | { screen: 'detail'; mapId: string; from: MenuView }
  | { screen: 'author'; authorName: string; from: MenuView }

export type MenuAction =
  | { type: 'switchTab'; tab: MenuTab }
  | { type: 'openDetail'; mapId: string }
  | { type: 'openAuthor'; authorName: string }
  | { type: 'back' }

function reducer(state: MenuView, action: MenuAction): MenuView {
  switch (action.type) {
    case 'switchTab':
      return { screen: 'tab', tab: action.tab }
    case 'openDetail':
      return { screen: 'detail', mapId: action.mapId, from: state }
    case 'openAuthor':
      return { screen: 'author', authorName: action.authorName, from: state }
    case 'back':
      if (state.screen === 'detail' || state.screen === 'author') {
        return state.from
      }
      return state
    default:
      return state
  }
}

export function useMenuNav(initialTab: MenuTab = 'browse') {
  const [view, dispatch] = React.useReducer(reducer, {
    screen: 'tab',
    tab: initialTab,
  })

  const activeTab: MenuTab = view.screen === 'tab'
    ? view.tab
    : view.screen === 'detail' || view.screen === 'author'
      ? 'browse'
      : 'browse'

  return {
    view,
    activeTab,
    switchTab: (tab: MenuTab) => dispatch({ type: 'switchTab', tab }),
    openDetail: (mapId: string) => dispatch({ type: 'openDetail', mapId }),
    openAuthor: (authorName: string) => dispatch({ type: 'openAuthor', authorName }),
    goBack: () => dispatch({ type: 'back' }),
  }
}
