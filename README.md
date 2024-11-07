<p align="center">
    <a href="https://sourga.me" target="_blank">
        <img src="gh-assets/header.png" alt="Sour Cover Image">
    </a>
</p>

<p align="center">
    <a href="https://discord.gg/WP3EbYym4M"><img src="https://img.shields.io/discord/1071091858576523274?color=5865F2&label=discord&style=flat-square" alt="Discord" /></a>
    <a href="https://github.com/cfoust/sour/releases"><img src="https://img.shields.io/github/downloads/cfoust/sour/latest/total?style=flat-square" alt="sour releases" /></a>
    <a href="https://github.com/cfoust/sour/blob/main/LICENSE"><img src="https://img.shields.io/github/license/cfoust/sour?color=48AC75&style=flat-square" alt="sour License Badge MIT" /></a>
</p>

## What is this?

Sour is a <a target="_blank" href="http://sauerbraten.org/">Cube 2: Sauerbraten</a> server that serves a fully-featured web-version of Sauerbraten (with support for mobile devices) in addition to accepting connections from the traditional, desktop version of the game. Sour is the easiest way to play Sauerbraten with your friends.

<a target="_blank" href="https://sourga.me/">Give it a try.</a>

## Installation

You can download an archive containing the Sour server and all necessary assets from [the releases page](https://github.com/cfoust/sour/releases). For now, only Linux and macOS are supported.

You can also install Sour via `brew`:

```bash
# Install the latest version:
brew install cfoust/taps/sour

# Or a specific one:
brew install cfoust/taps/sour@0.2.2
```

In addition to all of the base game assets, these archives only contain three maps: `complex`, `dust2`, and `turbine`.

## Running Sour

To run Sour, extract a release archive anywhere you wish, navigate to that directory, and run `./sour`. If you installed Sour with `brew`, just run `sour` in any terminal session.

Running `sour` will start a Sour server accessible to web clients at `http://0.0.0.0:1337` and to desktop clients on port 28785. In other words, you should be able to connect to the Sour server in the Sauerbraten desktop client by running `/connect localhost`.

By serving on `0.0.0.0` by default, the Sour server will be available to other devices on the local network at IP of the device running the Sour server.

## Configuration

Sour is highly configurable. When run without arguments, `sour` defaults to running `sour serve` with the [default Sour configuration](https://github.com/cfoust/sour/blob/main/pkg/config/default.yaml). You change Sour's configuration by providing the path to a configuration file to `sour serve`:

```bash
sour serve config.yaml
```

Sour can be configured using `.yaml` or `.json` files; the structure is the same in both cases.

To print the default configuration to standard output, run `sour config`:

```bash
sour config > config.yaml
```

Sour also supports merging configurations together.

```bash
sour serve config_a.yaml some_path/config_b.json config_c.yaml
```

These configurations are merged from left to right using [CUE](https://cuelang.org/docs/). In other words, configurations are evaluated in order from left to right. CUE merges JSON data by overwriting values (if they're scalar, such as strings, booleans, and numbers) or combining values (if they're arrays). In effect, this means that configurations can specify values for only a subset of properties without problems.

## Goals

- **Modernize Sauerbraten.** The gaming landscape has changed. Provide a modern multiplayer experience with matchmaking, private games, rankings, and seamless collaboration on maps. Make as much of this functionality available to the unmodified desktop game as possible.
- **Preserve the experience of playing the original game.** While it is possible that Sour may someday support arbitrary game modes, assets, clients, and server code, the vanilla game experience should still be available.
- **Be the best example of a cross-platform, open-source FPS.** Deployment of Sour on your own infrastructure with whatever configuration you like should be easy. Every aspect of Sour should be configurable.

## Architecture

Here is a high level description of the repository's contents:

- `pkg` and `cmd`: All Go code used in Sour and its services.
  - `cmd/sourdump`: A Go program that calculates the minimum list of files necessary for the game to load a given map.
  - `cmd/sour`: The Sour game server, which provides a number of services to web clients:
    - Gives clients both on the web and desktop client access to game servers managed by Sour.
- `game`: All of the Cube 2 code and Emscripten compilation scripts. Originally this was a fork of [BananaBread](https://github.com/kripken/BananaBread), kripken's original attempt at compiling Sauerbraten for the web. Since then I have upgraded the game to the newest mainline version several times and moved to WebGL2.
- `client`: A React web application that uses the compiled Sauerbraten game found in `game`, pulls assets, and proxies all server communication over a WebSocket.
- `assets`: Scripts for building web-compatible game assets. This is an extremely complicated topic and easily the most difficult aspect of shipping Sauerbraten to the web. Check out this [section's README](services/assets) for more information.

## Contributing

Join us on [Discord](https://discord.gg/WP3EbYym4M) to chat with us and see how you can help out! Check out the [issues tab](https://github.com/cfoust/sour/issues) to get an idea of what needs doing.

## Inspiration

Some years ago I came across [BananaBread](https://github.com/kripken/BananaBread), which was a basic tech demo that used [Emscripten](https://emscripten.org/) to compile Sauerbraten for the web. The project was limited in scope and done at a time when bandwidth was a lot more precious.

## License

Each project that was forked into this repository has its own original license intact, though the glue code and subsequent modifications I have made are licensed according to the MIT license specified in `LICENSE`.
