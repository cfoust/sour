from pathlib import Path
import argparse
import sys
import shutil
import glob
from os import path
import os
from typing import NamedTuple, Optional, Tuple, List, Any, Set
import json
import tempfile
import subprocess
import hashlib
from multiprocessing import Pool, cpu_count

import package

from pip._vendor.rich import progress
track = progress.track

class File(NamedTuple):
    url: str
    hash: str
    name: Optional[str]
    contents: Optional[List[str]]


class MapJob(NamedTuple):
    roots: List[str]
    # A map_path set to None implies that the file itself is a map; otherwise
    # it is a path inside of the archive
    map_path:  Optional[str]


class ModJob(NamedTuple):
    root: str

BANNED_SUFFIXES = [
    "native_server",
    "sauerbraten_unix",
    ".exe",
    ".bat",
    ".py",
    # for now we can't handle this, since the client can't load them (we do not
    # know how to calculate the textures they use)
    ".cgz",
]

def is_valid_mod(files: List[str]) -> bool:
    for file in files:
        for suffix in BANNED_SUFFIXES:
            if file.endswith(suffix):
                return False

    return True

def get_jobs(file: File) -> Tuple[List[MapJob], List[ModJob]]:
    maps: List[MapJob] = []
    mods: List[ModJob] = []

    contents: List[str] = file.contents or []

    if not contents and file.name:
        name = file.name

        # Some maps did not come in archive files
        if name.endswith('.ogz'):
            maps.append(
                MapJob(
                    map_path=None,
                    roots=[],
                )
            )

        if name.endswith('.cfg'):
            mods.append(
                ModJob(
                    root='',
                )
            )

        return (maps, mods)

    contents = list(filter(lambda a: not a.startswith('__MACOSX'), contents))

    if not contents or not file.name:
        return (maps, mods)

    roots: List[str] = []

    for entry in contents:
        parts = entry.split(path.sep)

        for i, part in enumerate(parts):
            if part != "data" and part != "packages":
                continue

            root_slice = parts[:i]
            root = ""
            if root_slice:
                root = path.join(*root_slice)

            roots.append(root)

    roots = list(set(roots))
    map_files = list(filter(lambda a: a.endswith('.ogz'), contents))

    _, extension = path.splitext(file.name)

    for _map in map_files:
        maps.append(
            MapJob(
                roots=roots if roots else [''],
                map_path=_map,
            )
        )

    for root in roots:
        root_files = list(filter(lambda v: v.startswith(root), contents))
        if not is_valid_mod(root_files):
            continue

        mods.append(
            ModJob(
                root=root,
            )
        )

    # Sometimes nodes did not have a root (e.g they were just a few cfgs)
    if not maps and not mods:
        if not is_valid_mod(contents):
            return (maps, mods)

        mods.append(
            ModJob(
                root='',
            )
        )

    return (maps, mods)


def build_map(
    p: package.Packager,
    params: package.BuildParams,
    map_file: str,
    name: str,
    description: str,
    image: str = None,
) -> Optional[package.GameMap]:

    try:
        map_bundle = p.build_map(
            params,
            map_file,
            name,
            description,
            image,
        )
    except Exception as e:
        if 'shims' in str(e):
            return None
        elif 'invalid header' in str(e):
            print('Map had invalid gzip header')
            return None
        elif 'invalid octsav' in str(e):
            print('Map had invalid octsav')
            return None
        else:
            raise e

    return map_bundle


class BuildResult(NamedTuple):
    assets: Set[str]
    bundles: List[package.Bundle]
    maps: List[package.GameMap]
    mods: List[package.Mod]
    failed_maps: List[str]


