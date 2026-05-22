import * as R from 'ramda'

import { BROWSER } from './utils'

export type Configuration = {
  assets: string[]
  catalog: string
  servers: string[]
  proxy: string
  menuOptions: string
}

export let CONFIG: Configuration = {
  assets: [],
  catalog: '',
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

function applyConfig() {
  CONFIG.assets = R.chain((v): string[] => {
    if (v.startsWith('mobile:')) {
      return BROWSER.isMobile ? [fillAssetHost(v.slice(7))] : []
    }
    if (v.startsWith('desktop:')) {
      return !BROWSER.isMobile ? [fillAssetHost(v.slice(8))] : []
    }
    return [fillAssetHost(v)]
  }, CONFIG.assets)
  if (CONFIG.catalog) {
    CONFIG.catalog = fillHost(CONFIG.catalog)
  }
  CONFIG.servers = R.map((v) => fillHost(v), CONFIG.servers)
  CONFIG.proxy = fillHost(CONFIG.proxy)
}

// true if config was available synchronously (script tag loaded before module)
export let configAvailable = false

const config = getInjected()
if (config != null) {
  CONFIG = config
  applyConfig()
  configAvailable = true
}

// Fallback for dev: fetch config from Go server if script tag failed.
// Only used when configAvailable is false.
export async function waitForConfig(): Promise<void> {
  if (configAvailable) return

  for (let i = 0; i < 30; i++) {
    try {
      const resp = await fetch('/api/client-config.js')
      if (resp.ok) {
        const text = await resp.text()
        const fn = new Function(text + '; return INJECTED_SOUR_CONFIG;')
        CONFIG = fn()
        applyConfig()
        configAvailable = true
        return
      }
    } catch (_) {
      // Server not ready yet
    }
    await new Promise((r) => setTimeout(r, 1000))
  }

  console.error('no configuration provided — could not reach server')
}
