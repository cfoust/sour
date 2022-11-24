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
from typing import NamedTuple, Optional, Tuple, List

# A mapping from a file on the filesystem to its target in Sour's filesystem.
# Example: ("/home/cfoust/Downloads/blah.ogz", "packages/base/blah.ogz")
Mapping = Tuple[str, str]

class Bundle(NamedTuple):
    # This is the hash of the bundle contents.
    hash: str


class GameMap(NamedTuple):
    """
    Represents a single game map and the data needed to load it.
    """
    # The map name as it would appear in Sauerbraten e.g. complex.
    name: str
    # This refers to the hash of the bundle that contains this map's data.
    bundle: str
    # An optional image that can be used in the map browser.  Usually this is
    # the mapshot (Sauer's term) that appears on the loading screen, but some
    # Quadropolis maps provided screenshots by other means.
    image: Optional[str]
    # A description of the map that can be shown to the user.
    description: str
    # To avoid naming collisions, maps can specify aliases so that they can
    # always be referenced. This is just used for specialized datasets like
    # Quadropolis.
    aliases: List[str]


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


def hash_files(files: List[str]) -> str:
    tar = subprocess.Popen([
        'tar',
        'cf',
        '-',
        '--ignore-failed-read',
        '--sort=name',
        "--mtime=UTC 2019-01-01",
        "--group=0",
        "--owner=0",
        "--numeric-owner",
        *files,
    ], stdout=subprocess.PIPE, stderr=subprocess.DEVNULL)
    sha = subprocess.check_output(['sha256sum'], stdin=tar.stdout)
    tar.wait()
    return sha.decode('utf-8').split(' ')[0]


def search_file(file: str, roots: List[str]) -> Optional[Mapping]:
    for root in roots:
        unprefixed = path.join(root, file)
        prefixed = path.join(root, "packages", file)

        if path.exists(unprefixed):
            return (
                unprefixed,
                path.relpath(unprefixed, root)
            )
        if path.exists(prefixed):
            return (
                prefixed,
                path.relpath(prefixed, root)
            )

    return None


def build_bundle(files: List[Mapping], outdir: str, compress_images: bool = True) -> str:
    """
    Given a list of files and a destination, build a Sour-compatible bundle.
    Images are compressed by default, but you can disable this with `compress_images`.
    """

    bundle_hash = hash_files(list(map(lambda a: a[0], files)))

    # We may remap files after conversion
    cleaned: List[Mapping] = []

    for _in, out in files:
        _, extension = path.splitext(_in)

        if not path.exists(_in): continue

        size = path.getsize(_in)

        # We can only compress certain file types
        if (
            not extension in [".dds", ".jpg", ".png"] or
            size < 128000 or
            not compress_images
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
                    "-resize",
                    "50%",
                    _to
                ],
                check=True
            )

        cleaned.append((compressed, out))

    js_file = "/tmp/preload_%s.js" % bundle_hash
    data_file = "/tmp/%s.data" % bundle_hash

    result = subprocess.run(
        [
            "python3",
            "%s/upstream/emscripten/tools/file_packager.py" % os.getenv('EMSDK', '/emsdk'),
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

    combine_bundle(data_file, js_file, path.join(outdir, "%s.sour" % bundle_hash))
    return bundle_hash


def get_map_files(map_file: str, roots: List[str]) -> List[Mapping]:
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
        # check=True,
        capture_output=True
    )

    if result.returncode != 0:
        raise Exception(result.stderr)

    files: List[Mapping] = []
    for line in result.stdout.decode('utf-8').split('\n'):
        parts = line.split('->')

        if len(parts) != 2: continue
        files.append((parts[0], parts[1]))

    return files


def build_map_bundle(map_file: str, roots: List[str], outdir: str) -> str:
    """
    Given a map file, roots, and an output directory, create a Sour bundle for
    the map and return its hash.
    """
    files = get_map_files(map_file, roots)
    print(files)
    return build_bundle(files, outdir)


if __name__ == "__main__": pass
