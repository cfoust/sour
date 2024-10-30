# Waiter

A game server for [Cube 2: Sauerbraten](http://sauerbraten.org/) forked from [sauerbraten/waiter](https://github.com/sauerbraten/waiter).

## Features

What works:

- ffa, insta, insta team, effic, effic team, tactics, tactics team
- ctf, insta ctf, effic ctf
- chat, team chat
- changing weapon, shooting, killing, suiciding, spawning
- global auth (`/auth` and `/authkick`)
- local auth (`/sauth`, `/dauth`, `/sauthkick`, `/dauthkick`, auth-on-connect)
- sharing master
- setting mastermode
- forcing gamemode and/or map
- pausing & resuming (with countdown)
- locking teams (`keepteams` server command)
- queueing maps (`queuemap` server command)
- changing your name
- extinfo (server mod ID: -9)

Server commands:

These can be used either as `#cmd bla foo` or `/servcmd cmd bla foo`:

- `keepteams 0|1` (a.k.a. `persist`): set to 1 to disable randomizing teams on map load
- `queuemap [map...]`: check the map queue or enqueue one or more maps
- `competitive 0|1`: in competitive mode, the server waits for all players to load the map before starting the game, and automatically pauses the game when a player leaves or goes to spectating mode

Pretty much everything else is not yet implemented:

- any modes requiring bases (capture) or tokens (collect)
- demo recording
- `/checkmaps` (will compare against server-side hash, not majority)
- overtime (& maybe golden goal)

Some things are specifically not planned and will likely never be implemented:

- bots
- map voting
- coop edit mode (including `/sendmap` and `/getmap`)
- claiming privileges using `/setmaster 1` (relinquishing them with `/setmaster 0` and sharing master using `/setmaster 1 <cn>` already works)

## Building

Make sure you have Go installed as well as the ENet development headers (on Fedora: `sudo dnf install enet-devel`, on macOS: `brew install enet`). Clone the repository, `cd waiter`, then `make all`.

You can then start the server with `./waiter`. The server requires `config.json`, `bans.json` and `users.json` to be placed in the working directory.

## To Do

- capture and regen capture (capture base events)
- intermission stats (depending on mode)
- #stats command
- store frags, deaths, etc. in case a player re-connects

## Project Structure

All functionality is organized into packages. [`/cmd/waiter/`](/cmd/waiter/) contains the actual command to start a server, i.e. configuration file parsing, initialization of all components, and preliminary handling of incoming packets. Detailed packet handling can be found in [`/pkg/server/`](/pkg/server/) along with other server logic like managing the current game. [`/pkg/game/`](/pkg/game/) has game mode logic like teams, timing, flags, and so on. Protocol definitions (like network message codes) can be found in [`pkg/protocol`](/pkg/protocol/).

Other interesting packages:

- [`pkg/protocol/cubecode`](pkg/protocol/cubecode)
- [`pkg/enet`](pkg/enet)

In [`cmd/genauth`](cmd/genauth), there is a command to generate auth keys for users. While you can use auth keys generated with Sauerbraten's `/genauthkey` command, `genauth` provides better output (`auth.cfg` line for the player, JSON object for this server's `users.json` file).

## Why?

I started this mainly as a challenge to myself and because I have ideas to improve the integration of Sauerbraten servers with other services and interfaces. For example, making the server state and game events available via WebSockets in real-time, instead of the UDP-based extinfo protocol, and integrating a third-party auth system (spanning multiple servers).

Writing a server that makes it easy to modify gameplay is not one of the goals of this project, neither is plugin support, although it might happen at some point. If you want that, now, use pisto's great [spaghettimod](https://github.com/pisto/spaghettimod).
