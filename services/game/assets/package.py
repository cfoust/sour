"""
Library for building and managing asset packs for Sour.
"""

from os import path
import hashlib
import json
import os
import re
import subprocess
import sys

# A mapping from a file on the filesystem to its target in Sour's filesystem.
# Example: ("/home/cfoust/Downloads/blah.ogz", "packages/base/blah.ogz")
Mapping = tuple[str, str]


def hash_string(string: str) -> str:
    return hashlib.sha256(string.encode('utf-8')).hexdigest()


def combine_bundle(data_file: str, js_file: str, dest: str):
    """
    Given the output of Emscripten's file packager, make a single file with the
    file list and data combined.
    """

    data = open(data_file, 'rb').read()
    js = open(js_file, 'r').read()
    package = re.search('loadPackage\((.+)\)', js)

    if not package:
        raise Exception("Failed to find loadPackage in %s" % js_file)

    # We could compute these directories from the file list alone, but I'm
    # lazy.
    paths = []
    for directory in re.finditer('createPath...(.+), true', js):
        paths.append(json.loads('[%s]' % directory[1][:-6]))

    directories = json.dumps(paths)
    metadata = package[1]

    with open(dest, 'wb') as out:
        out.write(len(directories).to_bytes(4, 'big'))
        out.write(bytes(directories, 'utf-8'))
        out.write(len(metadata).to_bytes(4, 'big'))
        out.write(bytes(metadata, 'utf-8'))
        out.write(data)


def build_bundle(files: list[Mapping], dest: str, compress_images: bool = True):
    """
    Given a list of files and a destination, build a Sour-compatible bundle.
    Images are compressed by default, but you can disable this with `compress_images`.
    """

    dest_hash = hash_string(dest)

    # We may remap files after conversion
    cleaned: list[Mapping] = []

    for _in, out in files:
        _, extension = path.splitext(_in)

        if not path.exists(_in): continue

        size = path.getsize(_in)

        # We can only compress certain file types
        if (
            not extension in [".dds", ".jpg", ".png"] or
            size < 128000
        ):
            cleaned.append((_in, out))
            continue

        # If multiple bundles rely on the same converted image, we don't want to
        # redo the calculation.
        compressed = path.join(
            "working/",
            "%s%s" % (
                hash_string(_in),
                extension
            )
        )

        if path.exists(compressed):
            cleaned.append((compressed, out))
            continue

        # Make the image 1/4 of the size using ImageMagick
        for _from, _to in [
                (_in, compressed),
                (compressed, compressed)
        ]:
            subprocess.run(
                [
                    "convert",
                    _from,
                    "-resize 50%",
                    _to
                ],
                check=True
            )

        cleaned.append((compressed, out))

    js_file = "/tmp/preload_%s.js" % dest_hash
    data_file = "/tmp/%s.data" % dest_hash

    result = subprocess.run(
        [
            "python3",
            "%s/upstream/emscripten/tools/file_packager.py" % os.environ['EMSDK_DIR'],
            data_file,
            "--use-preload-plugins",
            "--preload",
            *list(map(
                lambda v: "%s@%s" % (v[0], v[1]),
                cleaned
            )),
        ],
        check=True,
        capture_output=True
    )

    with open(js_file, 'wb') as f: f.write(result.stdout)

    combine_bundle(data_file, js_file, dest)


def get_map_files(map_file: str, roots: list[str]) -> list[Mapping]:
    """
    Get all of the files referenced by a Sauerbraten map.
    """
    root_args = []

    for root in roots:
        root_args.append("-root")
        root_args.append(root)

    result = subprocess.run(
        [
            "./mapdump",
            *root_args,
            map_file,
        ],
        check=True,
        capture_output=True
    )

    files: list[Mapping] = []
    for line in result.stdout.decode('utf-8').split('\n'):
        parts = line.split('->')

        if len(parts) != 2: continue
        files.append((parts[0], parts[1]))

    return files


if __name__ == "__main__": pass
