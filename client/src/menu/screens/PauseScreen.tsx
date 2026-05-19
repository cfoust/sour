import * as React from 'react'
import styled from '@emotion/styled'
import { t } from '../theme'
import { Btn } from '../styled'
import Icon from '../Icon'
import Scoreboard, { PlayerEntry } from '../Scoreboard'

const Backdrop = styled.div`
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 60px;
  background: linear-gradient(
    180deg,
    rgba(20, 17, 13, 0.55) 0%,
    rgba(20, 17, 13, 0.75) 100%
  );
`

const Card = styled.div`
  background: ${t.bgCard};
  border: 1px solid ${t.line};
  border-radius: 22px;
  padding: 30px 32px;
  width: 460px;
  max-width: 100%;
  display: flex;
  flex-direction: column;
  gap: 18px;
  box-shadow: ${t.modalShadow};
`

const Top = styled.div`
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  padding-bottom: 16px;
  border-bottom: 1px solid ${t.line};
`

const StatusLabel = styled.div`
  font-family: ${t.fontDisplay};
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: ${t.mute};
  display: inline-flex;
  align-items: center;
  gap: 8px;

  &::before {
    content: '';
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: ${t.accent};
    box-shadow: 0 0 8px ${t.accent};
  }
`

const GameName = styled.div`
  font-family: ${t.fontDisplay};
  font-size: 26px;
  font-weight: 800;
  letter-spacing: -0.025em;
  margin-top: 6px;
  line-height: 1.1;
  color: ${t.bone};
`

const Actions = styled.div`
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
`

const ArchiveLink = styled.button`
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 16px;
  border: 1px solid ${t.line};
  border-radius: 12px;
  cursor: pointer;
  background: ${t.surface};
  box-shadow: 0 2px 0 rgba(40, 28, 8, 0.05);

  &:hover { background: ${t.surface2}; }
`

const ArchiveLeft = styled.div`
  display: flex;
  flex-direction: column;
  gap: 2px;
`

const ArchiveLinkLabel = styled.span`
  font-family: ${t.fontDisplay};
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: ${t.mute};
`

const ArchiveLinkTitle = styled.span`
  font-family: ${t.fontDisplay};
  font-size: 16px;
  font-weight: 700;
  letter-spacing: -0.01em;
  color: ${t.bone};
`

const Arrow = styled.span`
  color: ${t.accent3};
  font-family: ${t.fontUi};
  font-weight: 700;
`

type Props = {
  mapName?: string
  onResume: () => void
  onDisconnect: () => void
  onBrowse: () => void
  mapCount?: number
}

export default function PauseScreen({ mapName, onResume, onDisconnect, onBrowse, mapCount }: Props) {
  return (
    <Backdrop>
      <Card>
        <Top>
          <div>
            <StatusLabel>Paused · in game</StatusLabel>
            {mapName && <GameName>{mapName}</GameName>}
          </div>
        </Top>

        <Actions>
          <Btn $primary $full $large onClick={onResume}>
            <Icon name="play" size={12} /> Resume game
          </Btn>
          <Btn>
            <Icon name="users" size={12} /> Switch team
          </Btn>
          <Btn>
            <Icon name="flag" size={12} /> Vote map
          </Btn>
          <Btn $full>
            <Icon name="settings" size={12} /> Settings
          </Btn>
          <Btn $full $danger onClick={onDisconnect}>
            Disconnect
          </Btn>
        </Actions>

        <ArchiveLink onClick={onBrowse}>
          <ArchiveLeft>
            <ArchiveLinkLabel>
              Maps{mapCount ? ` · ${mapCount.toLocaleString()}` : ''}
            </ArchiveLinkLabel>
            <ArchiveLinkTitle>Browse while you wait</ArchiveLinkTitle>
          </ArchiveLeft>
          <Arrow>→</Arrow>
        </ArchiveLink>
      </Card>
    </Backdrop>
  )
}
