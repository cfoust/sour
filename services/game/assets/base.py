import package
import sys
import glob
from os import path
import os
from typing import NamedTuple, Optional, Tuple, List

if __name__ == "__main__":
    args = sys.argv

    maps = glob.glob('roots/base/packages/base/*.ogz')

    if len(args) > 1:
        maps = list(map(lambda a: "roots/base/packages/base/%s.ogz" % a, args[1:]))

    outdir = os.getenv("ASSET_OUTPUT_DIR", "output/")
    os.makedirs(outdir, exist_ok=True)

    roots = [
        "sour",
        "roots/base",
    ]

    hashes: List[Tuple[str, str]] = []

    # Build base
    with open("base.list", "r") as f:
        files = f.read().split("\n")

        mappings: List[package.Mapping] = []
        for file in files:
            mapping = package.search_file(file, roots)
            if not mapping or path.isdir(mapping[0]): continue
            mappings.append(mapping)

        _hash = package.build_bundle(mappings, outdir, compress_images=False)
        hashes.append(("base", _hash))

    for _map in maps:
        base, _ = path.splitext(path.basename(_map))
        print("Building %s" % base)
        _hash = package.build_map_bundle(_map, roots, outdir)
        hashes.append((base, _hash))

    with open(path.join(outdir, ".index"), "w") as f:
        for _map, _hash in hashes:
            f.write("%s %s\n" % (_map, _hash))
