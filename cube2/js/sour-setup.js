var Module = {
  // If the url has 'serve' in it, run a listen server and let others connect to us
  arguments: checkPageParam('serve') ? ['-d1', '-j28780'] : [],
  benchmark: checkPageParam('benchmark') ? { totalIters: 2000, iter: 0 } : null,
  failed: false,
  preRun: [],
  postRun: [],
  preloadPlugins: [],
  print: function(text) {
    console.log('[STDOUT] ' + text);
    //if (!Module.stdoutElement) Module.stdoutElement = document.getElementById('stdout');
    //Module.stdoutElement.value += text + '\n';
  },
  printErr: function(text) {
    console.log(text);
  },
  canvas: checkPageParam('headlessCanvas') ? headlessCanvas() : document.getElementById('canvas'),
  statusMessage: 'Starting...',
  progressElement: document.getElementById('progress'),
  setStatus: function(text) {
console.log(text);
    if (Module.setStatus.interval) clearInterval(Module.setStatus.interval);
    var statusElement = document.getElementById('status-text');
    if (Module.finishedDataFileDownloads >= 1 && Module.finishedDataFileDownloads < Module.expectedDataFileDownloads) {
      // If we are in the middle of multiple datafile downloads, do not show preloading progress - show only download progress
      var m2 = text.match(/([^ ]+) .*/);
      if (m2) {
        if (m2[1] == 'Preparing...') return;
      }
    }
    var m = text.match(/([^(]+)\((\d+(\.\d+)?)\/(\d+)\)/);
    if (m) {
      text = m[1];
      Module.progressElement.value = parseInt(m[2])*100;
      Module.progressElement.max = parseInt(m[4])*100;
      Module.progressElement.hidden = false;
    } else {
      Module.progressElement.value = null;
      Module.progressElement.max = null;
      Module.progressElement.hidden = true;
    }
    statusElement.innerHTML = text;
  },
  totalDependencies: 0,
  monitorRunDependencies: function(left) {
    this.totalDependencies = Math.max(this.totalDependencies, left);
    Module.setStatus(left ? 'Preparing... (' + (this.totalDependencies-left) + '/' + this.totalDependencies + ')' : 'All downloads complete.');
  },
  onFullScreen: function(isFullScreen) {
    Module.isFullScreen = isFullScreen;
    if (isFullScreen) {
      Module.resumeMainLoop();
      Module.setOpacity(1);
      Module.setStatus('');
      document.querySelector('.status .ingame').classList.add( 'hide' );
      Module.canvas.classList.remove( 'paused' );
      Module.canvas.classList.remove( 'hide' );
      //BananaBread.execute('musicvol $oldmusicvol'); // XXX TODO: need to restart the music by name here
    } else {
      Module.pauseMainLoop();
      Module.setOpacity(0.333);
      Module.setStatus('<b>paused (enter fullscreen to resume)</b>');
      Module.canvas.classList.add( 'paused' );
      document.querySelector('.status .ingame').classList.remove( 'hide' );
      Module.canvas.classList.add( 'hide' );
      //BananaBread.execute('oldmusicvol = $musicvol ; musicvol 0');
    }
  }
};

Module.fullscreenCallbacks = [];

Module.postLoadWorld = function() {
  document.title = 'BananaBread';

  if (Module.loadingMusic) {
    Module.loadingMusic.pause();
    Module.loadingMusic = null;
  }
  Module.tweakDetail();

  BananaBread.execute('sensitivity 10');
  BananaBread.execute('clearconsole');

  setTimeout(function() {
    BananaBread.execute('oldmusicvol = $musicvol ; musicvol 0');
  }, 1); // Do after startup finishes so music will be prepared up

  if (checkPageParam('windowed')) {
    Module.canvas.classList.remove('hide');
    Module.isFullScreen = 1;
    Module.requestFullScreen = function() {
      setTimeout(function() {
        Module.onFullScreen(1);
        Module.canvas.classList.remove('hide');
      }, 0);
    }
  }

  if (!Module.isFullScreen) {
    // Pause and fade out until the user presses fullscreen

    Module.pauseMainLoop();
    setTimeout(function() {
      document.querySelector('.status-content.loading').classList.add('hide');
      document.querySelector('.status-content.fullscreen-buttons').classList.remove('hide');
    }, 0);

    Module.resume = function() {
      Module.requestFullScreen();
      Module.setOpacity(1);
      Module.setStatus('');
      Module.resumeMainLoop();
   };

    Module.fullscreenLow = function() {
      document.querySelector('.status-content.fullscreen-buttons').classList.add('hide');
      Module.canvas.classList.remove('hide');
      Module.requestFullScreen(true);
      Module.setOpacity(1);
      Module.setStatus('');
      Module.resumeMainLoop();
      Module.fullscreenCallbacks.forEach(function(callback) { callback() });
    };

    Module.fullscreenHigh = function() {
      document.querySelector('.status-content.fullscreen-buttons').classList.add('hide');
      Module.canvas.classList.remove('hide');
      Module.requestFullScreen(true);
      Module.setOpacity(1);
      Module.setStatus('');
      BananaBread.execute('screenres ' + screen.width + ' ' + screen.height);
      Module.resumeMainLoop();
      Module.fullscreenCallbacks.forEach(function(callback) { callback() });
    };

    // All set!
    if (Module.readySound) {
      Module.readySound.play();
      Module.readySound = null;
    }
  }

  if (Module.benchmark) {
    Module.print('<< start game >>');
    Module.gameStartTime = Date.realNow();
    Module.gameTotalTime = 0;
    if (!window.headless) document.getElementById('main_text').classList.add('hide');
    Module.canvas.classList.remove('hide');
  }
};

Module.autoexec = function(){}; // called during autoexec on load, so useful to tweak settings that require gl restart
Module.tweakDetail = function(){}; // called from postLoadWorld, so useful to make changes after the map has been loaded

(function() {
  var fraction = 0.65;
  var desired = Math.min(fraction*screen.availWidth, fraction*screen.availHeight, 600);
  var w, h;
  if (screen.width >= screen.height) {
    h = desired;
    w = Math.floor(desired * screen.width / screen.height);
  } else {
    w = desired;
    h = Math.floor(desired * screen.height / screen.width);
  }
  Module.desiredWidth = w;
  Module.desiredHeight = h;
})();

// Load scripts

(function() {
  function loadChildScript(name, then) {
    var js = document.createElement('script');
    if (then) js.onload = then;
    js.src = name;
    document.body.appendChild(js);
  }

  var urlParts = pageParams.split(',');
  var setup = urlParts[0], preload = urlParts[1];

  var levelTitleContainer = document.querySelector('.level-title span');
  if (levelTitleContainer) {
    var levelTitle;
    switch(setup) {
      case 'low':    levelTitle = 'Arena';        break;
      case 'medium': levelTitle = 'Two Towers';   break;
      case 'high':   levelTitle = 'Lava Chamber'; break;
      case 'four':   levelTitle = 'Future';       break;
      case 'five':   levelTitle = 'Lava Rooms';   break;
      default: throw('unknown setup: ' + setup);
    };
    levelTitleContainer.innerHTML = levelTitle;
  }

  var previewContainer = document.querySelector('.preview-content.' + setup );
  if (previewContainer) previewContainer.classList.add('show');

  if(!Module.failed){
    loadChildScript('game/gl-matrix.js', function() {
      loadChildScript('game/setup_' + setup + '.js', function() {
        loadChildScript('game/preload_base.js', function() {
          loadChildScript('game/preload_character.js', function() {
            loadChildScript('game/preload_' + preload + '.js', function() {
              var scriptParts = ['bb'];
              if (checkPageParam('debug')) scriptParts.push('debug');
              loadChildScript(scriptParts.join('.') + '.js');
            });
          });
        });
      });
    });
  }
})();

(function(){
  var lowResButton = document.querySelector('.fullscreen-button.low-res');
  if (lowResButton) lowResButton.addEventListener('click', function(e){
    Module.fullscreenLow();
  }, false);
  var highResButton = document.querySelector('.fullscreen-button.high-res');
  if (highResButton) highResButton.addEventListener('click', function(e){
    Module.fullscreenHigh();
  }, false);
  var resumeButton = document.querySelector('.fullscreen-button.resume');
  if (resumeButton) resumeButton.addEventListener('click', function(e){
    Module.resume();
  }, false);
  var quitButton = document.querySelector('.fullscreen-button.quit');
  if (quitButton) quitButton.addEventListener('click', function(e){
    window.location = 'index.html';
  }, false);
})();
