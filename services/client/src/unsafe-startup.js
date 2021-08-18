export default function start() {
  var statusElement = document.getElementById('status')
  var progressElement = document.getElementById('progress')
  var spinnerElement = document.getElementById('spinner')

  Module = {
    ...Module,
    preRun: [],
    postRun: [],
    print: (function () {
      return function (text) {
        console.log(text)
      }
    })(),
    printErr: function (text) {
      text = Array.prototype.slice.call(arguments).join(' ')
      if (0) {
        // XXX disabled for safety typeof dump == 'function') {
        dump(text + '\n') // fast, straight to the real console
      } else {
        console.error(text)
      }
    },
    canvas: (function () {
      var canvas = document.getElementById('canvas')

      // As a default initial behavior, pop up an alert when webgl context is lost. To make your
      // application robust, you may want to override this behavior before shipping!
      // See http://www.khronos.org/registry/webgl/specs/latest/1.0/#5.15.2
      canvas.addEventListener(
        'webglcontextlost',
        function (e) {
          alert('WebGL context lost. You will need to reload the page.')
          e.preventDefault()
        },
        false
      )

      canvas.addEventListener('click', function () {
        canvas.requestPointerLock()
      })

      return canvas
    })(),
    setStatus: function (text) {
      if (!Module.setStatus.last)
        Module.setStatus.last = { time: Date.now(), text: '' }
      if (text === Module.setStatus.text) return
      var m = text.match(/([^(]+)\((\d+(\.\d+)?)\/(\d+)\)/)
      var now = Date.now()
      if (m && now - Date.now() < 30) return // if this is a progress update, skip it if too soon
      if (m) {
        text = m[1]
        progressElement.value = parseInt(m[2]) * 100
        progressElement.max = parseInt(m[4]) * 100
        progressElement.hidden = false
        spinnerElement.hidden = false
      } else {
        progressElement.value = null
        progressElement.max = null
        progressElement.hidden = true
        if (!text) spinnerElement.style.display = 'none'
      }
      statusElement.innerHTML = text
    },
    totalDependencies: 0,
    monitorRunDependencies: function (left) {
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
        BananaBread.execute('screenres ' + 640 + ' ' + 480)
      }
    },
  }
  Module.setStatus('Downloading...')
  window.onerror = function () {
    Module.setStatus('Exception thrown, see JavaScript console')
    spinnerElement.style.display = 'none'
    Module.setStatus = function (text) {
      if (text) Module.printErr('[post-exception status] ' + text)
    }
  }

  const box = document.getElementById("box");
  Module.desiredWidth = box.clientWidth;
  Module.desiredHeight = box.clientHeight
  window.onresize = () => {
    Module.setCanvasSize(window.innerWidth, window.innerHeight)
  }

  Module.autoexec = function () {
    Module.setStatus("");
  };
  Module.postLoadWorld = function () {
    Module.tweakDetail();
    BananaBread.execute("sensitivity 10");
    BananaBread.execute("clearconsole");
  };
}
