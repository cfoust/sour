
<p align="center">
  <h1>🍋Sour</h1>
</p>

<p align="center">
    <a href="https://sourga.me" target="_blank">
        <img src="gh-assets/header.png" alt="Sour Cover Image">
    </a>
</p>


<p align="center">
    <!-- Gitpod -->
    <a target="_blank" href="https://gitpod.io/#https://github.com/cfoust/sour">
        <img src="https://img.shields.io/badge/gitpod-devenv-orange" alt="Sour Gitpod Development Environment" />
    </a>
    <!-- LICENSE -->
    <a target="_blank" href="https://github.com/cfoust/sour/blob/main/LICENSE">
        <img src="https://img.shields.io/github/license/cfoust/sour" alt="Sour License Badge MIT" />
    </a>
    <!-- Discord -->
    <a target="_blank" href="https://discord.gg/WP3EbYym4M">
        <img src="https://img.shields.io/discord/1071091858576523274?label=discord&logo=discord&style=social" />
    </a>
    <!-- Github Action Build-->
    <a target="_blank" href="https://github.com/cfoust/sour/actions/workflows/ci.yml">
        <img src="https://github.com/cfoust/sour/actions/workflows/ci.yml/badge.svg" />
    </a>
</p>

<p align="center">
    Sour is a multiplatform modernization of <a target="_blank" href="http://sauerbraten.org/">Cube 2: Sauerbraten</a> delivered as a single Docker image. <a target="_blank" href="https://sourga.me/">Give it a try.</a>
</p>

## Features
* **A complete web version of Sauerbraten**
    * All original game assets and features
    * Support for mobile devices
    * Connect to all existing Sauerbraten community servers and crossplay with desktop players
    * Arbitrary game mods
    * Sophisticated load-on-demand system for assets that allows for loading arbitrary content that has been packaged for the game
    * Links to servers (`/server/[ip]/[port]`) and maps (`/map/complex`)
* **An advanced Sauerbraten game server**
    * Supports connections both from the web and from desktop Sauerbraten clients
    * Server multiplexing: you can run arbitrary game servers with their own maps, modes, and configurations and allow users to move between them without disconnecting from the server
    * Players can create private game servers and invite other players on-demand
    * Discord authentication both for web and desktop clients
    * 1v1 matchmaking and persistent ELO scores for users
    * Persistent map editing: edits users make to maps are stored on the server side and visible to other clients who join (no more `/getmap` and `/sendmap`)
    * User-owned spaces that they can edit (player housing, basically)
    * User sessions can be saved as demos for debugging
    * Automatically sends game maps to desktop clients that don't have them
    * Mechanism for running client-side CubeScript on desktop clients
* **Utilities and libraries for working with Sauerbraten**
    * `sourdump`, a tool for calculating all of the files a Sauerbraten map, model, or `.cfg` file uses so that you can only send the minimum set of assets a client needs
    * Go library for opening, manipulating, and saving Sauerbraten game maps
    * Go library providing interoperation between Go and CubeScript

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

Check out the [issues tab](https://github.com/cfoust/sour/issues) to see what you might be able to help with.

## Inspiration

Some years ago I came across [BananaBread](https://github.com/kripken/BananaBread), which was a basic tech demo that used [Emscripten](https://emscripten.org/) to compile Sauerbraten for the web. The project was limited in scope and done at a time when bandwidth was a lot more precious.

## License

Each project that was forked into this repository has its own original license intact, though the glue code and subsequent modifications I have made are licensed according to the MIT license specified in `LICENSE`.
