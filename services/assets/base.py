import package
import sys
import glob
from os import path
import os
from typing import NamedTuple, Optional, Tuple, List, Dict, Set
import subprocess

if __name__ == "__main__":
    args = sys.argv

    maps = glob.glob('roots/base/packages/base/*.ogz')

    if len(args) > 1:
        maps = list(map(lambda a: "roots/base/packages/base/%s.ogz" % a, args[1:]))

    outdir = os.getenv("ASSET_OUTPUT_DIR", "output/")
    prefix = os.getenv("PREFIX", "")
    os.makedirs(outdir, exist_ok=True)

    p = package.Packager(outdir)

    MODEL_TYPES = [
        "md2",
        "md3",
        "md5",
        "obj",
        "smd",
        "iqm"
    ]

    models: List[str] = []
    for type_ in MODEL_TYPES:
        models += glob.glob(
            f'roots/base/packages/models/**/**/{type_}.cfg'
        )

    roots = [
        "sour",
        "roots/base",
    ]

    mods: List[package.Mod] = []

    assets: Set[str] = set()

    def fill_assets(new_assets: List[package.Asset]):
        for asset in new_assets:
            if asset.id in assets:
                continue
            assets.add(asset.id)

    skip_root = roots[1]

    # Build base
    with open("base.list", "r") as f:
        files = f.read().split("\n")

        mappings: List[package.Mapping] = []
        for file in files:
            mapping = package.search_file(file, roots)
            if not mapping or path.isdir(mapping[0]): continue
            mappings.append(mapping)

        p.build_mod(
            skip_root,
            mappings,
            "base",
            "Everything the base game needs.",
            compress_images=False,
        )

    for _map in maps:
        base, _ = path.splitext(path.basename(_map))
        print("Building %s" % base)
        p.build_map(
            roots,
            skip_root,
            _map,
            base,
            """Base game map %s as it appeared in game version r6584.
            """ % base,
        )

    p.dump_index(prefix)
