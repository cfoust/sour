import './fonts/fonts.css'
import * as React from 'react'
import { createRoot } from 'react-dom/client'
import { ChakraProvider, extendTheme } from '@chakra-ui/react'
import type { ThemeConfig } from '@chakra-ui/react'
import start from './unsafe-startup'
import { configAvailable, waitForConfig } from './config'
import App from './index'

start()

const colors = {
  brand: {
    900: '#1a365d',
    800: '#153e75',
    700: '#2a69ac',
  },
}

const config: ThemeConfig = {
  initialColorMode: 'dark',
  useSystemColorMode: false,
}

const theme = extendTheme({ colors, config })

function Root() {
  const [ready, setReady] = React.useState(configAvailable)
  React.useEffect(() => {
    if (!configAvailable) {
      waitForConfig().then(() => setReady(true))
    }
  }, [])
  if (!ready) return null
  return (
    <ChakraProvider theme={theme}>
      <App />
    </ChakraProvider>
  )
}

createRoot(document.getElementById('root')!).render(<Root />)
