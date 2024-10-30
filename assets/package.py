"""
Library for building and managing asset packs for Sour.
"""

from os import path
from pathlib import Path
import hashlib
import json
import cbor2
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
    desktop: bool
    web: bool


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
    # The bundle that contains this map's files.
    bundle: Optional[str]
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


class IndexAsset(NamedTuple):
    id: int
    path: str


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


def get_root_relative(file: str, roots: List[str]) -> Optional[str]:
    for root in roots:
        relative = path.relpath(file, root)

        if '..' in relative:
            continue

        return relative

    return None


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
    outdir: str,
    bundle: Bundle,
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

    for asset in bundle.assets:
        _in = path.join(outdir, asset.id)
        out = asset.path
        cleaned.append((_in, out))

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
        capture_output=True
    )

    if result.returncode != 0:
        raise Exception(result.stderr)

    with open(js_file, 'wb') as f: f.write(result.stdout)

    combine_bundle(data_file, js_file, sour_target)
    return bundle.id


def build_desktop_bundle(outdir: str, bundle: Bundle):
    added: Set[str] = set()
    zip_path = path.join(outdir, "%s.desktop" % bundle.id)

    if path.exists(zip_path):
        return

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


def run_sourdump(roots: List[str], args: List[str]) -> str:
    root_args = []

    for root in roots:
        root_args.append("-root")
        root_args.append(root)

    args = [
        "./sourdump",
        *root_args,
        *args,
    ]
    result = subprocess.run(
        args,
        # check=True,
        capture_output=True
    )

    if result.returncode != 0:
        raise Exception(result.stderr.decode('utf-8') + result.stdout.decode('utf-8'))

    return result.stdout.decode('utf-8')


def get_root_files(roots: List[str]) -> List[str]:
    files: List[str] = []
    for root in roots:
        if root.startswith("http"):
            out = run_sourdump(roots, [
                "list",
            ])

            files = files + out.strip().split("\n")
            continue

        for file in list(map(lambda a: str(a), Path(root).rglob('*'))):
            relative = file[len(root)+1:]
            if not path.isfile(file):
                continue
            files.append(relative)

    return files


def query_files(roots: List[str], files: List[str]) -> List[Mapping]:
    """
    Given a list of files, attempt to resolve those files to remote assets or
    paths on the local filesystem.
    """
    output = run_sourdump(roots, [
        "query",
        *files,
    ])

    resolved: List[Mapping] = []
    for line in output.strip().split("\n"):
        parts = line.split('->')
        # These are reversed when you query
        resolved.append((parts[1], parts[0]))

    return resolved


def dump_sour(type_: str, target: str, roots: List[str]) -> List[Mapping]:
    out = run_sourdump(roots, [
        "dump",
        "-type",
        type_,
        target,
    ])

    files: List[Mapping] = []
    for line in out.split('\n'):
        parts = line.split('->')

        if len(parts) != 2: continue
        if path.isdir(parts[0]): continue
        files.append((parts[0], parts[1]))

    return files


def get_map_files(map_file: str, roots: List[str]) -> List[Mapping]:
    """
    Get all of the files referenced by a Sauerbraten map.
    """
    return dump_sour("map", map_file, roots)


def download_assets(roots: List[str], outdir: str, assets: List[str]):
    run_sourdump(roots, [
        "download",
        "--outdir",
        outdir,
        *assets,
    ])


def hash_assets(roots: List[str], assets: List[str]):
    return run_sourdump(roots, [
        "hash",
        *assets,
    ])


MODEL_PREFIX = "packages/models"


class BuildParams(NamedTuple):
    roots: List[str]
    skip_root: str
    compress_images: bool
    download_assets: bool
    build_web: bool
    build_desktop: bool


