# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What is Sour

Sour is a Cube 2: Sauerbraten game server that serves a fully-featured web version of Sauerbraten (via Emscripten) while also accepting connections from the traditional desktop client. The Go server acts as the hub connecting both client types to the same game servers.

## Commands

### Go Server

```bash
# Build the server
go build ./cmd/sour

# Run with default config (web on :1337, desktop on :28785)
./sour

# Run with custom config
./sour serve config.yaml

# Print default configuration
./sour config

# Run all Go tests
go test ./...

# Run a single package's tests
go test ./pkg/gameserver/game/...
```

### Web Client

```bash
cd client

# Install dependencies
yarn install

# Development server (watches for changes)
yarn serve

# Production build (outputs to ./dist)
yarn build
```

### Asset Generation

```bash
cd assets

# Install Python deps (run once)
./setup

# Generate bundles for base game maps
python3 base.py

# Generate bundles for specific maps only
python3 base.py dust2 complex turbine
```

## Architecture

The system has four major components:

### 1. Go Server (`cmd/sour`, `pkg/`)

The server is built around a `Cluster` (`pkg/server/service/module.go`) which orchestrates:

- **Ingress** (`pkg/server/ingress`): Two ingress types funnel connections into a shared channel:
  - `WSIngress`: WebSocket connections from browser clients (`/ws/`)
  - `ENetIngress`: UDP connections from desktop Sauerbraten clients
- **ServerManager** (`pkg/server/servers`): Spawns and manages `GameServer` instances (goroutines wrapping the `gameserver` package).
- **UserOrchestrator**: Manages connected users and routes them to servers.
- **Matchmaker**: Pairs players for duels based on ELO.
- **SpaceManager** (`pkg/server/verse`): Manages persistent named "spaces" (lobbies) that players can join by name.

The HTTP mux serves:
- `/` — embedded static web client
- `/ws/` — WebSocket ingress
- `/api/` — REST API
- `/assets/<n>/` — locally-served asset files

### 2. Game Server (`pkg/gameserver`)

Pure Go implementation of Sauerbraten game logic: game modes (FFA, CTF, insta, coop), player state, scoring, timers, and packet handling. This package has no C dependencies.

### 3. Web Client (`client/src`)

React + TypeScript app built with Parcel. Key design decisions:
- All game network traffic is proxied over WebSocket (the Emscripten-compiled game binary communicates with `unsafe-startup.js` which pipes to/from the WS connection).
- Asset loading happens in a **Web Worker** (`src/assets/worker.ts`) because IndexedDB access is too slow for the rendering thread. Assets are cached to IndexedDB by content hash.
- The client config (asset sources, server addresses) is JSON-encoded by the Go server and injected into the static HTML at startup.

### 4. Assets (`assets/`)

Python scripts that build `.sour` bundles — minimal sets of game files needed to load each map. The pipeline:
1. `sourdump` (`cmd/sourdump`) parses `.ogz` map files and their `.cfg` scripts to enumerate every referenced file.
2. `package.py` calls Emscripten's `file_packager.py` to produce bundles and compresses large images with ImageMagick.
3. Each bundle is named by its content hash; an `.index.json` file lists all available maps/mods.

Asset sources at runtime are specified via the `Assets` field in server config (supports local filesystem paths prefixed `fs:` and HTTP URLs).

## Configuration

Configuration uses YAML or JSON with [CUE](https://cuelang.org/) for merging. The schema lives in `pkg/config/schema.cue`. Multiple config files are merged left-to-right:

```bash
./sour serve base.yaml overrides.yaml
```

Key config sections: `server.presets` (game server templates), `server.ingress` (web/desktop ports), `server.spaces` (persistent lobbies), `server.matchmaking`.

## Key Packages

| Package | Purpose |
|---|---|
| `pkg/game/protocol` | Sauerbraten wire protocol encoding/decoding |
| `pkg/game/commands` | In-game command parsing framework |
| `pkg/game/io` | Low-level packet I/O |
| `pkg/maps` | OGZ map format parser |
| `pkg/cs` | CubeScript bindings via SWIG (C++) |
| `pkg/enet` | ENet UDP networking bindings |
| `pkg/assets` | Asset fetching, caching, and index management |
| `pkg/config` | Configuration loading and CUE-based merging |
| `pkg/chanlock` | Channel-based mutex utilities |
