import argparse
from pathlib import Path
import package
import sys
import glob
from os import path
import os
from typing import NamedTuple, Optional, Tuple, List, Dict, Set, Union, Sequence, Iterable, TypeVar
import subprocess
from multiprocessing import Pool, cpu_count

ProgressType = TypeVar("ProgressType")

def _track(
    sequence: Union[Sequence[ProgressType], Iterable[ProgressType]],
    description: str = "Working...",
    total: int = 0,
) -> Iterable[ProgressType]:
    return sequence

track = _track

try:
    from pip._vendor.rich import progress
    track = progress.track
except ModuleNotFoundError:
    pass


class BuildResult(NamedTuple):
    assets: Set[str]
    bundles: List[package.Bundle]
    maps: List[package.GameMap]


def build_map(
    params: package.BuildParams,
    outdir: str,
    _map: str,
) -> Optional[BuildResult]:
    p = package.Packager(outdir)
    base, _ = path.splitext(path.basename(_map))
    p.build_map(
        params,
        _map,
        base,
        """Base game map %s as it appeared in game version r6481.
        """ % base,
    )

    return BuildResult(
        assets=p.assets,
        bundles=p.bundles,
        maps=p.maps,
    )


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Generate assets from the base game.')
    parser.add_argument('--textures', action="store_true", help="Include all textures from the base game.")
    parser.add_argument('--models', action="store_true", help="Include all models from the base game.")
    parser.add_argument('--download', action="store_true", help="Whether to download assets from remote sources.")
    parser.add_argument('--prefix', help="The prefix for the index file.", default="")
    parser.add_argument('--root', help="The base root for accessing game files.", default="")
    parser.add_argument('--player-models', action="store_true", help="Include only player models from the base game.")
    parser.add_argument('maps', nargs=argparse.REMAINDER)
    args = parser.parse_args()

    if not sys.argv[1:]:
        args.textures = True
        args.models = True

    if not args.root:
        print("You must provide the base root.")
        exit(1)

    roots = [
        "sour",
        args.root,
    ]

    files = package.get_root_files(roots)

    maps = list(filter(lambda a: a.endswith('.ogz'), files))
    if args.maps:
        maps = list(map(lambda a: "packages/base/%s.ogz" % a, args.maps))

    maps.append("packages/base/xmwhub.ogz")

    outdir = os.getenv("ASSET_OUTPUT_DIR", "output/")
    os.makedirs(outdir, exist_ok=True)

    skip_root = roots[1]

    p = package.Packager(outdir)

    params = package.BuildParams(
        roots=roots,
        skip_root=roots[1],
        compress_images=False,
        download_assets=args.download,
        build_web=False,
        build_desktop=False,
    )

    if args.textures:
        textures: List[str] = []
        TEXTURE_TYPES = [
            "jpg",
            "png",
        ]
        for type_ in TEXTURE_TYPES:
            textures += list(filter(lambda a: a.endswith(f".{type_}"), files))

        p.build_textures(
            params,
            textures,
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
            "packages/models"
        ]

        if args.player_models:
            paths = [
                "packages/models/mrfixit",
                "packages/models/mrfixit_blue",
                "packages/models/mrfixit_red",
                "packages/models/snoutx10k",
                "packages/models/snoutx10k_blue",
                "packages/models/snoutx10k_red",
                "packages/models/ogro2",
                "packages/models/ogro2_blue",
                "packages/models/ogro2_red",
                "packages/models/inky",
                "packages/models/inky_blue",
                "packages/models/inky_red",
                "packages/models/captaincannon",
                "packages/models/captaincannon_blue",
                "packages/models/captaincannon_red",
            ]

        models: List[str] = []
        for search_path in paths:
            for type_ in MODEL_TYPES:
                models += list(filter(lambda a: a.startswith(search_path) and a.endswith(f"{type_}.cfg"), files))
                models += list(filter(lambda a: a.startswith(search_path) and a.endswith(f"tris.{type_}"), files))

        ids: List[str] = []
        for model in models:
            ids.append(path.dirname(model[len("packages/models/"):]))

        ids = list(set(ids))
        # These should always be included
        ids += [
            "ammo/bullets",
            "ammo/cartridges",
            "ammo/grenades",
            "ammo/rockets",
            "ammo/rrounds",
            "ammo/shells",
            "armor/green",
            "armor/yellow",
            "boost",
            "carrot",
            "checkpoint",
            "health",
            "quad",
            "teleporter",
            "flags/neutral",
            "flags/red",
            "flags/blue",
            "base/red",
            "base/neutral",
            "base/blue",
            "skull/red",
            "skull/blue",
        ]

        for model in track(ids, description="building models"):
            model_result = p.build_model(
                params,
                model,
            )
            if not model_result:
                raise Exception('could not generate model')

    print("building base mod")
    # Build base
    with open("base.list", "r") as f:
        files = f.read().split("\n")

        mappings: List[package.Mapping] = package.query_files(
            params.roots,
            files,
        )

        p.build_mod(
            params._replace(
                download_assets=True,
                build_web=True,
            ),
            mappings,
            "base",
            "Everything the base game needs.",
        )

    def _build_map(_map: str) -> Optional[BuildResult]:
        return build_map(params, outdir, _map)

    with Pool(cpu_count()) as pool:
        for result in track(pool.imap_unordered(
            _build_map,
            maps,
        ), "building maps", total=len(maps)):
            if not result:
                continue
            p.assets = p.assets | result.assets
            p.maps += result.maps
            p.bundles += result.bundles

    p.dump_index(args.prefix)
