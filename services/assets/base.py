import package
import sys
import glob
from os import path
import os
from typing import NamedTuple, Optional, Tuple, List, Dict
import subprocess

if __name__ == "__main__":
    args = sys.argv

    maps = glob.glob('roots/base/packages/base/*.ogz')

    if len(args) > 1:
        maps = list(map(lambda a: "roots/base/packages/base/%s.ogz" % a, args[1:]))

    outdir = os.getenv("ASSET_OUTPUT_DIR", "output/")
    prefix = os.getenv("PREFIX", "")
    os.makedirs(outdir, exist_ok=True)

    roots = [
        "sour",
        "roots/base",
    ]

    mods: List[package.Mod] = []

    assets: Dict[str, package.Asset] = {}

    def fill_assets(new_assets: List[package.Asset]):
        for asset in new_assets:
            if asset.id in assets:
                continue
            assets[asset.id] = asset

    # Build base
    with open("base.list", "r") as f:
        files = f.read().split("\n")

        mappings: List[package.Mapping] = []
        for file in files:
            mapping = package.search_file(file, roots)
            if not mapping or path.isdir(mapping[0]): continue
            mappings.append(mapping)

        base_assets = package.build_assets(
            mappings,
            outdir,
            compress_images=False,
        )
        fill_assets(base_assets)
        mods.append(
            package.Mod(
                name="base",
                assets=package.get_asset_ids(base_assets)
            )
        )

    game_maps: List[package.GameMap] = []

    for _map in maps:
        base, _ = path.splitext(path.basename(_map))
        print("Building %s" % base)
        map_bundle = package.build_map_assets(
            _map,
            roots,
            outdir,
            roots[1]
        )
        if not map_bundle:
            raise Exception('map bundle was None')

        fill_assets(map_bundle.assets)
        game_maps.append(
            package.GameMap(
                id=map_bundle.id,
                name=base,
                ogz=map_bundle.ogz,
                assets=package.get_asset_ids(map_bundle.assets),
                image=map_bundle.image,
                description="""Base game map %s as it appeared in game version r6584.
""" % base,
            )
        )

    asset_list = [v for _, v in assets.items()]

    package.dump_index(game_maps, mods, asset_list, outdir, prefix)