def build_node(
    params: package.BuildParams,
    outdir: str,
    node: Any,
) -> Optional[BuildResult]:
    p = package.Packager(outdir)
    _id = node['id']
    files = node['files']
    failed_maps: List[str] = []

    node_prefix = str(_id)

    image = None
    for i, file in enumerate(files):
        name: str = file['name']
        if (
            not name or
            (not name.endswith('.jpg') and not name.endswith('.png'))
        ): continue
        image_path = path.join(
            node_prefix,
            str(i),
            name,
        )
        image = p.build_image(
            params,
            image_path,
        )

    description = node['content']

    for i, file in enumerate(files):
        file_name = file['name']
        file_hash = file['hash']

        file_dir = path.join(
            node_prefix,
            str(i),
        )
        file_root = f"{quad_root}@{file_dir}"

        contents = package.get_root_files(
            [file_root]
        )

        file_params = params._replace(
            roots=roots + [file_root],
        )

        file_params_no_skip = file_params._replace(
            skip_root=""
        )

        maps, mods = get_jobs(
            File(
                url=file['url'],
                hash=file_hash,
                name=file_name,
                contents=contents,
            )
        )

        # Node 4405 included a map _and_ a mod, and we still want both.
        if maps and mods and _id != 4405:
            mods = []

        # We don't need to do any extraction
        if not file['contents']:
            if mods:
                mod = mods[0]
                mod_file = path.basename(file_name)
                resolved = package.query_files(
                    file_params.roots,
                    [mod_file]
                )
                mapping = resolved[0]
                if mapping[0] == "nil":
                    continue
                p.build_mod(
                    file_params_no_skip,
                    resolved,
                    f"quad-{_id}",
                    description,
                    image=image,
                )
                continue

            if not maps:
                continue

            # The file itself is a map
            map_name, _ = path.splitext(path.basename(file_name))

            try:
                build_map(
                    p,
                    file_params,
                    path.basename(file_name),
                    map_name,
                    description,
                    image,
                )
            except Exception as e:
                failed_maps.append(file_dir + map_name)
                print(f"failed to build map id={_id} map={file_name} err={str(e)}")
                break

            continue

        for i, mod in enumerate(mods):
            mod_files = list(filter(lambda a: a.startswith(mod.root), contents))

            if not mod_files:
                continue

            resolved = package.query_files(
                file_params.roots,
                mod_files
            )

            name = f"quad-{_id}"
            if len(mods) > 1:
                name += f"-{i}"

            p.build_mod(
                file_params_no_skip,
                resolved,
                name,
                description,
                image=image,
            )

        for job in maps:
            map_path = job.map_path

            # The file itself is a map, we handled this above
            if not map_path:
                continue

            map_roots = list(map(lambda v: path.join(file_root, v), job.roots)) + roots
            name, _ = path.splitext(path.basename(map_path))
            try:
                build_map(
                    p,
                    file_params,
                    map_path,
                    name,
                    description,
                    image,
                )
            except Exception as e:
                failed_maps.append(file_dir + map_path)
                print(f"failed to build map id={_id} map={map_path} err={str(e)}")
                break

    return BuildResult(
        assets=p.assets,
        bundles=p.bundles,
        maps=p.maps,
        mods=p.mods,
        failed_maps=failed_maps,
    )


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Generate assets from Quadropolis.')
    parser.add_argument('--dry', action="store_true", help="Don't build anything, just print what would be built.")
    parser.add_argument('--prefix', help="The prefix for the index file.", default="")
    parser.add_argument('nodes', nargs=argparse.REMAINDER, help="Particular node IDs you want to build.")
    args = parser.parse_args()

    node_targets = args.nodes

    outdir = os.getenv("ASSET_OUTPUT_DIR", "output/quad")
    os.makedirs(outdir, exist_ok=True)

    p = package.Packager(outdir)

    quaddir = 'quadropolis'

    quad_root = "https://static.sourga.me/quadropolis/4412/.index.source"
    roots = [
        "sour",
        "https://static.sourga.me/blobs/6481/.index.source",
        quad_root,
    ]

    params = package.BuildParams(
        roots=roots,
        skip_root=roots[1],
        compress_images=False,
        download_assets=False,
        build_web=False,
        build_desktop=False,
    )

    nodes = json.loads(open('nodes.json', 'r').read())

    failures = open('failures.txt', 'w', buffering=1)

    nodes = reversed(nodes)

    node_targets = list(map(int, node_targets))
    nodes = list(filter(lambda node: node['id'] in node_targets if node_targets else True, nodes))

    def _build_node(node: Any) -> Optional[BuildResult]:
        return build_node(params, outdir, node)

    with Pool(cpu_count()) as pool:
        for result in track(pool.imap_unordered(
            _build_node,
            nodes,
        ), "building nodes", total=len(nodes)):
            if not result:
                continue
            p.assets = p.assets | result.assets
            p.mods += result.mods
            p.maps += result.maps
            p.bundles += result.bundles

            for map_ in result.failed_maps:
                failures.write(map_ + "\n")

    num_mods = len(p.mods)
    num_maps = len(p.maps)

    failures.close()
    print(f"built {num_mods} mods and {num_maps} maps")
    p.dump_index(args.prefix)
