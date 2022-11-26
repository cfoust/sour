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

class File(NamedTuple):
    url: str
    hash: str
    name: Optional[str]
    contents: Optional[List[str]]


class BuildJob(NamedTuple):
    file_hash: str
    file_name: str
    extension: Optional[str]
    roots: List[str]
    # A map_path set to None implies that the file itself is a map; otherwise
    # it is a path inside of the archive
    map_path:  Optional[str]


def get_jobs(file: File) -> List[BuildJob]:
    jobs: List[BuildJob] = []

    contents: List[str] = file.contents or []

    if not contents and file.name:
        name = file.name

        # Some maps did not come in archive files
        if name.endswith('.ogz'):
            jobs.append(
                BuildJob(
                    file_hash=file.hash,
                    file_name=file.name,
                    extension=None,
                    map_path=None,
                    roots=[],
                )
            )

        return jobs

    contents = list(filter(lambda a: not a.startswith('__MACOSX'), contents))

    if not contents or not file.name: return []

    roots: List[str] = []

    for entry in contents:
        parts = entry.split(path.sep)

        for i, part in enumerate(parts):
            if part != "data" and part != "packages": continue

            root_slice = parts[:i]
            root = ""
            if root_slice:
                root = path.join(*root_slice)

            roots.append(root)

    roots = list(set(roots))
    maps = list(filter(lambda a: a.endswith('.ogz'), contents))

    _, extension = path.splitext(file.name)

    for _map in maps:
        jobs.append(
            BuildJob(
                file_hash=file.hash,
                file_name=file.name,
                extension=extension,
                roots=roots,
                map_path=_map,
            )
        )

    return jobs


def build_map_bundle(
    map_file: str,
    roots: List[str],
    outdir: str,
) -> Optional[package.BuiltMap]:

    try:
        map_bundle = package.build_map_bundle(
            map_file,
            roots,
            outdir
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

    prefix = os.getenv("PREFIX")
    quaddir = 'quadropolis'

    roots = [
        "sour",
        "roots/base",
    ]

    mods: List[package.Mod] = []
    game_maps: List[package.GameMap] = []

    nodes = json.loads(open(path.join(quaddir, 'nodes.json'), 'r').read())

    def tmp(file): return path.join("/tmp", file)
    def out(file): return path.join(outdir, file)
    def db(file): return path.join(quaddir, "db", file)

    jobs: List[BuildJob] = []
    for node in nodes:
        _id = node['id']
        if node_targets and not _id in node_targets: continue
        files = node['files']

        image = None
        for file in files:
            name = file['name']
            if (
                not name or
                (not name.endswith('.jpg') and not name.endswith('.png'))
            ): continue
            _, extension = path.splitext(file['name'])
            image = '%d%s' % (_id, extension)
            shutil.copy(db(file['hash']), out(image))

        base_map = package.GameMap(
            name='',
            bundle='',
            image=image,
            description=node['content'],
            aliases=['quad_%d' % _id]
        )

        for file in files:
            for job in get_jobs(
                File(
                    url=file['url'],
                    hash=file['hash'],
                    name=file['name'],
                    contents=file['contents'],
                )
            ):
                file_hash = job.file_hash
                map_path = job.map_path
                map_hash = package.hash_string("%d-%s" % (_id, job.map_path))
                cache_file = path.join(cachedir, "%s.json" % map_hash)

                if path.exists(cache_file):
                    game_map = package.GameMap(**json.loads(open(cache_file, 'r').read()))
                    game_maps.append(game_map)
                    continue

                # The file itself is a map
                if not map_path:
                    print("%d: %s" % (_id, job.file_name))
                    target = tmp("%s.ogz" % file_hash)
                    shutil.copy(db(file_hash), target)

                    map_bundle = build_map_bundle(
                        target,
                        roots,
                        outdir,
                    )

                    if not map_bundle: continue

                    map_image = map_bundle.image if map_bundle.image else image

                    game_map = base_map._replace(
                        name=job.file_name,
                        bundle=map_bundle.bundle,
                        image=map_image,
                    )

                    with open(cache_file, 'w') as f:
                        f.write(json.dumps(game_map._asdict()))

                    game_maps.append(game_map)
                    continue

                print("%d: %s" % (_id, map_path))
                tmpdir = tempfile.mkdtemp()

                file_name = job.file_name

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
                    print('Archive %s did not contain %s' % (job.file_name, map_path))
                    continue

                map_bundle = build_map_bundle(
                    target_map,
                    map_roots,
                    outdir,
                )

                if not map_bundle: continue

                map_image = map_bundle.image if map_bundle.image else image

                game_map = base_map._replace(
                    name=path.basename(map_path),
                    bundle=map_bundle.bundle,
                    image=map_image,
                )

                with open(cache_file, 'w') as f:
                    f.write(json.dumps(game_map._asdict()))

                game_maps.append(game_map)
                shutil.rmtree(tmpdir, ignore_errors=True)

    package.dump_index(game_maps, mods, outdir, prefix)