class Packager:
    outdir: str
    assets: Set[str]

    refs: List[Asset]
    bundles: List[Bundle]
    maps: List[GameMap]
    models: List[Model]
    mods: List[Mod]
    textures: List[Asset]

    def __init__(self, outdir: str):
        self.outdir = outdir
        self.assets = set()
        self.refs = []
        self.bundles = []
        self.maps = []
        self.models = []
        self.mods = []
        self.textures = []


    def build_asset(
        self,
        params: BuildParams,
        file: Mapping,
    ) -> Optional[Asset]:
        _in, out = file
        _, extension = path.splitext(_in)

        if _in == "nil":
            return None

        os.makedirs("working/", exist_ok=True)

        if _in.startswith("id:"):
            id_ = _in[3:]
            asset = Asset(path=out, id=id_)
            if params.download_assets:
                download_assets(params.roots, self.outdir, [id_])
                self.assets.add(asset.id)
            return asset

        # Remove the fs: bit
        _in = _in[3:]

        if not path.exists(_in) or not path.isfile(_in):
            return None

        file_hash = hash_file(_in)
        out_file = path.join(self.outdir, file_hash)
        asset = Asset(path=out, id=file_hash)

        if path.exists(out_file):
            self.assets.add(asset.id)
            return asset

        size = path.getsize(_in)

        # We can only compress certain file types
        if (
            extension not in [".dds", ".jpg", ".png"] or
            size < 128000 or
            not params.compress_images
        ):
            shutil.copy(_in, out_file)
            self.assets.add(asset.id)
            return asset

        compressed = path.join(
            "working/",
            "%s%s" % (
                file_hash,
                extension
            )
        )

        if path.exists(compressed):
            shutil.copy(compressed, path.join(self.outdir, asset.id))
            self.assets.add(asset.id)
            return asset

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

        shutil.copy(compressed, path.join(self.outdir, asset.id))
        self.assets.add(asset.id)
        return asset


    def build_ref(
        self,
        params: BuildParams,
        file: Mapping,
    ) -> Optional[Asset]:
        asset = self.build_asset(
            params,
            file,
        )

        if not asset:
            return None

        self.refs.append(asset)
        return asset


    def build_assets(
        self,
        params: BuildParams,
        files: List[Mapping],
    ) -> List[Asset]:
        """
        Given a list of files and a destination, build Sour-compatible assets.
        Images are uncompressed by default, but you can disable this with
        `compress_images`.
        """

        # We may remap files after conversion
        cleaned: List[Asset] = []

        for file in files:
            asset = self.build_asset(
                params,
                file,
            )

            if not asset:
                continue

            cleaned.append(asset)

        return cleaned


    def build_bundle(
        self,
        params: BuildParams,
        files: List[Mapping],
    ) -> Optional[Bundle]:
        assets = self.build_assets(
            params,
            files,
        )

        id_ = hash_string(
            ''.join(
                sorted(list(
                    map(
                        lambda a: a.id,
                        assets,
                    )
                ))
            )
        )

        bundle = Bundle(
            id=id_,
            assets=assets,
            desktop=params.build_desktop,
            web=params.build_web,
        )

        if params.build_web:
            build_sour_bundle(self.outdir, bundle)

        if params.build_desktop:
            desktop_bundle = bundle
            if params.skip_root:
                resolved = zip(query_files(
                    [params.skip_root],
                    list(map(lambda a: a.path, assets)),
                ), assets)

                new_assets: List[Asset] = []
                for result, asset in resolved:
                    in_, out = result
                    if in_ == "nil":
                        new_assets.append(asset)

                desktop_bundle = bundle._replace(assets=new_assets)
            build_desktop_bundle(self.outdir, desktop_bundle)

        self.bundles.append(bundle)

        return bundle


    def build_mod(
        self,
        params: BuildParams,
        files: List[Mapping],
        name: str,
        description: str,
        image: Optional[str] = None,
    ) -> Optional[Mod]:
        bundle = self.build_bundle(
            params,
            files,
        )

        if not bundle:
            return None

        mod = Mod(
            id=bundle.id,
            name=name,
            image=image,
            description=description
        )

        self.mods.append(mod)

        return mod


    def build_model(
        self,
        params: BuildParams,
        name: str,
    ) -> Model:
        model_files = dump_sour("model", name, params.roots)
        bundle = self.build_bundle(
            params,
            model_files,
        )

        if not bundle:
            raise Exception('failed to build bundle for model')

        model = Model(
            id=bundle.id,
            name=name,
        )

        self.models.append(model)

        return model


    def build_texture(
        self,
        params: BuildParams,
        file: str,
    ) -> Optional[Asset]:
        resolved = query_files(params.roots, [file])
        assets = self.build_assets(params, resolved)
        if not assets:
            return None

        texture = assets[0]
        self.textures.append(texture)
        return texture


    def build_textures(
        self,
        params: BuildParams,
        files: List[str],
    ):
        batch = 500
        for i in range(0, (len(files) // batch) + 1):
            sub = files[i * batch:(i + 1) * batch]
            resolved = query_files(params.roots, sub)
            assets = self.build_assets(params, resolved)
            if not assets:
                return None
            self.textures += assets


    def build_image(
        self,
        params: BuildParams,
        file: str,
    ) -> Optional[str]:
        _, extension = path.splitext(file)
        query = query_files(
            params.roots,
            [file]
        )
        resolved = query[0]
        if resolved[0] == "nil": return None
        asset = self.build_asset(
            params._replace(
                download_assets=True,
            ),
            resolved,
        )
        if not asset:
            return None
        result = path.join(self.outdir, asset.id)
        image = "%s%s" % (asset.id, extension)
        shutil.copy(result, path.join(self.outdir, image))
        return image


    def build_map(
        self,
        params: BuildParams,
        map_file: str,
        name: str,
        description: str,
        image: Optional[str] = None,
    ) -> Optional[GameMap]:
        """
        Given a map file, roots, and an output directory, create a Sour bundle for
        the map and return its hash.

        `skip_root` is one of the roots from `roots`. If a file from the map's
        files exists in that root, it will be skipped when creating the vanilla zip
        file.
        """
        map_files = dump_sour("map", map_file, params.roots)
        assets = self.build_assets(
            params,
            map_files,
        )

        base, _ = path.splitext(map_file)

        map_hash_files = [map_file, "%s.cfg" % (base)]
        map_hash = hash_assets(params.roots, map_hash_files)

        if not image:
            # Look for an image file adjacent to the map
            for extension in ['.png', '.jpg']:
                image_path = "%s%s" % (base, extension)
                image = self.build_image(
                    params,
                    image_path,
                )
                if image:
                    break

        ogz_id = None
        for asset in assets:
            if asset.path.endswith('.ogz'):
                ogz_id = asset.id

        if not ogz_id:
            return None

        bundle = self.build_bundle(
            params,
            map_files,
        )
        if not bundle:
            raise Exception('built bundle was missing')

        map_ = GameMap(
            id=map_hash,
            name=name,
            bundle=bundle.id,
            ogz=ogz_id,
            assets=assets,
            image=image,
            description=description,
        )

        self.maps.append(map_)

        return map_


    def dump_index(
            self,
            prefix = ''
    ) -> None:
        index = '%s.index.source' % prefix

        lookup: Dict[str, int] = {}
        for i, asset in enumerate(self.assets):
            lookup[asset] = i

        def replace_asset(asset: Asset) -> IndexAsset:
            return IndexAsset(
                id=lookup[asset.id],
                path=asset.path,
            )

        with open(path.join(self.outdir, index), 'wb') as f:
            cbor2.dump(
                {
                    'assets': list(self.assets),
                    'textures': self.textures,
                    'refs': list(map(replace_asset, self.refs)),
                    'bundles': list(map(lambda bundle: bundle._asdict(), self.bundles)),
                    'models': list(map(lambda model: model._asdict(), self.models)),
                    'maps': list(map(lambda _map: _map._asdict(), self.maps)),
                    'mods': list(map(lambda mod: mod._asdict(), self.mods)),
                },
                f
            )


if __name__ == "__main__": pass
