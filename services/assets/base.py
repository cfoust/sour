import argparse
from pathlib import Path
import package
import sys
import glob
from os import path
import os
from typing import NamedTuple, Optional, Tuple, List, Dict, Set
import subprocess

from pip._vendor.rich import progress

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Generate assets from the base game.')

    parser.add_argument('--textures', action="store_true", help="Include all textures from the base game.")
    parser.add_argument('--models', action="store_true", help="Include all models from the base game.")
    parser.add_argument('--player-models', action="store_true", help="Include only player models from the base game.")
    parser.add_argument('maps', nargs=argparse.REMAINDER)

    args = parser.parse_args()
    print(args)

    maps = glob.glob('roots/base/packages/base/*.ogz')
    if args.maps:
        maps = list(map(lambda a: "roots/base/packages/base/%s.ogz" % a, args.maps))

    outdir = os.getenv("ASSET_OUTPUT_DIR", "output/")
    prefix = os.getenv("PREFIX", "")
    os.makedirs(outdir, exist_ok=True)

    roots = [
        "sour",
        "roots/base",
    ]

    roots = list(map(lambda root: path.abspath(root), roots))

    skip_root = roots[1]

    p = package.Packager(outdir)

    search = Path('roots/base/packages')

    if args.textures:
        textures: List[str] = []
        TEXTURE_TYPES = [
            "jpg",
            "png",
        ]
        for type_ in TEXTURE_TYPES:
            textures += list(map(lambda a: str(a), search.rglob(f"*.{type_}")))

        for texture in progress.track(textures, description="building textures"):
            p.build_texture(
                roots,
                texture,
            )

    if args.models:
        MODEL_TYPES = [
            "md2",
            "md3",
            "md5",
            "obj",
            "smd",
            "iqm"
        ]

        paths = [
            "roots/base/packages/models"
        ]

        if args.player_models:
            paths = [
                "roots/base/packages/models/mrfixit",
                "roots/base/packages/models/mrfixit_blue",
                "roots/base/packages/models/mrfixit_red",
                "roots/base/packages/models/snoutx10k",
                "roots/base/packages/models/snoutx10k_blue",
                "roots/base/packages/models/snoutx10k_red",
                "roots/base/packages/models/ogro2",
                "roots/base/packages/models/ogro2_blue",
                "roots/base/packages/models/ogro2_red",
                "roots/base/packages/models/inky",
                "roots/base/packages/models/inky_blue",
                "roots/base/packages/models/inky_red",
                "roots/base/packages/models/captaincannon",
                "roots/base/packages/models/captaincannon_blue",
                "roots/base/packages/models/captaincannon_red",
            ]

        models: List[str] = []
        for search_path in paths:
            for type_ in MODEL_TYPES:
                models += list(map(lambda a: str(a), Path(search_path).rglob(f"{type_}.cfg")))

        models = list(map(lambda model: path.abspath(model), models))

        for model in progress.track(models, description="building models"):
            result = p.build_model(
                roots,
                skip_root,
                model,
            )
            if not result:
                raise Exception('could not generate model')

    print("building base mod")
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

    for _map in progress.track(maps, description="building maps"):
        base, _ = path.splitext(path.basename(_map))
        p.build_map(
            roots,
            skip_root,
            _map,
            base,
            """Base game map %s as it appeared in game version r6584.
            """ % base,
        )

    p.dump_index(prefix)
