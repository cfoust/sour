import package
import shutil
import glob
from os import path
import os
from typing import NamedTuple, Optional, Tuple, List, Any
import json
import tempfile
import subprocess


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


if __name__ == "__main__":
    outdir = os.getenv("ASSET_OUTPUT_DIR", "output/quad")
    prefix = os.getenv("PREFIX")
    quaddir = 'quadropolis'

    roots = [
        "sour",
        "roots/base",
    ]

    os.makedirs(outdir, exist_ok=True)

    mods: List[package.Mod] = []
    game_maps: List[package.GameMap] = []

    nodes = json.loads(open(path.join(quaddir, 'nodes.json'), 'r').read())

    def tmp(file): return path.join("/tmp", file)
    def out(file): return path.join(outdir, file)
    def db(file): return path.join(quaddir, "db", file)

    jobs: List[BuildJob] = []
    for node in nodes:
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

                # The file itself is a map
                if not map_path:
                    print("%d: %s" % (_id, job.file_name))
                    target = tmp("%s.ogz" % file_hash)
                    shutil.copy(db(file_hash), target)
                    map_bundle = package.build_map_bundle(
                        target,
                        roots,
                        outdir
                    )

                    map_image = map_bundle.image if map_bundle.image else image

                    game_maps.append(
                        base_map._replace(
                            name=job.file_name,
                            bundle=map_bundle.bundle,
                            image=map_image,
                        )
                    )
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
                            target,
                            "-d",
                            tmpdir
                        ],
                        stderr=subprocess.DEVNULL,
                        stdout=subprocess.DEVNULL,
                        check=True
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

                map_bundle = package.build_map_bundle(
                    path.join(tmpdir, map_path),
                    map_roots,
                    outdir
                )

                map_image = map_bundle.image if map_bundle.image else image

                game_maps.append(
                    base_map._replace(
                        name=path.basename(map_path),
                        bundle=map_bundle.bundle,
                        image=map_image,
                    )
                )
                shutil.rmtree(tmpdir, ignore_errors=True)

    package.dump_index(game_maps, mods, outdir, prefix)
