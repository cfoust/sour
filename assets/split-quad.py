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

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Generate an asset cache from Quadropolis data.')
    parser.add_argument('--outdir', help="The output directory for the asset source.", default="output/")
    parser.add_argument('--prefix', help="The prefix for the index file.", default="")
    args = parser.parse_args()

    os.makedirs(args.outdir, exist_ok=True)

    p = package.Packager(args.outdir)

    quaddir = 'quadropolis'

    nodes = json.loads(open(path.join(quaddir, 'nodes.json'), 'r').read())

    def db(file): return path.join(quaddir, "db", file)

    for node in track(nodes, "extracting nodes"):
        _id = node['id']
        files = node['files']

        node_path = f"{_id}/"

        for i, file in enumerate(files):
            file_name = file['name']
            file_hash = file['hash']
            contents = file['contents']

            file_path = path.join(node_path, str(i))

            if not file_name:
                continue

            if not contents:
                p.build_ref((db(file_hash), path.join(file_path, file_name)))
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

            for subfile in list(Path(tmpdir).rglob("*")):
                if not path.isfile(str(subfile)):
                    continue

                target = path.relpath(subfile, tmpdir)
                p.build_ref((str(subfile), path.join(file_path, target)))

            shutil.rmtree(tmpdir, ignore_errors=True)

    p.dump_index(args.prefix)
