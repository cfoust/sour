# sour
<p align="center">
  <img src="gh-assets/header.gif">
</p>

[![License:
MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

`sour` is a complete [Cube 2: Sauerbraten](http://sauerbraten.org/) experience in the web delivered as a single Docker image

## Warning: I am still fixing issues after rebasing onto Sauerbraten 2020. Multiplayer is not working. Use the r4349 tag if you want a working copy.

## Overview

I have always loved playing Sauerbraten because of its simplicity: it's fast to download, easy to pick up, and keeps you in the action with instant respawns. Despite playing lots of games over the course of my life I haven't really found anything that scratches the same itch.

Some years ago I found [BananaBread](https://github.com/kripken/BananaBread), which was a basic tech demo that uses Emscripten to compile Sauerbraten for the web. The project was limited in scope and done at a time when bandwidth was a lot more precious. It also lacked multiplayer out of the box.

My goal was to ship an updated version of it in a single Docker image that I could deploy anywhere and play without forcing anyone to download the whole game. That's where `sour` comes in.

## Getting started

All you need is Docker and [Earthly](https://earthly.dev/) to build. Just run `earthly +image` and it will make the `sour:latest` image.

To use the `sour` image, expose ports `1234` and `28785` when you run a container like this:

```bash
docker run --rm -it -p 1234:1234 -p 28785:28785 sour:latest
```

It's worth nothing that you can change the first mapping (`1234:1234`) to whatever you want (e.g `80:1234`) but the second one has to be `28785`, since the frontend expects the proxy service to be at that port.

## Architecture

Here is a high level description of the repository's directory structure:
* `services/game/cube2`: A fork of [BananaBread](https://github.com/kripken/BananaBread), which was kripken's initial attempt at getting a version of Sauerbraten running using Emscripten. He forked Sauerbraten at the mainline [r4059](https://sourceforge.net/p/sauerbraten/code/4059), I upgraded to [r4349](https://sourceforge.net/p/sauerbraten/code/4349), then finally upgraded to the latest mainline at the time [r6519](https://sourceforge.net/p/sauerbraten/code/6519). Contains a handful of modifications and restrictions to make sure it can run well in the web.
* `services/game/assets`: A checkout of Sauerbraten's `packages/` directory, which contains all of the game's default assets. Also includes Sour's asset bundling mechanism to generate prepackaged Emscripten file bundles for each game map.
* `services/server/`: A fork of [QServCollect](https://github.com/deathstar/QServCollect), which is a dedicated Sauerbraten server.
* `services/proxy/`: A fork of [wsproxy](https://github.com/FWGS/wsproxy) which I changed to only allow proxying from TCP `28785` to UDP `28786`. This was the quickest way I found to get client/server communication working, though presumably you could just do this in a Python script.
* `services/client/`: A React web application that glues together the compiled Sauerbraten code and our asset fetching mechanism.

The resulting Docker image will run services on these ports when started:
* `tcp:1234`: An nginx server that serves up the `client`, compiled Sauerbraten binaries, and assets.
* `tcp:28785`: The hacky `wsproxy` version we use that proxies WS connections to the container's UDP port `28786`.
* `udp:28786`: A real QServCollect server. Typically you don't expose this, but if you want to do crossplay (connect with real Sauerbraten) you absolutely can.

## Contributing

All contributions are welcome. Developing Sour is made a bit easier with Earthly but there are still some caveats.

To hack on Sour:
1. Run `earthly +game` (compiles the game), `earthly +assets` (builds its assets for the web,) and `earthly +image-slim` (builds the Sour image without assets.)
2. If you want to work on multiplayer, run this Docker container: `docker run --rm -it -p 28785:28785 sour:slim`. It makes the QServCollect server available (over a WebSocket) on `28785`.
3. Start the web client.
    1. If you have [warm](https://github.com/cfoust/warm/blob/master/warm), run `warm shell`, `cd services/client` and run `yarn serve`.
    2. If you do not have `warm`, run `docker run --rm -it --volume=$(pwd):$(pwd) -w $(pwd) --init --network=host node:14.17.5 /bin/bash`, `cd services/client` and run `yarn serve`.
4. Navigate to `http://localhost:1234`. To recompile the game, just do `earthly +game` and refresh the page. The same applies to `earthly +assets`. If you change the client code, also just refresh.

Check out the roadmap below to see what you might be able to help with.

* [X] Restore the rendering pipeline to working condition
* [X] Fix up the main menu
  * [X] Make sure textures used in the main menu are not downsized
  * [X] Remove options not pertinent to the web version
  * [X] Allow the user to connect to the Sour server
* [X] Fix full screen UX
* [ ] Fix cursor and crosshair transparency
* [ ] Add pretty loading screen before canvas is ready
* [ ] Improve map transitions and use real `renderprogress` for asset downloads
* [ ] Fix the transparency in the cursor texture
* [ ] Fix model textures. For some reason none of them are working.
* [ ] Issue with name being malformed (bad welcome packet?)
* [ ] Make it so players cannot change the map to one that does not exist
  * [ ] Come up with a better cache invalidation strategy (likely commit hash?)

**After this, we're back to working condition**

---

* [ ] Allow for players to create custom matches
* [ ] Allow Sour to read from the real master server and connect to real Sauerbraten servers
* [ ] Support saving and loading `.ogz` maps from the user's device
* [ ] Save demos for any game played to IndexedDB and allow for download
* [ ] Demo player with seek/play/pause
  * [ ] Stretch goal: generate gifs in the browser

## License

Each project that was forked into this repository has its own original license intact, though the glue code and subsequent modifications I have made are licensed according to the MIT license specified in `LICENSE`.
