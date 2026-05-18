import * as R from 'ramda'

import { BROWSER } from './utils'

export type Configuration = {
  assets: string[]
  servers: string[]
  proxy: string
  menuOptions: string
}

export let CONFIG: Configuration = {
  assets: [],
  servers: [],
  proxy: '',
  menuOptions: '',
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

  // Rebase assets to current path if pointing at same-origin /assets
  // This makes assets work when the app is hosted under a subpath (e.g., /sour/)
  const rebase = (absolute: string): string => {
    try {
      const u = new URL(absolute, window.location.href)
      if (
        u.origin === window.location.origin &&
        u.pathname.startsWith('/assets/')
      ) {
        const baseAssetsPath = new URL('assets/', window.location.href).pathname
        u.pathname = baseAssetsPath + u.pathname.slice('/assets/'.length)
        return u.toString()
      }
      return u.toString()
    } catch (_e) {
      return absolute
    }
  }

  const rebased = rebase(newHost)

  // Don't cache asset sources pointing to this host
  if (url.includes(REPLACED.HOST) || url.includes(REPLACED.ORIGIN)) {
    return `!${rebased}`
  }

  return rebased
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
    if (v.startsWith('desktop:')) {
      return !BROWSER.isMobile ? [fillAssetHost(v.slice(8))] : []
    }
    return [fillAssetHost(v)]
  }, CONFIG.assets)
  CONFIG.servers = R.map((v) => fillHost(v), CONFIG.servers)
  CONFIG.proxy = fillHost(CONFIG.proxy)
}

init()
