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
from typing import NamedTuple, Optional, Tuple, List, Set, Dict

# A mapping from a file on the filesystem to its target in Sour's filesystem.
# Example: ("/home/cfoust/Downloads/blah.ogz", "packages/base/blah.ogz")
Mapping = Tuple[str, str]


class Asset(NamedTuple):
    # The hash of the asset's file contents. Also used as a unique reference.
    id: str
    # Where the asset appears in the filesystem.
    path: str


class Mod(NamedTuple):
    name: str
    # A list of asset IDs
    assets: List[Asset]


class GameMap(NamedTuple):
    """
    Represents a single game map and the data needed to load it.
    """
    # For maps, we hash both the .ogz and the .cfg.
    id: str
    # The map name as it would appear in Sauerbraten e.g. complex.
    name: str
    # The asset ID of the map file.
    ogz: str
    assets: List[Asset]
    # An optional image that can be used in the map browser.  Usually this is
    # the mapshot (Sauer's term) that appears on the loading screen, but some
    # Quadropolis maps provided screenshots by other means.
    image: Optional[str]
    # A description of the map that can be shown to the user.
    description: str


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
            "./sourdump",
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


class Packager:
    outdir: str
    assets: Set[str]
    maps: List[GameMap]

    def __init__(self, outdir: str):
        self.outdir = outdir
        self.assets = set()
        self.maps = []


    def build_asset(self, file: Mapping, compress_images: bool = True) -> Optional[Asset]:
        _in, out = file
        _, extension = path.splitext(_in)

        os.makedirs("working/", exist_ok=True)

        if not path.exists(_in):
            return None

        file_hash = hash_file(_in)
        out_file = path.join(self.outdir, file_hash)
        asset = Asset(path=out, id=file_hash)

        if path.exists(out_file):
            return asset

        size = path.getsize(_in)

        # We can only compress certain file types
        if (
            extension not in [".dds", ".jpg", ".png"] or
            size < 128000 or
            not compress_images
        ):
            shutil.copy(_in, out_file)
            return asset

        compressed = path.join(
            "working/",
            "%s%s" % (
                file_hash,
                extension
            )
        )

        if path.exists(compressed):
            shutil.copy(compressed, out_file)
            return Asset(
                path=out,
                id=hash_file(compressed)
            )

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
        return Asset(
            path=out,
            id=hash_file(compressed)
        )


    def build_assets(
        self,
        files: List[Mapping],
        compress_images: bool = True,
    ) -> List[Asset]:
        """
        Given a list of files and a destination, build Sour-compatible bundles.
        Images are compressed by default, but you can disable this with
        `compress_images`.
        """

        # We may remap files after conversion
        cleaned: List[Asset] = []

        for file in files:
            asset = self.build_asset(file, compress_images=compress_images)

            if not asset:
                continue

            self.assets.add(asset.id)
            cleaned.append(asset)

        return cleaned


    def build_map(
        self,
        roots: List[str],
        skip_root: str,
        map_file: str,
        name: str,
        description: str,
        image: str = None,
    ) -> Optional[GameMap]:
        """
        Given a map file, roots, and an output directory, create a Sour bundle for
        the map and return its hash.

        `skip_root` is one of the roots from `roots`. If a file from the map's
        files exists in that root, it will be skipped when creating the vanilla zip
        file.
        """
        map_files = get_map_files(map_file, roots)
        assets = self.build_assets(map_files)

        base, _ = path.splitext(map_file)

        map_hash_files = [map_file]
        cfg = "%s.cfg" % (base)
        if path.exists(cfg):
            map_hash_files.append(cfg)

        map_hash = hash_files(map_hash_files)

        if not image:
            # Look for an image file adjacent to the map
            for extension in ['.png', '.jpg']:
                result = "%s%s" % (base, extension)
                if not path.exists(result): continue
                image = "%s%s" % (hash_file(result), extension)
                shutil.copy(result, path.join(self.outdir, image))

        ogz_id = None
        for asset in assets:
            if asset.path.endswith('.ogz'):
                ogz_id = asset.id

        if not ogz_id:
            return None

        map_ = GameMap(
            id=map_hash,
            name=name,
            ogz=ogz_id,
            assets=assets,
            image=image,
            description=description,
        )

        self.maps.append(map_)

        return map_


class IndexAsset(NamedTuple):
    id: int
    path: str


class IndexMap(NamedTuple):
    id: str
    name: str
    ogz: int
    assets: List[IndexAsset]
    image: Optional[str]
    description: str


class IndexMod(NamedTuple):
    name: str
    assets: List[IndexAsset]


def dump_index(maps: List[GameMap], mods: List[Mod], assets: List[str], outdir: str, prefix = ''):
    index = '%s.index.json' % prefix

    lookup: Dict[str, int] = {}
    for i, asset in enumerate(assets):
        lookup[asset] = i

    def replace_asset(asset: Asset) -> IndexAsset:
        return IndexAsset(
            id=lookup[asset.id],
            path=asset.path,
        )

    index_maps: List[IndexMap] = list(map(
        lambda map_: IndexMap(
            id=map_.id,
            name=map_.name,
            ogz=lookup[map_.ogz],
            assets=list(map(replace_asset, map_.assets)),
            image=map_.image,
            description=map_.description,
        ),
        maps
    ))

    index_mods: List[IndexMod] = list(map(
        lambda mod: IndexMod(
            name=mod.name,
            assets=list(map(replace_asset, mod.assets)),
        ),
        mods
    ))

    with open(path.join(outdir, index), 'w') as f:
        f.write(json.dumps(
            {
                'assets': assets,
                'maps': list(map(lambda _map: _map._asdict(), index_maps)),
                'mods': list(map(lambda mod: mod._asdict(), index_mods)),
            },
            indent=4
        ))


if __name__ == "__main__": pass
