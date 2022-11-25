import package
import sys
import glob
from os import path
import os
from typing import NamedTuple, Optional, Tuple, List

if __name__ == "__main__":
    args = sys.argv

    maps = glob.glob('roots/base/packages/base/*.ogz')

    if len(args) > 1:
        maps = list(map(lambda a: "roots/base/packages/base/%s.ogz" % a, args[1:]))

    outdir = os.getenv("ASSET_OUTPUT_DIR", "output/")
    prefix = os.getenv("PREFIX")
    os.makedirs(outdir, exist_ok=True)

    roots = [
        "sour",
        "roots/base",
    ]

    mods: List[package.Mod] = []

    # Build base
    with open("base.list", "r") as f:
        files = f.read().split("\n")

        mappings: List[package.Mapping] = []
        for file in files:
            mapping = package.search_file(file, roots)
            if not mapping or path.isdir(mapping[0]): continue
            mappings.append(mapping)

        _hash = package.build_bundle(mappings, outdir, compress_images=False)
        mods.append(
            package.Mod(
                name="base",
                bundle=_hash
            )
        )

    game_maps: List[package.GameMap] = []

    for _map in maps:
        base, _ = path.splitext(path.basename(_map))
        print("Building %s" % base)
        map_bundle = package.build_map_bundle(_map, roots, outdir)
        game_maps.append(
            package.GameMap(
                name=base,
                bundle=map_bundle.bundle,
                image=map_bundle.image,
                description="""
Base game map %s as it appeared in game version r6584.
""" % base,
                aliases=[]
            )
        )

    package.dump_index(game_maps, mods, outdir, prefix)
