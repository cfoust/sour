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

I wrote a Go program by porting much of Sauerbraten's own map loading code to Go. The program does two things:
1. Parses the actual map file (`.ogz`) to determine what texture slots (here: vslots) are actually used on cube faces in the map.
2. Recursively reads the map file's `.cfg` file to (a) establish the available texture slots and (b) make a note of any files like sounds, map models, and sky boxes the map uses.

It then produces mappings for all of the files the map referenced from a path on your filesystem (e.g. `/maps/complex.ogz`) into the game's filesystem (e.g. `/packages/base/complex.ogz`).

If you are interested in this topic, I recommend reading the program's source code. It's a 2,000-line doozy, I'll have to warn you. Because I was porting code that I didn't understand, I had to mimic the structure of Sauerbraten's code rather closely, which at times made the Go code rather unreadable.
