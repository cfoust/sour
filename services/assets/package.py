"""
Library for building and managing asset packs for Sour.
"""

from os import path
import hashlib
import json
import os
import re
import shutil
import subprocess
import sys
import zipfile
from typing import NamedTuple, Optional, Tuple, List, Set

# A mapping from a file on the filesystem to its target in Sour's filesystem.
# Example: ("/home/cfoust/Downloads/blah.ogz", "packages/base/blah.ogz")
Mapping = Tuple[str, str]


class Asset(NamedTuple):
    path: str
    hash: str


class Mod(NamedTuple):
    name: str
    # This is the hash of the bundle contents.
    assets: List[Asset]


class BuiltMap(NamedTuple):
    image: Optional[str]
    assets: List[Asset]


class GameMap(NamedTuple):
    """
    Represents a single game map and the data needed to load it.
    """
    # The map name as it would appear in Sauerbraten e.g. complex.
    name: str
    assets: List[Asset]
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


def hash_files(files: List[str]) -> str:
    if len(files) == 1:
        sha = subprocess.check_output(['sha256sum', files[0]])
        return sha.decode('utf-8').split(' ')[0]

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


def hash_file(file: str) -> str:
    return hash_files([file])


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

# https://stackoverflow.com/a/1094933
def sizeof_fmt(num, suffix="B"):
    for unit in ["", "Ki", "Mi", "Gi", "Ti", "Pi", "Ei", "Zi"]:
        if abs(num) < 1024.0:
            return f"{num:3.1f}{unit}{suffix}"
        num /= 1024.0
    return f"{num:.1f}Yi{suffix}"


def build_assets(
    files: List[Mapping],
    outdir: str,
    compress_images: bool = True,
) -> List[Asset]:
    """
    Given a list of files and a destination, build Sour-compatible bundles.
    Images are compressed by default, but you can disable this with
    `compress_images`.
    """

    # We may remap files after conversion
    cleaned: List[Asset] = []

    os.makedirs("working/", exist_ok=True)

    for _in, out in files:
        _, extension = path.splitext(_in)

        if not path.exists(_in):
            continue

        file_hash = hash_file(_in)
        out_file = path.join(outdir, file_hash)
        asset = Asset(path=out, hash=file_hash)

        if path.exists(out_file):
            cleaned.append(asset)
            continue

        size = path.getsize(_in)

        # We can only compress certain file types
        if (
            extension not in [".dds", ".jpg", ".png"] or
            size < 128000 or
            not compress_images
        ):
            shutil.copy(_in, out_file)
            cleaned.append(asset)
            continue

        compressed = path.join(
            "working/",
            "%s%s" % (
                file_hash,
                extension
            )
        )

        if path.exists(compressed):
            shutil.copy(compressed, out_file)
            cleaned.append(Asset(
                path=out,
                hash=hash_file(compressed)
            ))
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

        shutil.copy(compressed, out_file)
        cleaned.append(Asset(
            path=out,
            hash=hash_file(compressed)
        ))

    return cleaned


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
        if path.isdir(parts[0]): continue
        files.append((parts[0], parts[1]))

    return files


def build_map_assets(map_file: str, roots: List[str], outdir: str, skip_root: str) -> BuiltMap:
    """
    Given a map file, roots, and an output directory, create a Sour bundle for
    the map and return its hash.

    `skip_root` is one of the roots from `roots`. If a file from the map's
    files exists in that root, it will be skipped when creating the vanilla zip
    file.
    """
    map_files = get_map_files(map_file, roots)
    assets = build_assets(map_files, outdir)
    return BuiltMap(assets=assets, image=None)


def dump_index(maps: List[GameMap], mods: List[Mod], outdir: str, prefix = ''):
    index = '%s.index.json' % prefix

    with open(path.join(outdir, index), 'w') as f:
        f.write(json.dumps(
            {
                'maps': list(map(lambda _map: _map._asdict(), maps)),
                'mods': list(map(lambda mod: mod._asdict(), mods)),
            },
            indent=4
        ))


if __name__ == "__main__": pass
