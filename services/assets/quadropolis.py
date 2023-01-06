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
    roots: List[str],
    skip_root: str,
    map_file: str,
    name: str,
    description: str,
    image: str = None,
    build_desktop: bool = False,
) -> Optional[package.GameMap]:

    try:
        map_bundle = p.build_map(
            roots,
            skip_root,
            map_file,
            name,
            description,
            image,
            build_desktop,
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


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Generate assets from Quadropolis.')
    parser.add_argument('--dry', action="store_true", help="Don't build anything, just print what would be built.")
    parser.add_argument('nodes', nargs=argparse.REMAINDER, help="Particular node IDs you want to build.")
    args = parser.parse_args()

    node_targets = args.nodes

    outdir = os.getenv("ASSET_OUTPUT_DIR", "output/quad")
    os.makedirs(outdir, exist_ok=True)

    p = package.Packager(outdir)

    prefix = os.getenv("PREFIX", '')
    quaddir = 'quadropolis'

    roots = [
        "sour",
        "roots/base",
    ]

    nodes = json.loads(open(path.join(quaddir, 'nodes.json'), 'r').read())

    nodes = reversed(nodes)

    def tmp(file): return path.join("/tmp", file)
    def out(file): return path.join(outdir, file)
    def db(file): return path.join(quaddir, "db", file)

    node_targets = list(map(int, node_targets))
    nodes = list(filter(lambda node: node['id'] in node_targets if node_targets else True, nodes))

    num_mods = 0
    num_maps = 0

    jobs: List[MapJob] = []
    for node in track(nodes, "building nodes"):
        _id = node['id']
        files = node['files']

        image = None
        for file in files:
            name = file['name']
            if (
                not name or
                (not name.endswith('.jpg') and not name.endswith('.png'))
            ): continue
            _, extension = path.splitext(file['name'])
            image = '%s%s' % (file['hash'], extension)
            shutil.copy(db(file['hash']), out(image))

        description = node['content']

        for file in files:
            file_name = file['name']
            file_hash = file['hash']

            maps, mods = get_jobs(
                File(
                    url=file['url'],
                    hash=file_hash,
                    name=file_name,
                    contents=file['contents'],
                )
            )

            # Node 4405 included a map _and_ a mod, and we still want both.
            if maps and mods and _id != 4405:
                mods = []

            # We don't need to do any extraction
            if not file['contents']:
                if args.dry:
                    continue

                if mods:
                    mod = mods[0]
                    mod_file = path.basename(file_name)
                    p.build_mod(
                        '', # Don't skip anything
                        [
                            (db(file_hash), mod_file),
                        ],
                        f"quad-{_id}",
                        description,
                        image=image,
                        compress_images=False,
                        build_web=False,
                        build_desktop=True,
                    )
                    num_mods += 1
                    continue

                if not maps:
                    continue

                # The file itself is a map
                map_name, _ = path.splitext(path.basename(file_name))
                target = tmp("%s.ogz" % map_name)
                shutil.copy(db(file_hash), target)

                build_map(
                    p,
                    roots,
                    roots[1],
                    target,
                    map_name,
                    description,
                    image,
                    build_desktop=True,
                )
                continue

            if args.dry:
                num_maps += len(maps)
                num_mods += len(mods)
                continue

            tmpdir = tempfile.mkdtemp()

            if file_name.endswith('.tar.gz'):
                target = path.join("/tmp", "%s.tar.gz" % file_hash)
                shutil.copy(db(file_hash), target)
                subprocess.run(
                    [
                        "tar",
                        "xf",
                        target,
                        "-C",
                        tmpdir
                    ],
                    stderr=subprocess.DEVNULL,
                    stdout=subprocess.DEVNULL,
                    check=True
                )
            elif file_name.endswith('.zip'):
                target = path.join("/tmp", "%s.zip" % file_hash)
                shutil.copy(db(file_hash), target)
                subprocess.run(
                    [
                        "unzip",
                        "-n",
                        target,
                        "-d",
                        tmpdir
                    ],
                    stderr=subprocess.DEVNULL,
                    stdout=subprocess.DEVNULL,
                )
            elif file_name.endswith('.rar'):
                target = path.join("/tmp", "%s.rar" % file_hash)
                shutil.copy(db(file_hash), target)
                subprocess.run(
                    [
                        "unrar",
                        "x",
                        target,
                    ],
                    cwd=tmpdir,
                    stderr=subprocess.DEVNULL,
                    stdout=subprocess.DEVNULL,
                    check=True
                )
            else:
                print('Unhandled archive: %s' % file_name)
                continue

            for i, mod in enumerate(mods):
                mod_files: List[package.Mapping] = []
                mod_dir = path.join(tmpdir, mod.root)
                for file in list(Path(mod_dir).rglob('*')):
                    relative = package.get_root_relative(file, [mod_dir])
                    if not relative or not path.isfile(str(file)):
                        continue
                    mod_files.append((str(file), relative))

                name = f"quad-{_id}"
                if len(mods) > 1:
                    name += f"-{i}"

                p.build_mod(
                    '', # Don't skip anything
                    mod_files,
                    name,
                    description,
                    image=image,
                    compress_images=False,
                    build_web=False,
                    build_desktop=True,
                )
                num_mods += 1

            for job in maps:
                map_path = job.map_path
                map_hash = package.hash_string("%d-%s" % (_id, job.map_path))

                num_maps += 1

                # The file itself is a map, we handled this above
                if not map_path:
                    continue

                if args.dry:
                    continue

                map_roots = list(map(lambda v: path.join(tmpdir, v), job.roots)) + roots
                target_map = path.join(tmpdir, map_path)

                if not path.exists(target_map):
                    print('Archive %s did not contain %s' % (file_name, map_path))
                    continue

                name, _ = path.splitext(path.basename(map_path))
                build_map(
                    p,
                    map_roots,
                    roots[1],
                    target_map,
                    name,
                    description,
                    image,
                    build_desktop=True,
                )

                shutil.rmtree(tmpdir, ignore_errors=True)

    print(f"built {num_mods} mods and {num_maps} maps")
    p.dump_index(prefix)
