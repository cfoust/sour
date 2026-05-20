import * as React from 'react'
import styled from '@emotion/styled'
import { css, keyframes } from '@emotion/react'
import { t } from '../theme'
import { Btn } from '../styled'
import Icon from '../Icon'
import ModePill from '../ModePill'

const gamePulse = keyframes`
  0%   { box-shadow: 0 0 0 0 rgba(255, 214, 10, 0.85); }
  100% { box-shadow: 0 0 0 10px rgba(255, 214, 10, 0); }
`

const Wrap = styled.div`
  flex: 1 1 auto;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px;
`

const Card = styled.div`
  width: 440px;
  background: ${t.bgCard};
  border: 1px solid ${t.line};
  border-radius: 20px;
  padding: 28px 28px 22px;
  display: flex;
  flex-direction: column;
  gap: 16px;
  box-shadow: 0 10px 0 rgba(40, 28, 8, 0.07), 0 28px 56px rgba(40, 28, 8, 0.2);
`

const Head = styled.div`
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding-bottom: 14px;
  border-bottom: 1px dashed ${t.line};
`

const Eyebrow = styled.div`
  font-family: ${t.fontDisplay};
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.22em;
  text-transform: uppercase;
  color: ${t.accent3};
  display: inline-flex;
  align-items: center;
  gap: 9px;
`

const Pulse = styled.span`
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: ${t.accent};
  box-shadow: 0 0 0 0 ${t.accent};
  animation: ${gamePulse} 1.4s ease-out infinite;
`

const Title = styled.h2`
  font-family: ${t.fontDisplay};
  font-size: 34px;
  font-weight: 900;
  letter-spacing: -0.03em;
  color: ${t.bone};
  line-height: 1;
  margin: 2px 0 0;
`

const Sub = styled.div`
  font-family: ${t.fontUi};
  font-size: 13px;
  color: ${t.mute};
  display: inline-flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  margin-top: 8px;

  strong { color: ${t.bone2}; font-weight: 600; }
`

const Dot = styled.span`
  color: ${t.dim};
`

const ResumeBtn = styled.button`
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 14px 18px;
  background: ${t.accent};
  color: ${t.bone};
  border: none;
  border-radius: 14px;
  cursor: pointer;
  font-family: ${t.fontDisplay};
  font-size: 17px;
  font-weight: 800;
  letter-spacing: -0.015em;
  box-shadow: 0 4px 0 ${t.accent3};
  transition: transform 0.08s, box-shadow 0.08s, background 0.12s;

  &:hover { background: ${t.accent2}; }
  &:active { transform: translateY(2px); box-shadow: 0 2px 0 ${t.accent3}; }
`

const ResumeHint = styled.span`
  margin-left: auto;
  font-family: ${t.fontMono};
  font-size: 10px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  background: ${t.bone};
  color: ${t.accent};
  padding: 3px 8px;
  border-radius: 6px;
  font-weight: 700;
`

const ActionGrid = styled.div`
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
`

const DisconnectBtn = styled.button`
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 11px 14px;
  background: transparent;
  border: 1px solid ${t.line2};
  border-radius: 12px;
  color: ${t.coral2};
  font-family: ${t.fontDisplay};
  font-weight: 600;
  font-size: 13px;
  cursor: pointer;
  margin-top: 4px;

  &:hover { background: rgba(255, 85, 112, 0.08); border-color: ${t.coral}; }
`

type Props = {
  mapName?: string
  serverName?: string
  serverMode?: string
  onResume: () => void
  onDisconnect: () => void
  onVoteMap: () => void
  onSwitchTeam: () => void
  onToggleSpectator: () => void
  onMaster: () => void
}

export default function PauseScreen({
  mapName, serverName, serverMode,
  onResume, onDisconnect, onVoteMap, onSwitchTeam, onToggleSpectator, onMaster
}: Props) {
  return (
    <Wrap>
      <Card>
        <Head>
          <Eyebrow>
            <Pulse />
            Paused
          </Eyebrow>
          {mapName && <Title>{mapName}</Title>}
          <Sub>
            {serverName && <strong>{serverName}</strong>}
            {serverName && serverMode && <Dot>·</Dot>}
            {serverMode && <ModePill mode={serverMode.toLowerCase()} primary />}
          </Sub>
        </Head>

        <ResumeBtn onClick={onResume}>
          <Icon name="play" size={14} />
          <span>Resume</span>
          <ResumeHint>esc</ResumeHint>
        </ResumeBtn>

        <ActionGrid>
          <Btn onClick={onVoteMap} style={{ justifyContent: 'center', padding: '11px 14px', fontSize: '13.5px' }}>
            <Icon name="flag" size={12} /> Vote map…
          </Btn>
          <Btn onClick={onSwitchTeam} style={{ justifyContent: 'center', padding: '11px 14px', fontSize: '13.5px' }}>
            <Icon name="users" size={12} /> Switch team
          </Btn>
          <Btn onClick={onToggleSpectator} style={{ justifyContent: 'center', padding: '11px 14px', fontSize: '13.5px' }}>
            <Icon name="crosshair" size={12} /> Spectator
          </Btn>
          <Btn onClick={onMaster} style={{ justifyContent: 'center', padding: '11px 14px', fontSize: '13.5px' }}>
            <Icon name="settings" size={12} /> Master…
          </Btn>
        </ActionGrid>

        <DisconnectBtn onClick={onDisconnect}>
          Disconnect from server
        </DisconnectBtn>
      </Card>
    </Wrap>
  )
}
