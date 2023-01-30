import * as R from 'ramda'

import { BROWSER } from './utils'

export type Configuration = {
  assets: string[]
  clusters: string[]
  proxy: string

  auth: {
    enabled: boolean
    domain: string
    redirectURI: string
    authorizationURL: string
  }
}

export let CONFIG: Configuration = {
  assets: [],
  clusters: [],
  proxy: '',
  auth: {
    enabled: false,
    domain: '',
    authorizationURL: '',
    redirectURI: '',
  },
}

const REPLACED = {
  ORIGIN: '#origin',
  HOST: '#host',
  PROTOCOL: '#protocol',
}

function fillHost(url: string): string {
  return url
    .replace(REPLACED.ORIGIN, window.location.origin)
    .replace(REPLACED.HOST, window.location.host)
    .replace(REPLACED.PROTOCOL, window.location.protocol)
}

function fillAssetHost(url: string): string {
  const newHost = fillHost(url)

  // Don't cache asset sources pointing to this host
  if (url.includes(REPLACED.HOST) || url.includes(REPLACED.ORIGIN)) {
    return `!${newHost}`
  }

  return newHost
}

function getInjected(): Maybe<Configuration> {
  try {
    const injected = INJECTED_SOUR_CONFIG
    // This will never run if INJECTED_SOUR_CONFIG is not defined
    return injected
  } catch (e) {
    return null
  }
}

function init() {
  const config = getInjected()
  if (config != null) {
    CONFIG = config
  } else {
    const configStr = process.env.SOUR_CONFIG
    if (configStr == null) {
      new Error('no configuration provided')
      return
    }

    CONFIG = JSON.parse(configStr)
  }

  CONFIG.assets = R.chain((v): string[] => {
    if (v.startsWith('mobile:')) {
      return BROWSER.isMobile ? [fillAssetHost(v.slice(7))] : []
    }
    return [fillAssetHost(v)]
  }, CONFIG.assets)
  CONFIG.clusters = R.map((v) => fillHost(v), CONFIG.clusters)
  CONFIG.proxy = fillHost(CONFIG.proxy)
}

init()
