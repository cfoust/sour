"""
Generate a catalog.json from Quadropolis nodes.json.

Usage:
    python3 quad_catalog.py --nodes ~/Developer/quadropolis.github.io/nodes.json \
                            --db ~/Developer/quadropolis.github.io/db \
                            --node-map output/quad/node_map.json \
                            --outdir catalogs/quadropolis
"""

import argparse
import json
import os
import shutil
from os import path
from typing import Any, Dict, List, Optional

from catalog import parse_author_date, parse_description, write_catalog


def find_image_hash(node: Any) -> Optional[str]:
    """Find the first image file's hash in a node's files."""
    for f in node.get('files', []):
        name = f.get('name', '')
        if name and (name.endswith('.png') or name.endswith('.jpg')):
            return f['hash'], name
    return None, None


def main():
    parser = argparse.ArgumentParser(description='Generate catalog from Quadropolis nodes.')
    parser.add_argument('--nodes', required=True, help='Path to nodes.json')
    parser.add_argument('--db', required=True, help='Path to quadropolis db/ directory')
    parser.add_argument('--node-map', required=True, help='Path to node_map.json (from quadropolis.py)')
    parser.add_argument('--outdir', default='catalogs/quadropolis', help='Output directory')
    args = parser.parse_args()

    nodes = json.load(open(args.nodes))
    node_map = json.load(open(args.node_map))

    # Build node lookup by ID
    nodes_by_id: Dict[int, Any] = {}
    for node in nodes:
        nodes_by_id[node['id']] = node

    os.makedirs(args.outdir, exist_ok=True)

    catalog_maps: Dict[str, Any] = {}

    for node_id_str, map_names in node_map.items():
        node_id = int(node_id_str)
        node = nodes_by_id.get(node_id)
        if not node:
            continue

        author, date = parse_author_date(node.get('author', ''))
        description = parse_description(node.get('content', ''))

        # Find and copy screenshot
        image = None
        img_hash, img_name = find_image_hash(node)
        if img_hash:
            src = path.join(args.db, img_hash)
            if path.exists(src):
                _, ext = path.splitext(img_name)
                dest_name = img_hash + ext
                dest = path.join(args.outdir, dest_name)
                if not path.exists(dest):
                    shutil.copy2(src, dest)
                image = dest_name

        for map_name in map_names:
            entry = {}
            if author:
                entry['author'] = author
            if date:
                entry['date'] = date
            if description:
                entry['description'] = description
            if image:
                entry['image'] = image

            catalog_maps[map_name] = entry

    catalog = {'maps': catalog_maps}
    out_path = write_catalog(args.outdir, catalog)
    print(f"Wrote {len(catalog_maps)} map entries to {out_path}")


if __name__ == '__main__':
    main()
