"""
Shared utilities for generating catalog files.
"""

import hashlib
import json
import os
import shutil
from os import path
from typing import Dict, Optional, Tuple


def hash_file(filepath: str) -> str:
    """Return the SHA-256 hash of a file's contents."""
    sha = hashlib.sha256()
    with open(filepath, 'rb') as f:
        for chunk in iter(lambda: f.read(8192), b''):
            sha.update(chunk)
    return sha.hexdigest()


def copy_image(src_path: str, outdir: str) -> str:
    """Copy an image to outdir using its content hash as filename. Returns the hash."""
    file_hash = hash_file(src_path)
    _, ext = path.splitext(src_path)
    dest = path.join(outdir, file_hash + ext)
    if not path.exists(dest):
        shutil.copy2(src_path, dest)
    return file_hash + ext


def parse_author_date(author_field: str) -> Tuple[str, str]:
    """Parse 'AuthorName | YYYY-MM-DD HH:MM' into (author, date)."""
    parts = author_field.split(' | ')
    if len(parts) == 2:
        return parts[0].strip(), parts[1].strip()
    return author_field.strip(), ''


def parse_description(content: str) -> str:
    """Extract description from node content, skipping the pipe-delimited metadata first line."""
    lines = content.strip().split('\n')
    if not lines:
        return ''

    # If first line looks like pipe-delimited metadata, skip it
    first = lines[0]
    if '|' in first:
        lines = lines[1:]

    return '\n'.join(lines).strip()


def write_catalog(outdir: str, catalog: Dict) -> str:
    """Write a catalog.json to outdir. Returns the path."""
    os.makedirs(outdir, exist_ok=True)
    catalog_path = path.join(outdir, 'catalog.json')
    with open(catalog_path, 'w') as f:
        json.dump(catalog, f, indent=2)
    return catalog_path
