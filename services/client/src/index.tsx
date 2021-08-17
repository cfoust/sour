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
    <Box w="100%" h="100%" background="yellow.400">
      <Flex h="100%" align="center" justify="center">
        <VStack>
          <Heading>🍋Sour</Heading>
          <Button>Join game</Button>
        </VStack>
      </Flex>
    </Box>
  )
}

ReactDOM.render(
  <ChakraProvider theme={theme}>
    <App />
  </ChakraProvider>,
  document.getElementById('root')
)
