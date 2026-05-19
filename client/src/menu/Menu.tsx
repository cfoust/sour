import * as React from 'react'
import styled from '@emotion/styled'
import { t } from './theme'
import { MenuOverlay } from './styled'
import TopBar from './TopBar'
import { useMenuNav } from './useMenuNav'
import type { MenuTab } from './TopBar'
import type { BrowseMapEntry } from '../catalog/types'
import { BROWSER } from '../utils'

import HomeScreen from './screens/HomeScreen'
import BrowseScreen from './screens/BrowseScreen'
import DetailScreen from './screens/DetailScreen'
import AuthorScreen from './screens/AuthorScreen'
import PauseScreen from './screens/PauseScreen'
import MobileBrowse from './screens/MobileBrowse'
import MobileDetail from './screens/MobileDetail'

const Loading = styled.div`
  display: flex;
  align-items: center;
  justify-content: center;
  flex: 1;
  font-family: ${t.fontDisplay};
  font-size: 16px;
  color: ${t.mute};
`

const Placeholder = styled.div`
  display: flex;
  align-items: center;
  justify-content: center;
  flex: 1;
  font-family: ${t.fontDisplay};
  font-size: 16px;
  color: ${t.mute};
`

import type { ClusterServerInfo } from '../protocol'

type Props = {
  maps: BrowseMapEntry[]
  loading: boolean
  onPlay: (mapName: string) => void
  onJoinServer: (serverName: string) => void
  onClose: () => void
  isInGame: boolean
  initialView?: 'pause' | 'browse'
  servers?: ClusterServerInfo[]
}

export default function Menu({ maps, loading, onPlay, onJoinServer, onClose, isInGame, initialView, servers = [] }: Props) {
  const { view, activeTab, switchTab, openDetail, openAuthor, goBack } = useMenuNav('home')
  const [searchQuery, setSearchQuery] = React.useState('')
  const [showPause, setShowPause] = React.useState(initialView === 'pause')

  // If entering in-game, show pause overlay
  React.useEffect(() => {
    if (initialView === 'pause') {
      setShowPause(true)
    }
  }, [initialView])

  const handlePlay = React.useCallback((name: string) => {
    onPlay(name)
  }, [onPlay])

  const handleOpenMap = React.useCallback((map: BrowseMapEntry) => {
    openDetail(map.name)
  }, [openDetail])

  // Loading state
  if (loading) {
    return (
      <MenuOverlay>
        <Loading>Loading catalog…</Loading>
      </MenuOverlay>
    )
  }

  // Pause overlay (in-game)
  if (showPause && isInGame) {
    return (
      <MenuOverlay style={{
        background: 'linear-gradient(180deg, rgba(20,17,13,0.55) 0%, rgba(20,17,13,0.75) 100%)',
      }}>
        <TopBar
          active={activeTab}
          onTab={(tab) => {
            setShowPause(false)
            switchTab(tab)
          }}
          isInGame
          mapCount={maps.length}
          searchQuery={searchQuery}
          onSearch={setSearchQuery}
        />
        <PauseScreen
          onResume={onClose}
          onDisconnect={() => {
            // Try to execute disconnect command if available
            try {
              if (typeof BananaBread !== 'undefined' && BananaBread.execute) {
                BananaBread.execute('disconnect')
              }
            } catch (e) {}
            onClose()
          }}
          onBrowse={() => {
            setShowPause(false)
            switchTab('browse')
          }}
          mapCount={maps.length}
        />
      </MenuOverlay>
    )
  }

  // Mobile layout
  if (BROWSER.isMobile) {
    // Mobile detail
    if (view.screen === 'detail') {
      const map = maps.find(m => m.name === view.mapId)
      if (map) {
        return (
          <MenuOverlay>
            <MobileDetail
              map={map}
              onBack={goBack}
              onPlay={handlePlay}
              onOpenAuthor={openAuthor}
            />
          </MenuOverlay>
        )
      }
    }

    return (
      <MenuOverlay>
        <MobileBrowse
          maps={maps}
          activeTab={activeTab}
          onTab={switchTab}
          onOpenMap={handleOpenMap}
          onPlayMap={handlePlay}
          searchQuery={searchQuery}
          onSearch={setSearchQuery}
        />
      </MenuOverlay>
    )
  }

  // Desktop layout
  const renderScreen = () => {
    if (view.screen === 'detail') {
      const map = maps.find(m => m.name === view.mapId)
      if (map) {
        return (
          <DetailScreen
            map={map}
            allMaps={maps}
            onBack={goBack}
            onPlay={handlePlay}
            onOpenAuthor={openAuthor}
            onOpenMap={handleOpenMap}
          />
        )
      }
    }

    if (view.screen === 'author') {
      return (
        <AuthorScreen
          authorName={view.authorName}
          allMaps={maps}
          onBack={goBack}
          onOpenMap={handleOpenMap}
          onPlayMap={handlePlay}
        />
      )
    }

    // Tab screens
    switch (activeTab) {
      case 'home':
        return (
          <HomeScreen
            maps={maps}
            servers={servers}
            onOpenMap={handleOpenMap}
            onPlayMap={handlePlay}
            onSeeAll={() => switchTab('browse')}
            onJoinServer={onJoinServer}
          />
        )
      case 'browse':
        return (
          <BrowseScreen
            maps={maps}
            searchQuery={searchQuery}
            onOpenMap={handleOpenMap}
            onPlayMap={handlePlay}
          />
        )
      case 'mods':
        return <Placeholder>Mods — coming soon</Placeholder>
      case 'servers':
        return <Placeholder>Servers — coming soon</Placeholder>
      case 'settings':
        return <Placeholder>Settings — coming soon</Placeholder>
      default:
        return null
    }
  }

  return (
    <MenuOverlay>
      <TopBar
        active={activeTab}
        onTab={switchTab}
        isInGame={isInGame}
        mapCount={maps.length}
        searchQuery={searchQuery}
        onSearch={setSearchQuery}
      />
      {renderScreen()}
    </MenuOverlay>
  )
}
