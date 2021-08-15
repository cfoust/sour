sour
====

`sour` is a complete multiplayer [Sauerbraten](http://sauerbraten.org/) experience in the web delivered as a single Docker image.

## Overview

I have always loved playing Sauerbraten because of its simplicity: it's fast to download, easy to pick up, and keeps you in the action with instant respawns. Despite playing lots of games over the course of my life I haven't really found anything that scratches the same itch.

Some years ago I found [BananaBread](https://github.com/kripken/BananaBread), which was a basic tech demo that uses Emscripten to compile Sauerbraten for the web. The project was limited in scope and done at a time when bandwidth was a lot more precious. It also lacked multiplayer out of the box.

My goal was to ship an updated version of it in a single Docker image that I could deploy anywhere and play without forcing anyone to download the whole game. That's where `sour` comes in.

## Getting started

All you need is [Earthly](https://earthly.dev/) to build. Just run `earthly +docker` and it will make the `sour:latest` image.

To use, the `sour` image, expose ports `1234` and `28785` when you run a container like this:

```bash
docker run --rm -it -p 1234:1234 -p 28785:28785 sour:latest
```

It's worth nothing that you can change the first mapping (`1234`) to whatever you want (e.g `80`) but the second one has to be `28785`, since the frontend expects the proxy service to be at that port.

## Architecture

The current implementation involves three services, each of which corresponds to a directory in `services/`:
* `client/`: A fork of [BananaBread](https://github.com/kripken/BananaBread) with lots of modifications to add support for QServCollect's protocol and the WebSocket intermediary. There are also significant UI changes.
* `server/`: A fork of [QServCollect](https://github.com/deathstar/QServCollect), which is a dedicated Sauerbraten server.
* `proxy/`: A fork of [wsproxy](https://github.com/FWGS/wsproxy) which I changed to only allow proxying from TCP `28785` to UDP `28786`. This was the quickest way I found to get client/server communication working, though presumably you could just do this in a Python script.

## License

Each project that was forked into this repository has its own original license intact, though the glue code and subsequent modifications I have made are licensed according to the MIT license specified in `LICENSE`.
