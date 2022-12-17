import * as R from 'ramda'

export type Configuration = {
  assets: string[]
  clusters: string[]
  proxy: string
}

export let CONFIG: Configuration = {
  assets: [],
  clusters: [],
  proxy: '',
}

function fillHost(url: string): string {
  return url
    .replace('#origin', window.location.origin)
    .replace('#host', window.location.host)
    .replace('#protocol', window.location.protocol)
}

function getInjected(): Maybe<Configuration> {
  try {
    const injected = INJECTED_SOUR_CONFIG
    return injected.client
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

    CONFIG = JSON.parse(configStr).client
  }

  CONFIG.assets = R.map((v) => fillHost(v), CONFIG.assets)
  CONFIG.clusters = R.map((v) => fillHost(v), CONFIG.clusters)
  CONFIG.proxy = fillHost(CONFIG.proxy)
}

init()
