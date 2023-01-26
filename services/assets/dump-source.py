import argparse
from pathlib import Path
import package
import sys
import glob
import json
import cbor2
from os import path
import os
from typing import NamedTuple, Optional, Tuple, List, Dict, Set, Union, Sequence, Iterable, TypeVar
import subprocess


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Dump a source to JSON.')
    parser.add_argument('source')
    args = parser.parse_args()

    with open(args.source, 'rb') as f:
        with open(args.source + '.json', 'w') as g:
            g.write(json.dumps(cbor2.load(f), indent=4))
