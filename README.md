# sour
<p align="center">
  <img src="gh-assets/header.gif">
</p>

[![License:
MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

`sour` is a complete [Cube 2: Sauerbraten](http://sauerbraten.org/) experience in the web delivered as a single Docker image. [Give it a try.](https://sourga.me/)

## Introduction

Sauerbraten has a special place in my heart: it's fast to download, easy to pick up, and keeps you in the action with instant respawns. Despite playing lots of games over the course of my life I haven't really found anything that scratches the same itch.

Some years ago I found [BananaBread](https://github.com/kripken/BananaBread), which was a basic tech demo that uses [Emscripten](https://emscripten.org/) to compile Sauerbraten for the web. The project was limited in scope and done at a time when bandwidth was a lot more precious. It also lacked multiplayer out of the box.

My goal was to ship an updated version of it in a single Docker image that I could deploy anywhere and play without forcing anyone to download the whole game. That's where `sour` comes in.

## Project goals

The Sauerbraten community is small and it will probably always remain that way. There are a few main goals for this project:
* Make it easier to play Sauerbraten. Web technologies and bandwidth have gotten to the point where it is practical and desirable to play Sauerbraten in the browser without forcing players to download a desktop client.
* Mimic the experience of playing the original game as closely as possible. While it is possible that Sour may someday support arbitrary game modes, assets, clients, and server code, the vanilla game experience should still be available.
* Deployment of Sour on your own infrastructure with whatever configuration you like should be easy. Every aspect of Sour should be configurable.

## Running

```
docker run --rm -it -p 1234:1234 -p 28785:28785 ghcr.io/cfoust/sour
```

You can then access Sour at `http://localhost:1234/`.

**Note:** The public Docker image only ships with the `complex` and `xenon` maps for now. While Sour supports _all_ of Sauerbraten's maps, images that include all of them are very big. Your mileage may vary.

The Sour container runs services on these ports when started:
* `tcp:1234`: An nginx server that serves up the game client, compiled Sauerbraten binaries, and assets.
* `tcp:28785`: This is the port the client connects to by default to reach the game server. This proxies WS connections to the container's UDP port `28786`.
* `udp:28786`: A real [QServCollect](https://github.com/deathstar/QServCollect) server. Typically you don't expose this, but if you want to do crossplay (connect with real Sauerbraten) you absolutely can.

Should you wish to change where the WebSocket service is hosted **you must also indicate that to the static site.** You can do this by providing an environment variable:

```
docker run --rm -it -p 1234:1234 -p 28785:28785 -e GAME_SERVER=wss://server.sourga.me ghcr.io/cfoust/sour
```

## Deploying

If you wish to deploy Sour more seriously, I provide an example configuration for [docker-compose](https://docs.docker.com/compose/) [here](https://github.com/cfoust/sour/blob/main/examples/docker-compose.yml) using [letsencrypt-nginx-sidecar](https://github.com/jwulf/letsencrypt-nginx-sidecar).

## Building

All you need is Docker and [Earthly](https://earthly.dev/) to build. Just run `earthly +image` and it will make the `sour:latest` image.

## Architecture

Here is a high level description of the repository's directory structure:
* `services/game/cube2`: A fork of [BananaBread](https://github.com/kripken/BananaBread), which was kripken's initial attempt at getting a version of Sauerbraten running using Emscripten. He forked Sauerbraten at the mainline [r4059](https://sourceforge.net/p/sauerbraten/code/4059), I upgraded to [r4349](https://sourceforge.net/p/sauerbraten/code/4349), then finally upgraded to the latest mainline at the time [r6519](https://sourceforge.net/p/sauerbraten/code/6519). My fork contains a handful of modifications and restrictions to make sure it can run well in the web.
* `services/game/assets`: A checkout of Sauerbraten's `packages/` directory, which contains all of the game's default assets. This directory also includes Sour's asset bundling mechanism to generate prepackaged Emscripten file bundles for each game map.
* `services/server/`: A fork of [QServCollect](https://github.com/deathstar/QServCollect), which is a dedicated Sauerbraten server.
* `services/proxy/`: A fork of [wsproxy](https://github.com/FWGS/wsproxy) which I changed to only allow proxying from TCP `28785` to UDP `28786`. This was the quickest way I found to get client/server communication working, though presumably you could just do this in a Python script.
* `services/client/`: A React web application that glues together the compiled Sauerbraten code and our asset fetching mechanism.

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

* [ ] Better development experience with simple docker-compose setup
* [ ] Better documentation on services, how to build assets, et cetera
* [ ] Allow for providing the desired maps in an image as a build argument
* [ ] Support all player models (right now it's just snout)
* [ ] Add CTF assets to the base game
* [ ] Explore running Sour in a Web Worker rather than the rendering thread
* [ ] Investigate differences in font colors between the real Sauer and Sour
* [ ] Make sure `getmap` and `sendmap` don't break the game server
* [ ] Allow for players to create custom matches
* [ ] Allow Sour to read from the real master server and connect to real Sauerbraten servers
* [ ] Support saving and loading `.ogz` maps from the user's device
* [ ] Upgrade the Emscripten version
* [ ] Save demos for any game played to IndexedDB and allow for download
* [ ] Demo player with seek/play/pause
  * [ ] Stretch goal: generate gifs in the browser

## License

Each project that was forked into this repository has its own original license intact, though the glue code and subsequent modifications I have made are licensed according to the MIT license specified in `LICENSE`.
