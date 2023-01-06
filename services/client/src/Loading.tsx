import * as React from 'react'

import { Flex, VStack, HStack, Heading, Progress } from '@chakra-ui/react'

import type { GameState, DownloadingState } from './types'
import { GameStateType, DownloadingType } from './types'

type Props = {
  state: GameState
}

function Downloading(props: { state: DownloadingState }) {
  const {
    state: { downloadedBytes, totalBytes, downloadType },
  } = props

  return (
    <VStack>
      <Heading>
        Loading {DownloadingType[downloadType].toLowerCase()} data...
      </Heading>
      <Progress
        colorScheme="yellow"
        hasStripe
        isAnimated
        width={200}
        value={(downloadedBytes / totalBytes) * 100}
      />
    </VStack>
  )
}

export default function StatusOverlay(props: Props) {
  const { state } = props
  return (
    <Flex align="center" justify="center">
      <VStack paddingTop="20%">
        {state.type === GameStateType.PageLoading && (
          <Heading>Initializing...</Heading>
        )}
        {state.type === GameStateType.Downloading && (
          <Downloading state={state} />
        )}
        {state.type === GameStateType.Running && (
          <Heading>Waiting for game to start...</Heading>
        )}
        {state.type === GameStateType.MapChange && (
          <Heading>Loading map {state.map}...</Heading>
        )}
        {state.type === GameStateType.GameError && (
          <Heading>There was an unknown error with the game.</Heading>
        )}
      </VStack>
    </Flex>
  )
}
