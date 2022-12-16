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

function init() {
  const config = process.env.SOUR_CONFIG
  if (config == null) {
    new Error('no configuration provided')
    return
  }

  CONFIG = JSON.parse(config).client

  CONFIG.assets = R.map((v) => fillHost(v), CONFIG.assets)
  CONFIG.clusters = R.map((v) => fillHost(v), CONFIG.clusters)
  CONFIG.proxy = fillHost(CONFIG.proxy)
}

init()
