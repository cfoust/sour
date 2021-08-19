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

function App() {
  return (
    <div>
      <div id="box" className="container">
        <div className="stuff">
          <div className="spinner" id="spinner"></div>
          <div className="emscripten" id="status">
            Downloading...
          </div>

          <div className="emscripten">
            <progress
              value="0"
              max="100"
              id="progress"
              hidden={true}
            ></progress>
          </div>
        </div>

        <canvas
          className="game"
          id="canvas"
          ref={(canvas) => (Module.canvas = canvas)}
          onContextMenu={(event) => event.preventDefault()}
        ></canvas>
      </div>
      <Box w="100%" h="100%" background="yellow.400">
        <Flex align="center" justify="center">
          <VStack paddingTop="20%">
            <Heading>🍋Sour</Heading>
            <Button onClick={() => {}}>Join game</Button>
          </VStack>
        </Flex>
      </Box>
    </div>
  )
}

ReactDOM.render(
  <ChakraProvider theme={theme}>
    <App />
  </ChakraProvider>,
  document.getElementById('root')
)
