#!/usr/bin/env python3
"""Cuts a risq minimap tile from a full hex tile and builds its hills variant.

The flat tile is the center crop of the full tile at native size, hex-masked with
the alpha every existing minimap tile shares. The hills tile
is the hill overlay run on that crop, using only the central --relief-keep
fraction of the relief image (1.0 uses all of it), leveled so its mean brightness
inside the hexagon matches the flat tile.

--detail-strength keeps some of the flat tile's own fine detail (farmland uses 0.4).

Usage: make_minimap_tiles.py <full_tile> <flat_out> <hills_out> [--relief-keep FLOAT] [--detail-strength FLOAT]
"""

import argparse
import tempfile
from pathlib import Path

import numpy as np
from PIL import Image

from apply_hill_overlay import DEFAULT_LEVEL, DEFAULT_RELIEF, apply_hill_overlay

MINIMAP_WIDTH = 104
MINIMAP_HEIGHT = 120
DEFAULT_RELIEF_KEEP = 0.8
MINIMAP_MASK = 'backend/static/images/risq/terrains/minimap/grass.png'


def center_crop(image: Image.Image, width: int, height: int) -> Image.Image:
    left, top = (image.width - width) // 2, (image.height - height) // 2
    return image.crop((left, top, left + width, top + height))


def opaque_mean_value(path: str) -> float:
    image = Image.open(path).convert('RGBA')
    value = np.asarray(image.convert('RGB').convert('HSV'), dtype=float)[..., 2]
    return value[np.asarray(image.split()[-1]) > 0].mean()


def make_minimap_tiles(full_tile: str, flat_out: str, hills_out: str, relief_keep: float, detail_strength: float) -> None:
    flat = center_crop(Image.open(full_tile).convert('RGBA'), MINIMAP_WIDTH, MINIMAP_HEIGHT)
    output = Image.new('RGBA', flat.size)
    output.paste(flat, (0, 0))
    output.putalpha(Image.open(MINIMAP_MASK).convert('RGBA').split()[-1])
    output.save(flat_out, format='PNG')
    relief = Image.open(DEFAULT_RELIEF).convert('L')
    relief = center_crop(relief, round(relief.width * relief_keep), round(relief.height * relief_keep))
    with tempfile.TemporaryDirectory() as tmp:
        relief_path = str(Path(tmp) / 'relief.png')
        relief.save(relief_path)
        apply_hill_overlay(flat_out, hills_out, relief_path, DEFAULT_LEVEL, detail_strength)
        level = DEFAULT_LEVEL * opaque_mean_value(flat_out) / opaque_mean_value(hills_out)
        apply_hill_overlay(flat_out, hills_out, relief_path, level, detail_strength)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument('full_tile')
    parser.add_argument('flat_out')
    parser.add_argument('hills_out')
    parser.add_argument('--relief-keep', type=float, default=DEFAULT_RELIEF_KEEP)
    parser.add_argument('--detail-strength', type=float, default=0.0)
    args = parser.parse_args()
    make_minimap_tiles(args.full_tile, args.flat_out, args.hills_out, args.relief_keep, args.detail_strength)
