import styled from '@emotion/styled'
import start from './unsafe-startup'
import * as React from 'react'
import ReactDOM from 'react-dom'
import {
  Center,
  ChakraProvider,
  Button,
  extendTheme,
  Flex,
  Box,
  VStack,
  Heading,
  Spacer,
} from '@chakra-ui/react'

start()

const colors = {
  brand: {
    900: '#1a365d',
    800: '#153e75',
    700: '#2a69ac',
  },
}

const theme = extendTheme({ colors })

const OuterContainer = styled.div`
  width: 100%;
  height: 100%;
  position: absolute;
`

const GameContainer = styled.div`
  width: 100%;
  height: 100%;
  position: absolute;
  z-index: 0;
`

const ControlContainer = styled.div`
  width: 100%;
  height: 100%;
  position: absolute;
  z-index: 1;
`

function App() {
  const [showGame, setShowGame] = React.useState<boolean>(false)

  React.useLayoutEffect(() => {
    const box = document.getElementById('box')
    if (box == null) return
    Module.desiredWidth = box.clientWidth
    Module.desiredHeight = box.clientHeight
  }, [])

  return (
    <OuterContainer>
      <GameContainer id="box">
        <canvas
          className="game"
          id="canvas"
          ref={(canvas) => (Module.canvas = canvas)}
          onContextMenu={(event) => event.preventDefault()}
        ></canvas>
      </GameContainer>
      {!showGame && (
        <ControlContainer>
          <Box w="100%" h="100%" background="yellow.400">
            <Flex align="center" justify="center">
              <VStack paddingTop="20%">
                <Heading>🍋Sour</Heading>
                <Button onClick={() => setShowGame(true)}>Join game</Button>
              </VStack>
            </Flex>
          </Box>
        </ControlContainer>
      )}
    </OuterContainer>
  )
}

ReactDOM.render(
  <ChakraProvider theme={theme}>
    <App />
  </ChakraProvider>,
  document.getElementById('root')
)
