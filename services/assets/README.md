# All about assets

## Introduction

Sauerbraten assumes that all of the game assets it needs are already on the filesystem. In other words, the desktop version of the game comes with all of the textures, sounds, and 3D models the user might want to load. The current version of the game contains about a gigabyte of assets, everything necessary to load the game's 300 maps and run all of its game modes.

This poses extreme problems for running Sauerbraten on the web. While devices are powerful and bandwidth is cheap these days, sending a gigabyte of assets along when the page loads is impractical and undesirable. Since the goal of Sour is to allow the user to do anything the desktop version of the game offers, we need to devise a way to only load the files the game needs when it needs them.

The most important use of this is loading game maps. It's harder than it sounds. Emscripten, the toolkit for building C++ applications for the web, has a mechanism for packaging sets of files (bundles) and loading them on demand. Unfortunately, that still leaves us with the problem of deciding what files should be included for each map. For the sake of posterity I will outline the approaches I took to solving this problem, which can be summarized as:
1. Use `strace` to detect what files the desktop version of the game reads when loading a map.
2. Parse the real map file and calculate all of the files it actually references.

## Approach #1: strace

`strace` is a Linux program that allows you to log all of the system calls a program makes during operation. For our purposes we only had to look at file reads (e.g. when Sauer loaded a texture from the filesystem) then map that file into the correct place in the web version's filesystem. I wrote a script that ran Sauerbraten, loaded the map we were interested in generating, and built a bundle containing all of the files it read from while it loaded.

While this sounds like it would solve the problem, it had serious drawbacks:
* Sauerbraten loads some textures and sounds on demand. Some game settings even affect which assets are actually loaded. This meant that critical assets were just missing from the bundle.
* The generation process was cumbersome and time-consuming because I had to be at a Linux machine with an X server running. Loading up the game ~300 times, even on a powerful gaming computer, was still slow. This also meant that building maps in CI or on contributors' machines was impractical.

In addition, my goal with Sour was to allow the player to load _any_ map that has ever been created for the game. I wanted it to be fast and easy to generate bundles for any arbitrary Cube 2 map.

## Approach #2: Parse the map files

The Sauerbraten map format, it turns out, is a bit of a nightmare. I was able to find [a](https://incoherency.co.uk/interest/sauer_map.html
) [few](https://github.com/SalatielSauer/OGZ-Editor) [projects](https://github.com/sauerbraten/genserverogz) [around](https://github.com/bsummer4/ogz) the web for reading or writing them, but nothing that was sophisticated enough to read the data that I needed. Therefore I had to start from scratch.

I wrote a Go program (`sourdump`) by porting much of Sauerbraten's own map loading code to Go. The program does two things:
1. Parses the actual map file (`.ogz`) to determine what texture slots (here: vslots) are actually used on cube faces in the map.
2. Recursively reads the map file's `.cfg` file to (a) establish the available texture slots and (b) make a note of any files like sounds, map models, and sky boxes the map uses.

It then produces mappings for all of the files the map referenced from a path on your filesystem (e.g. `/maps/complex.ogz`) into the game's filesystem (e.g. `/packages/base/complex.ogz`).

If you are interested in this topic, I recommend reading the program's source code. It's a 2,000-line doozy, I'll have to warn you. Because I was porting code that I didn't understand, I had to mimic the structure of Sauerbraten's code rather closely, which at times made the Go code rather unreadable.

## Packaging up the assets

I wrote a small Python library (`package.py`) that makes generating content for Sour easy. We use Emscripten's [file_packager.py](https://github.com/emscripten-core/emscripten/blob/dcfb771db8dae4955708d516de871dfdfc2ef393/tools/file_packager.py) utility to create bundles (also referred to in the code as blobs) -- basically `.tar` files -- of content that we can deliver to the browser and mount on demand. Each map bundle contains the minimum set of files necessary to render the map and no more. This is to minimize the data we need to send over the wire to clients. In addition, we compress images above 128k in size using ImageMagick.

It is worth noting that Sour uses [its own file format](https://github.com/cfoust/sour/blob/0eab96c89d863eccf63d4f209a96345e3f631f8d/services/assets/package.py#L50) (`.sour`) for storing content. Emscripten's file packager outputs a lot of superfluous data and code and I wanted asset bundles to be single files for simplicity's sake. If the map includes a mapshot, that is included alongside the bundle. Asset bundles are given names corresponding to _the byte-level hash of their contents_. Because we save assets to IndexedDB on the front end according to their hash, if a bundle has not changed we do not want the client to have to fetch it again.

This repository includes code for generating content from two sources:
* `base.py`: Generates maps from the base game, ie the version of game assets as they exist in Sauerbraten's svn repository. By default this script generates Sour blobs for all of the maps, but you can generate just a subset by providing them as arguments to the script a la `python3 base.py dust2 alithia`. You'll need to run `./setup` before running `base.py`.
* `quadropolis.py`: I built a dataset of all of the content from [Quadropolis](http://quadropolis.us/) totaling ~1,800 maps. This script turns that dataset into Sour assets by finding all of the maps and building the ones that it can. As of writing, this generates Sour bundles for 1,400 maps. Be warned that this ends up being about 12GB on disk. It's easiest to run this in the cloud so as to be colocated with your storage. Run `./setup-quad` before running `quadropolis.py`.

For most users' purposes, using `base.py` is all that is necessary. Generally speaking, generating assets is easiest in Gitpod and Docker (using the `+assets` image built by Earthly). I wouldn't recommend trying to generate these in your host as the system dependencies are heavy and annoying.

When building assets, you can (really, _should_) set the `PREFIX` variable to determine the prefix of the index file (described below). (For example: `PREFIX=$(git rev-parse --short HEAD) python3 base.py`). This is to ensure that the user's browser (or your CDN) does not load the index file from the cache.

## Asset sources and index files

The Sour game client understands what assets it can load by looking at index files, which are JSON files that describe the content an asset source makes available. (Look at [`dump_index` in `package.py`](https://github.com/cfoust/sour/blob/0eab96c89d863eccf63d4f209a96345e3f631f8d/services/assets/package.py#L269) to see what these look like. Each index file contains a listing of all of the maps (and mods, currently only used for the game's basic assets) available in that asset source.

Asset sources are specified at runtime using the `ASSET_SOURCE` environment variable. Importantly, you can specify multiple sources and Sour will search for assets in the order the sources appear.

```bash
# Valid ASSET_SOURCES:
######################

# /assets/.index.json is the asset source that comes baked into the image. Generally you want this even if you're using your own map sources; this is because Sour automatically loads the `base` bundle, which contains all of the basic assets necessary to run the game, like main menu graphics.
ASSET_SOURCE="/assets/.index.json"

# Asset sources are separated by single semicolons.
ASSET_SOURCE="/assets/.index.json;https://example.com/2bfc017.index.json"
# As an example, if a user runs `/map complex`, Sour first searches /assets/.index.json; if there is a `complex` map, it loads that verssion even if one also exists in the second source.

# In production (sourga.me) the ASSET_SOURCE looks like this:
ASSET_SOURCE="/assets/.index.json;https://static.sourga.me/blobs/XXXXX.index.json;https://static.sourga.me/quadropolis/XXXXX.index.json"
# In other words, Sour will load maps that appear in the latest SVN version of the game _first_, then from Quadropolis if the map did not appear in the base game.
```

Everything related to assets is handled in the [assets Web Worker](https://github.com/cfoust/sour/blob/main/services/client/src/assets/worker.ts). We cache asset bundles to IndexedDB, which is too slow to use in the rendering thread.
