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


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Generate an asset source for a directory.')
    parser.add_argument('--outdir', help="The output directory for the asset source.", default="output/")
    parser.add_argument('--prefix', help="The prefix for the index file.", default="")
    parser.add_argument('root')
    args = parser.parse_args()
    os.makedirs(args.outdir, exist_ok=True)

    p = package.Packager(args.outdir)

    for file in track(list(Path(args.root).rglob("*"))):
        if not path.isfile(str(file)):
            continue

        target = path.relpath(file, args.root)
        p.build_ref((str(file), target))

    p.dump_index(args.prefix)
