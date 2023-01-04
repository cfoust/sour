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


class Bundle(NamedTuple):
    id: str
    assets: List[Asset]


class Mod(NamedTuple):
    # A mod's id is equivalent to its bundle id
    id: str
    name: str
    image: Optional[str]
    description: str


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


class Model(NamedTuple):
    # The hash of the model and all of its contents
    # Refers to a bundle
    id: str
    # The path that Sauer would use, a reference to a directory in
    # package/models e.g. skull/blue
    name: str


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

    os.remove(data_file)
    os.remove(js_file)



def build_sour_bundle(
    bundle: Bundle,
    outdir: str,
    compress_images: bool = True,
    print_summary: bool = False
) -> str:
    """
    Given a list of files and a destination, build a Sour-compatible bundle.
    Images are compressed by default, but you can disable this with
    `compress_images`.
    """

    # We may remap files after conversion
    cleaned: List[Mapping] = []

    sour_target = path.join(outdir, "%s.sour" % bundle.id)

    if path.exists(sour_target):
        return bundle.id

    sizes: List[Tuple[str, int]] = []

    for asset in bundle.assets:
        _in = path.join(outdir, asset.id)
        out = asset.path
        _, extension = path.splitext(_in)

        if not path.exists(_in):
            continue

        size = path.getsize(_in)

        sizes.append((_in, size))

        # We can only compress certain file types
        if (
            extension not in [".dds", ".jpg", ".png"] or
            size < 128000 or
            not compress_images
        ):
            cleaned.append((_in, out))
            continue

        os.makedirs("working/", exist_ok=True)

        # If multiple bundles rely on the same converted image, we don't want
        # to redo the calculation.
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

    if print_summary:
        sizes = list(reversed(sorted(sizes, key=lambda a: a[1])))

        for _in, size in sizes:
            print(f"{_in} {sizeof_fmt(size)}")

    js_file = "/tmp/preload_%s.js" % bundle.id
    data_file = "/tmp/%s.data" % bundle.id

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

    combine_bundle(data_file, js_file, sour_target)
    return bundle.id


def build_desktop_bundle(outdir: str, bundle: Bundle):
    added: Set[str] = set()
    zip_path = path.join(outdir, "%s.desktop" % bundle.id)
    with zipfile.ZipFile(
        zip_path,
        'w',
        compression=zipfile.ZIP_DEFLATED,
        compresslevel=9
    ) as desktop:
        for asset in bundle.assets:
            _in = path.join(outdir, asset.id)
            _out = asset.path

            if _out in added:
                continue

            with desktop.open(_out, 'w') as outfile:
                with open(_in, 'rb') as infile:
                    outfile.write(infile.read())

            added.add(_out)


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

    bundles: List[Bundle]
    maps: List[GameMap]
    models: List[Model]
    mods: List[Mod]
    textures: List[Asset]

    def __init__(self, outdir: str):
        self.outdir = outdir
        self.assets = set()
        self.bundles = []
        self.maps = []
        self.models = []
        self.mods = []
        self.textures = []


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


# def dump_index(maps: List[GameMap], mods: List[Mod], assets: List[str], outdir: str, prefix = ''):
    # index = '%s.index.json' % prefix

    # lookup: Dict[str, int] = {}
    # for i, asset in enumerate(assets):
        # lookup[asset] = i

    # def replace_asset(asset: Asset) -> IndexAsset:
        # return IndexAsset(
            # id=lookup[asset.id],
            # path=asset.path,
        # )

    # index_maps: List[IndexMap] = list(map(
        # lambda map_: IndexMap(
            # id=map_.id,
            # name=map_.name,
            # ogz=lookup[map_.ogz],
            # assets=list(map(replace_asset, map_.assets)),
            # image=map_.image,
            # description=map_.description,
        # ),
        # maps
    # ))

    # index_mods: List[IndexMod] = list(map(
        # lambda mod: IndexMod(
            # name=mod.name,
            # assets=list(map(replace_asset, mod.assets)),
        # ),
        # mods
    # ))

    # with open(path.join(outdir, index), 'w') as f:
        # f.write(json.dumps(
            # {
                # 'assets': assets,
                # 'maps': list(map(lambda _map: _map._asdict(), index_maps)),
                # 'mods': list(map(lambda mod: mod._asdict(), index_mods)),
            # },
            # indent=4
        # ))


if __name__ == "__main__": pass
