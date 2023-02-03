import { CONFIG } from './config'

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
      //{ "mrfixit", "mrfixit/blue", "mrfixit/red", "mrfixit/hudguns", NULL, "mrfixit/horns", { "mrfixit/armor/blue", "mrfixit/armor/green", "mrfixit/armor/yellow" }, "mrfixit", "mrfixit_blue", "mrfixit_red", true },
      BananaBread.setPlayerModelInfo(
        'mrfixit',
        'mrfixit/blue',
        'mrfixit/red',
        'mrfixit/hudguns',
        0,
        'mrfixit/horns',
        'mrfixit/armor/blue',
        'mrfixit/armor/green',
        'mrfixit/armor/yellow',
        'mrfixit',
        'mrfixit_blue',
        'mrfixit_red',
        true
      )
      //{ "snoutx10k", "snoutx10k/blue", "snoutx10k/red", "snoutx10k/hudguns", NULL, "snoutx10k/wings", { "snoutx10k/armor/blue", "snoutx10k/armor/green", "snoutx10k/armor/yellow" }, "snoutx10k", "snoutx10k_blue", "snoutx10k_red", true },
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
      //{ "ogro2", "ogro2/blue", "ogro2/red", "mrfixit/hudguns", NULL, "ogro2/quad", { "ogro2/armor/blue", "ogro2/armor/green", "ogro2/armor/yellow" }, "ogro", "ogro_blue", "ogro_red", true },
      BananaBread.setPlayerModelInfo(
        'ogro2',
        'ogro2/blue',
        'ogro2/red',
        'mrfixit/hudguns',
        0,
        'ogro2/quad',
        'ogro2/armor/blue',
        'ogro2/armor/green',
        'ogro2/armor/yellow',
        'ogro',
        'ogro2_blue',
        'ogro2_red',
        true
      )

      //{ "inky", "inky/blue", "inky/red", "inky/hudguns", NULL, "inky/quad", { "inky/armor/blue", "inky/armor/green", "inky/armor/yellow" }, "inky", "inky_blue", "inky_red", true },
      BananaBread.setPlayerModelInfo(
        'inky',
        'inky/blue',
        'inky/red',
        'inky/hudguns',
        0,
        'inky/quad',
        'inky/armor/blue',
        'inky/armor/green',
        'inky/armor/yellow',
        'ogro',
        'inky_blue',
        'inky_red',
        true
      )

      //{ "captaincannon", "captaincannon/blue", "captaincannon/red", "captaincannon/hudguns", NULL, "captaincannon/quad", { "captaincannon/armor/blue", "captaincannon/armor/green", "captaincannon/armor/yellow" }, "captaincannon", "captaincannon_blue", "captaincannon_red", true }
      BananaBread.setPlayerModelInfo(
        'captaincannon',
        'captaincannon/blue',
        'captaincannon/red',
        'captaincannon/hudguns',
        0,
        'captaincannon/quad',
        'captaincannon/armor/blue',
        'captaincannon/armor/green',
        'captaincannon/armor/yellow',
        'ogro',
        'captaincannon_blue',
        'captaincannon_red',
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
      if (
        text.startsWith('Preparing...') ||
        text.startsWith('All downloads') ||
        text.startsWith('Please wait...') ||
        text.length === 0
      )
        return
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
    addRunDependency: (file) => {
      console.log(`add ${file}`)
    },
  }

  window.onerror = function (_, __, ___, ____, error) {
    console.log(error)
    return true
  }

  Module.addRunDependency = (file) => {
    console.log(`add ${file}`)
  }

  Module['removeRunDependency'] = (file) => {
    console.log(`remove ${file}`)
  }

  Module.setStatus('Downloading...')

  Module.autoexec = function () {
    Module.setStatus('')
  }
}
