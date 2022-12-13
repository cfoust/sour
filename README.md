# sour
<p align="center">
  <img src="gh-assets/header.png">
</p>

[![License:
MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

[![Open in Gitpod](https://gitpod.io/button/open-in-gitpod.svg)](https://gitpod.io/#https://github.com/cfoust/sour)


`sour` is a complete [Cube 2: Sauerbraten](http://sauerbraten.org/) experience in the web delivered as a single Docker image. [Give it a try.](https://sourga.me/)

## Introduction

Some years ago I came across [BananaBread](https://github.com/kripken/BananaBread), which was a basic tech demo that used [Emscripten](https://emscripten.org/) to compile Sauerbraten for the web. The project was limited in scope and done at a time when bandwidth was a lot more precious. It also lacked multiplayer out of the box.

My goal was to ship an updated version of it in a single Docker image that I could deploy anywhere and play without forcing anyone to download the whole game. That's where `sour` comes in.

## Project goals

* Make it easier to play Sauerbraten. Web technologies and bandwidth have gotten to the point where it is practical and desirable to play Sauerbraten in the browser.
* Mimic the experience of playing the original game as closely as possible. While it is possible that Sour may someday support arbitrary game modes, assets, clients, and server code, the vanilla game experience should still be available.
* Deployment of Sour on your own infrastructure with whatever configuration you like should be easy. Every aspect of Sour should be configurable.

## Running

```
docker run --rm -it -p 1234:1234 ghcr.io/cfoust/sour
```

You can then access Sour at `http://localhost:1234/`.

**Note:** The public Docker image only ships with the `turbine` and `dust2` maps for now. While Sour supports _all_ of Sauerbraten's maps, images that include all of them are very big. Your mileage may vary.

## Deploying

If you wish to deploy Sour more seriously, I provide an example configuration for [docker-compose](https://docs.docker.com/compose/) [here](https://github.com/cfoust/sour/blob/main/examples/docker-compose.yml) using [letsencrypt-nginx-sidecar](https://github.com/jwulf/letsencrypt-nginx-sidecar).

## Architecture

Here is a high level description of the repository's directory structure:
* `services/game`: All of the Cube 2 code and Emscripten compilation scripts. Originally this was a fork of [BananaBread](https://github.com/kripken/BananaBread), kripken's original attempt at compiling Sauerbraten for the web. Since then I have upgraded the game to the newest mainline version several times and moved to WebGL2.
* `services/assets`: Scripts for building web-compatible game assets. This is an extremely complicated topic and easily the most difficult aspect of shipping Sauerbraten to the web. Check out this [section's README](services/assets) for more information.
* `services/go/`: All Go code used in Sour and its services.
  * A Go program that calculates the minimum list of files necessary for the game to load a given map.
  * The Sour game server, which provides a number of services to web clients:
      * Gives clients access to game servers managed by Sour, including ones only accessible to the web and otherwise.
      * Periodically fetches Sauerbraten server information from the master server, pings all of the available servers, and broadcasts the results to web clients. This is so we can fill in the server browser.
* `services/ingress/`: `nginx` configurations for development, production, and Gitpod.
* `services/server/`: A fork of [QServCollect](https://github.com/deathstar/QServCollect), which is a dedicated Sauerbraten server. I added support for accepting connections on a Unix domain socket.
* `services/proxy/`: A fork of [wsproxy](https://github.com/FWGS/wsproxy). This allows web clients to connect to _all_ of the existing Sauerbraten servers and crossplay with desktop clients.
* `services/client/`: A React web application that glues together the compiled Sauerbraten code and our asset fetching mechanism.

## Contributing

The easiest way to hack on Sour is in Gitpod using the button below. Gitpod is a web-based VSCode environment that runs everything necessary for development in a cloud-based container, meaning that everything is set up and working for you right away. You do not even have to use VSCode; Gitpod supports [custom dotfiles](https://www.gitpod.io/docs/config-dotfiles) which allows me to use my full [vim-based setup](https://github.com/cfoust/cawnfig/tree/master/configs/vim) from a browser tab.

[![Open in Gitpod](https://gitpod.io/button/open-in-gitpod.svg)](https://gitpod.io/#https://github.com/cfoust/sour)

If you want to run things locally (some people are old fashioned that way) all you need is Docker, [Earthly](https://earthly.dev/), and `docker-compose`. After that, just run `./serve`. Then navigate to `http://localhost:1234`. All of the game's services will recompile and restart when you make changes.

Check out the roadmap below to see what you might be able to help with.

### General
* [ ] Better documentation
  * [ ] Create a README for each directory in `services/`
* [ ] Ensure Sour works in Firefox
* [ ] Allow for providing the desired maps in an image as a build argument
* [ ] Full Terraform support for deploying Sour
* [ ] Allow parts of Sour to be configured with a high-level configuration file
### Gameplay
* [ ] CTF still doesn't work and flags don't show up
* [ ] Modern multiplayer
  * [X] Use socket comms to connect to Sour-internal servers
  * [X] Arbitrary creation and hot swapping of users between servers
  * [ ] Allow users to create private servers
    * [ ] Update the URL to the current server on join eg `sourga.me/server/127.0.0.1:28785`
  * [ ] Creating servers from presets
  * [ ] Matchmaking (1v1)
  * [ ] Stretch: ELO
  * [ ] Stretch: allow desktop clients full access to Sour clusters
* [ ] Allow for backgrounding the tab by responding to pings
* [ ] Simple, beautiful main menu
  * [ ] Clean loading page for slow connections
  * [ ] Add `<noscript>` with a plea to enable JavaScript
  * [ ] Ensure page load time isn't horrible
* [ ] Demo player with seek/play/pause
  * [ ] Stretch: generate gifs in the browser
* [ ] Save demos for any game played to IndexedDB and allow for download
* [ ] Explore running Sour in a Web Worker rather than the rendering thread
* [ ] Support all player models (right now it's just snout)
### Map editing
* [ ] Repair or port shaders that were disabled in the game upgrade
  * [ ] Fix the outline shader (pressing 7 in edit mode)
  * [ ] Water reflection and refraction (this is really hard)
* [ ] Support saving and loading `.ogz` maps from the user's device

## License

Each project that was forked into this repository has its own original license intact, though the glue code and subsequent modifications I have made are licensed according to the MIT license specified in `LICENSE`.
