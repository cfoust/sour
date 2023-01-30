import argparse
from pathlib import Path
import package
import sys
import glob
from os import path
import os
from typing import NamedTuple, Optional, Tuple, List, Dict, Set, Union, Sequence, Iterable, TypeVar
import subprocess

ProgressType = TypeVar("ProgressType")

def _track(
    sequence: Union[Sequence[ProgressType], Iterable[ProgressType]],
    description: str = "Working...",
) -> Iterable[ProgressType]:
    return sequence

track = _track

try:
    from pip._vendor.rich import progress
    track = progress.track
except ModuleNotFoundError:
    pass

IMAGE_TYPES = [
    ".jpg",
    ".png",
    ".dds",
]

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Generate an asset source for a directory.')
    parser.add_argument('--outdir', help="The output directory for the asset source.", default="output/")
    parser.add_argument('--prefix', help="The prefix for the index file.", default="")
    parser.add_argument('--mobile', action="store_true", help="Only generate compresed textures.")
    parser.add_argument('root')
    args = parser.parse_args()
    os.makedirs(args.outdir, exist_ok=True)

    p = package.Packager(args.outdir)

    params = package.BuildParams(
        roots=[],
        skip_root="",
        compress_images=True,
        download_assets=False,
        build_web=False,
        build_desktop=False,
    )

    files: List[str] = list(map(lambda a: str(a), Path(args.root).rglob("*")))

    def should_include(name: str) -> bool:
        _, extension = path.splitext(name)
        return extension in IMAGE_TYPES and not 'data/' in name

    if args.mobile:
        files = list(filter(lambda a: should_include(str(a)), files))

    print(len(files))
    for file in track(files):
        if not path.isfile(file):
            continue

        target = path.relpath(file, args.root)
        p.build_ref(params, (f"fs:{file}", target))

    p.dump_index(args.prefix)
