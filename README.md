# sour
<p align="center">
  <img src="gh-assets/header.png">
</p>

[![License:
MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

[![Open in Gitpod](https://gitpod.io/button/open-in-gitpod.svg)](https://gitpod.io/#https://github.com/cfoust/sour)

`sour` is a complete [Cube 2: Sauerbraten](http://sauerbraten.org/) experience in the web delivered as a single Docker image. [Give it a try.](https://sourga.me/)

## Goals

* **Modernize Sauerbraten.** The gaming landscape has changed. Provide a modern multiplayer experience with matchmaking, private games, rankings, and seamless collaboration on maps. Make as much of this functionality available to the unmodified desktop game as possible.
* **Preserve the experience of playing the original game.** While it is possible that Sour may someday support arbitrary game modes, assets, clients, and server code, the vanilla game experience should still be available.
* **Be the best example of a cross-platform, open-source FPS.** Deployment of Sour on your own infrastructure with whatever configuration you like should be easy. Every aspect of Sour should be configurable.

## Running

```
docker run --rm -it -p 1234:1234 -p 28785:28785/udp ghcr.io/cfoust/sour
```

You can then access Sour at `http://localhost:1234/` or by connecting in [the desktop client](http://sauerbraten.org/) with `/connect localhost`.

## Deploying

If you wish to deploy Sour more seriously, I provide an example configuration for [docker-compose](https://docs.docker.com/compose/) [here](https://github.com/cfoust/sour/blob/main/examples/docker-compose.yml) using [letsencrypt-nginx-sidecar](https://github.com/jwulf/letsencrypt-nginx-sidecar).

## Architecture

Here is a high level description of the repository's contents:
* `services/game`: All of the Cube 2 code and Emscripten compilation scripts. Originally this was a fork of [BananaBread](https://github.com/kripken/BananaBread), kripken's original attempt at compiling Sauerbraten for the web. Since then I have upgraded the game to the newest mainline version several times and moved to WebGL2.
* `services/go/`: All Go code used in Sour and its services.
  * A Go program that calculates the minimum list of files necessary for the game to load a given map.
  * The Sour game server, which provides a number of services to web clients:
      * Gives clients both on the web and desktop client access to game servers managed by Sour.
      * Periodically fetches Sauerbraten server information from the master server, pings all of the available servers, and broadcasts the results to web clients. This is so we can fill in the server browser.
* `services/assets`: Scripts for building web-compatible game assets. This is an extremely complicated topic and easily the most difficult aspect of shipping Sauerbraten to the web. Check out this [section's README](services/assets) for more information.
* `services/ingress/`: `nginx` configurations for development, production, and Gitpod.
* `services/proxy/`: A fork of [wsproxy](https://github.com/FWGS/wsproxy). This allows web clients to connect to _all_ of the existing Sauerbraten servers and crossplay with desktop clients.
* `services/client/`: A React web application that controls Sauerbraten, pulls assets, and proxies all game communication over WebSockets.

## Contributing

The easiest way to hack on Sour is in Gitpod using the button below. Gitpod is a web-based VSCode environment that runs everything necessary for development in a cloud-based container, meaning that everything is set up and working for you right away. You do not even have to use VSCode; Gitpod supports [custom dotfiles](https://www.gitpod.io/docs/config-dotfiles) which allows me to use my full [vim-based setup](https://github.com/cfoust/cawnfig/tree/master/configs/vim) from a browser tab.

[![Open in Gitpod](https://gitpod.io/button/open-in-gitpod.svg)](https://gitpod.io/#https://github.com/cfoust/sour)

If you want to run things locally all you need is Docker, [Earthly](https://earthly.dev/), and `docker-compose`. After that, just run `./serve`. Then navigate to `http://localhost:1234`. All of the game's services will recompile and restart when you make changes.

Check out the roadmap below to see what you might be able to help with.

### General
* [ ] Better documentation
  * [ ] Configuration
* [ ] Fix Sour in browsers other than Chrome
  * [ ] (Mobile) Safari
  * [ ] Firefox
* [ ] Allow for providing the desired maps in an image as a build argument
* [ ] Terraform recipes for deployment
### Bugs
* [ ] Preserve the reliable flag on ENet packets coming from the server
### Assets
* [ ] Simplify index file format
### Sourverse
* [ ] Server-side map editing state
* [ ] Rooms vs maps
  * [ ] Map address: map/12312321
  * [ ] Room address: room/abcdef0123
  * [ ] Room and map namespace is shared?
* [ ] Loading maps from the backend
* [ ] Allow teleports to rooms, maps, and locations within them
* [ ] Assign each user a room + map on join
* [ ] Hub world with portals to rooms
* [ ] Consent to automatic map sending on desktop
* [ ] Map browser
* [ ] Map usage statistics, favorite maps, etc
### Gameplay
* [ ] Reset master mode when you swap servers
* [ ] Fix nasty asset base asset sizes for web clients
* [ ] LRU cache for cluster map fetching
* [ ] Leaderboard for ELO rankings
* [ ] Separate the rendering and input loops on web (or at least don't vsync game inputs)
* [ ] Simple, beautiful main menu
  * [ ] Add `<noscript>` with a plea to enable JavaScript
  * [ ] Ensure page load time isn't horrible
* [ ] Save demos for every duel
* [ ] Allow for backgrounding the tab by responding to pings
* [ ] Demo player with seek/play/pause
  * [ ] Automatically save demos for every Sour game
  * [ ] Stretch: generate gifs in the browser
* [ ] Save demos for any game played to IndexedDB and allow for download
* [ ] Explore running Sour in a Web Worker rather than the rendering thread
* [ ] Use password field for queueing or room joining eg /connect sourga.me 28785 ffa
* [ ] Stretch: spectate any running Sour match
### Map editing
* [ ] Repair or port shaders that were disabled in the game upgrade
  * [ ] Fix the outline shader (pressing 7 in edit mode)
  * [ ] Water reflection and refraction (this is really hard)
* [ ] Support saving and loading `.ogz` maps from the user's device

## Inspiration

Some years ago I came across [BananaBread](https://github.com/kripken/BananaBread), which was a basic tech demo that used [Emscripten](https://emscripten.org/) to compile Sauerbraten for the web. The project was limited in scope and done at a time when bandwidth was a lot more precious.

## License

Each project that was forked into this repository has its own original license intact, though the glue code and subsequent modifications I have made are licensed according to the MIT license specified in `LICENSE`.
