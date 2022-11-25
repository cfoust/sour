import package
import glob
from os import path
import os
from typing import NamedTuple, Optional, Tuple, List
import json


class File(NamedTuple):
    url: str
    hash: str
    name: Optional[str]
    contents: Optional[List[str]]


class BuildJob(NamedTuple):
    file_hash: str
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
                    extension=None,
                    map_path=None,
                    roots=[]
                )
            )

        return jobs

    contents = list(filter(lambda a: not a.startswith('__MACOSX'), contents))

    for replacement in ['Packages', 'Data', 'Base']:
        contents = list(
            map(lambda a: a.replace(replacement, replacement.lower()), contents)
        )

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

    jobs: List[BuildJob] = []
    for node in nodes:
        files = node['files']

        for file in files:
            jobs += get_jobs(
                File(
                    url=file['url'],
                    hash=file['hash'],
                    name=file['name'],
                    contents=file['contents'],
                )
            )

    package.dump_index(game_maps, mods, outdir, prefix)
