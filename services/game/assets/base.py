import package
import sys
import glob
from os import path
import os

if __name__ == "__main__":
    args = sys.argv

    maps = glob.glob('roots/base/packages/base/*.ogz')

    if len(args) > 1:
        maps = list(map(lambda a: "roots/base/packages/base/%s.ogz" % a, args[1:]))

    outdir = "output/"
    os.makedirs(outdir, exist_ok=True)

    for _map in maps:
        print("Building %s" % _map)
        package.build_map_bundle(
            _map,
            [
                "sour",
                "roots/base",
            ],
            outdir
        )
