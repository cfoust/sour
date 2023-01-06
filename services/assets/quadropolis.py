import sys
import shutil
import glob
from os import path
import os
from typing import NamedTuple, Optional, Tuple, List, Any
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
                roots=roots,
                map_path=_map,
            )
        )

    for root in roots:
        mods.append(
            ModJob(
                root=root,
            )
        )

    # Sometimes nodes did not have a root (e.g they were just a few cfgs)
    if not maps and not mods:
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
    args = sys.argv

    node_targets = []
    if len(args) > 1:
        node_targets = list(map(lambda a: int(a), args[1:]))

    outdir = os.getenv("ASSET_OUTPUT_DIR", "output/quad")
    os.makedirs(outdir, exist_ok=True)

    cachedir = path.join("/tmp", "quad-cache")
    os.makedirs(cachedir, exist_ok=True)

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

    nodes = list(filter(lambda node: node['id'] in node_targets if node_targets else True, nodes))

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
            maps, mods = get_jobs(
                File(
                    url=file['url'],
                    hash=file['hash'],
                    name=file['name'],
                    contents=file['contents'],
                )
            )

            # Node 4405 included a map _and_ a mod, and we still want both.
            if maps and mods and _id != 4405:
                mods = []

            for job in maps:
                file_hash = file.file_hash
                map_path = job.map_path
                map_hash = package.hash_string("%d-%s" % (_id, job.map_path))
                cache_file = path.join(cachedir, "%s.json" % map_hash)

                if path.exists(cache_file):
                    game_map = package.GameMap(**json.loads(open(cache_file, 'r').read()))
                    continue

                # The file itself is a map
                if not map_path:
                    map_name, _ = path.splitext(path.basename(file.file_name))
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

                tmpdir = tempfile.mkdtemp()

                file_name = file.file_name

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

                map_roots = list(map(lambda v: path.join(tmpdir, v), job.roots)) + roots
                target_map = path.join(tmpdir, map_path)

                if not path.exists(target_map):
                    print('Archive %s did not contain %s' % (file_name, map_path))
                    continue

                print(map_roots, roots[1])
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

    p.dump_index(prefix)
