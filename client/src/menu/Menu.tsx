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

const PausedOverlay = styled.div`
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  color: ${t.bone};
  font-family: ${t.fontUi};
  font-size: 14.5px;
  line-height: 1.5;
  letter-spacing: -0.005em;
  -webkit-font-smoothing: antialiased;
  backdrop-filter: blur(6px) saturate(0.9);
  -webkit-backdrop-filter: blur(6px) saturate(0.9);
  background: rgba(0, 0, 0, 0.35);
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
  currentMap?: string
  currentServer?: string
  playerName?: string
  playerModel?: string
  onNameChange?: (name: string) => void
  onModelChange?: (model: string) => void
}

function execCommand(cmd: string) {
  try {
    if (typeof BananaBread !== 'undefined' && BananaBread.execute) {
      BananaBread.execute(cmd)
    }
  } catch (e) {}
}

export default function Menu({
  maps, loading, onPlay, onJoinServer, onClose, isInGame, initialView,
  servers = [], currentMap, currentServer, playerName, playerModel,
  onNameChange, onModelChange
}: Props) {
  const { view, activeTab, switchTab, openDetail, openAuthor, goBack } = useMenuNav('home')
  const [searchQuery, setSearchQuery] = React.useState('')
  const [showPause, setShowPause] = React.useState(initialView === 'pause')

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

  const handleTab = React.useCallback((tab: MenuTab) => {
    if (tab === 'game') {
      setShowPause(true)
    } else {
      setShowPause(false)
      switchTab(tab)
    }
  }, [switchTab])

  // Loading state
  if (loading) {
    return (
      <MenuOverlay>
        <Loading>Loading catalog…</Loading>
      </MenuOverlay>
    )
  }

  // Build pause action callbacks
  const pauseActions = {
    onResume: onClose,
    onDisconnect: () => {
      execCommand('disconnect')
      // Module.onDisconnect will set browsing=true and clear game state
    },
    onVoteMap: () => {
      onClose()
      setTimeout(() => execCommand('showgui gamemode'), 50)
    },
    onSwitchTeam: () => {
      execCommand('if (strcmp (getteam) "good") [team evil] [team good]')
      onClose()
    },
    onToggleSpectator: () => {
      execCommand('spectator (! (isspectator (getclientnum)))')
      onClose()
    },
    onMaster: () => {
      onClose()
      setTimeout(() => execCommand('showgui master'), 50)
    },
  }

  // Determine the overlay component
  const Overlay = isInGame ? PausedOverlay : MenuOverlay

  // Pause overlay (in-game, Game tab active)
  if (showPause && isInGame) {
    return (
      <Overlay>
        <TopBar
          active={'game'}
          onTab={handleTab}
          isInGame
          mapCount={maps.length}
          searchQuery={searchQuery}
          onSearch={setSearchQuery}
          playerName={playerName}
          playerModel={playerModel}
          onNameChange={onNameChange}
          onModelChange={onModelChange}
        />
        <PauseScreen
          mapName={currentMap}
          serverName={currentServer}
          {...pauseActions}
        />
      </Overlay>
    )
  }

  // Mobile layout
  if (BROWSER.isMobile) {
    if (view.screen === 'detail') {
      const map = maps.find(m => m.name === view.mapId)
      if (map) {
        return (
          <Overlay>
            <MobileDetail
              map={map}
              onBack={goBack}
              onPlay={handlePlay}
              onOpenAuthor={openAuthor}
            />
          </Overlay>
        )
      }
    }

    return (
      <Overlay>
        <MobileBrowse
          maps={maps}
          activeTab={activeTab}
          onTab={switchTab}
          onOpenMap={handleOpenMap}
          onPlayMap={handlePlay}
          searchQuery={searchQuery}
          onSearch={setSearchQuery}
        />
      </Overlay>
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
    <Overlay>
      <TopBar
        active={activeTab}
        onTab={handleTab}
        isInGame={isInGame}
        mapCount={maps.length}
        searchQuery={searchQuery}
        onSearch={setSearchQuery}
        playerName={playerName}
        playerModel={playerModel}
        onNameChange={onNameChange}
        onModelChange={onModelChange}
      />
      {renderScreen()}
    </Overlay>
  )
}
