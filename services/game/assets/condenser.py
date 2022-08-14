"""
Given the output of Emscripten's file packager, make a single file with the
file list and data combined.

Usage:
python3 condenser.py [.data file] [.js file] [target filename]

Example:
python3 condenser.py /tmp/.base.data /tmp/.preload_base.js base.blob
"""
import json
import sys
import os
import re

if __name__ == "__main__":
    args = sys.argv[1:]

    if len(args) != 3:
        exit(1)

    output = args[2]

    data = open(args[0], 'rb').read()
    js = open(args[1], 'r').read()
    package = re.search('loadPackage\((.+)\)', js)

    if not package:
        exit(1)

    # We could compute these directories from the file list alone, but I'm lazy.
    paths = []
    for directory in re.finditer('createPath...(.+), true', js):
        paths.append(json.loads('[%s]' % directory[1][:-6]))

    directories = json.dumps(paths)
    metadata = package[1]

    with open(output, 'wb') as out:
        out.write(len(directories).to_bytes(4, 'big'))
        out.write(bytes(directories, 'utf-8'))
        out.write(len(metadata).to_bytes(4, 'big'))
        out.write(bytes(metadata, 'utf-8'))
        out.write(data)
