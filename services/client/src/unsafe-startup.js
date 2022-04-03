export default function start() {
  Module = {
    ...Module,
    // This makes it so Emscripten's WS connections just go to this regardless
    // of the host
    websocket: {
      url: (addr, port) => {
        const { protocol, host } = window.location
        const prefix = `${
          protocol === 'https:' ? 'wss://' : 'ws:/'
        }${host}/service/proxy/`

        return addr === 'sour' ? prefix : `${prefix}u/${addr}:${port}`
      },
    },
    setPlayerModels: function () {
      BananaBread.setPlayerModelInfo(
        'snoutx10k',
        'snoutx10k/blue',
        'snoutx10k/red',
        'snoutx10k/hudguns',
        0,
        'snoutx10k/wings',
        'snoutx10k/armor/blue',
        'snoutx10k/armor/green',
        'snoutx10k/armor/yellow',
        'snoutx10k',
        'snoutx10k_blue',
        'snoutx10k_red',
        true
      )
    },
    tweakDetail: () => {},
    loadDefaultMap: () => {
      const { innerWidth: width, innerHeight: height } = window
      Module.setCanvasSize(width, height)
      BananaBread.execute(`screenres ${width} ${height}`)
    },
    locateFile: (file) => {
      if (file.endsWith('.data')) {
        // Strip the hash
        const stripped = file.split('.').slice(1).join('.')
        return `${ASSET_PREFIX}${stripped}`
      }
      if (file.endsWith('.wasm')) return `/game/${file}`
      return null
    },
    preRun: [],
    postRun: [],
    printErr: function (text) {
      if (
        // These two happen a lot while playing and they don't mean anything.
        text.startsWith('Cannot find preloaded audio') ||
        text.startsWith("Couldn't find file for:")
      )
        return
      console.error(text)
    },
    setStatus: function (text) {
      console.log(text)
    },
    totalDependencies: 0,
    monitorRunDependencies: function (left) {
      Module.runDependencies = left
      this.totalDependencies = Math.max(this.totalDependencies, left)
      Module.setStatus(
        left
          ? 'Preparing... (' +
              (this.totalDependencies - left) +
              '/' +
              this.totalDependencies +
              ')'
          : 'All downloads complete.'
      )
    },
    goFullScreen: function () {
      Module.requestFullScreen(true, false)
    },
    onFullScreen: function (isFullScreen) {
      if (isFullScreen) {
        BananaBread.execute('screenres ' + screen.width + ' ' + screen.height)
      } else {
        const { innerWidth: width, innerHeight: height } = window
        BananaBread.execute(`screenres ${width} ${height}`)
      }
    },
  }

  window.onerror = function (_, __, ___, ____, error) {
    console.log(error)
    return true
  }

  Module['removeRunDependency'] = null

  Module.setStatus('Downloading...')

  Module.autoexec = function () {
    Module.setStatus('')
  }
}
