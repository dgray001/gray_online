#!/usr/bin/env python3
"""Convert a rectangular map image to a Risq grass/deep-water map; requires Pillow and NumPy."""

import argparse
import json
import math
from collections import Counter
from pathlib import Path

import numpy as np
from numpy.typing import NDArray
from PIL import Image

RGB = tuple[int, int, int]
HEX_WIDTH = 1.732


def parse_rgb(value: str) -> RGB:
    try:
        channels = bytes.fromhex(value.removeprefix('#'))
    except ValueError as error:
        raise argparse.ArgumentTypeError('water color must be six hex digits') from error
    if len(channels) != 3:
        raise argparse.ArgumentTypeError('water color must be six hex digits')
    return channels[0], channels[1], channels[2]


def land_mask(image: Image.Image, water_colors: list[RGB], tolerance: int) -> NDArray[np.bool_]:
    pixels = np.asarray(image.convert('RGBA'), dtype=np.int16)
    opaque = pixels[:, :, 3] >= 128
    if not water_colors and np.any(~opaque):
        return opaque
    if not water_colors:
        border = np.concatenate((pixels[0], pixels[-1], pixels[:, 0], pixels[:, -1]))
        counts = Counter((int(p[0]), int(p[1]), int(p[2])) for p in border)
        water_colors = [counts.most_common(1)[0][0]]
    water = np.zeros(opaque.shape, dtype=np.bool_)
    for color in water_colors:
        water |= np.max(np.abs(pixels[:, :, :3] - color), axis=2) <= tolerance
    return opaque & ~water


def hex_terrain(mask: NDArray[np.bool_], cx: float, cy: float, radius: float, threshold: float) -> int:
    left = max(0, math.floor(cx - HEX_WIDTH * radius / 2))
    right = min(mask.shape[1], math.ceil(cx + HEX_WIDTH * radius / 2))
    top = max(0, math.floor(cy - radius))
    bottom = min(mask.shape[0], math.ceil(cy + radius))
    ys, xs = np.mgrid[top:bottom, left:right]
    dx, dy = np.abs(xs + 0.5 - cx), np.abs(ys + 0.5 - cy)
    footprint = (dx <= HEX_WIDTH * radius / 2) & (dx + HEX_WIDTH * dy <= HEX_WIDTH * radius)
    count = np.count_nonzero(footprint)
    if not count:
        return 1 if mask[min(int(cy), mask.shape[0] - 1), min(int(cx), mask.shape[1] - 1)] else 301
    coverage = np.count_nonzero(mask[top:bottom, left:right] & footprint) / count
    return 1 if coverage >= threshold else 301


def map_document(mask: NDArray[np.bool_], columns: int, name: str, threshold: float) -> dict[str, object]:
    radius = mask.shape[1] / (columns * HEX_WIDTH)
    rows = math.ceil(mask.shape[0] / (1.5 * radius))
    row_min, col_min = -2 * (rows // 4), -(columns // 2)
    spaces: list[dict[str, int]] = []
    for row in range(rows):
        y = row + row_min
        for col in range(columns):
            x = col + col_min - y // 2
            cx, cy = (col + (y % 2) / 2) * HEX_WIDTH * radius, row * 1.5 * radius
            spaces.append({'x': x, 'y': y, 'terrain': hex_terrain(mask, cx, cy, radius, threshold)})
    size = max(max(abs(s['x']), abs(s['y']), abs(s['x'] + s['y'])) for s in spaces)
    top_left = [col_min - row_min // 2, row_min]
    return {
        'board_size': size,
        'players': 1,
        'spaces': spaces,
        'regions': [],
        'background_image': {
            'name': name,
            'top_left': top_left,
            'top_right': [top_left[0] + columns, row_min],
        },
    }


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('input_image', type=Path)
    parser.add_argument('output_map', type=Path)
    parser.add_argument('--columns', type=int, default=50, help='hex columns (default: 50)')
    parser.add_argument('--water-color', type=parse_rgb, action='append', default=[], metavar='RRGGBB',
                        help='sea color; repeat for multiple colors; defaults to transparency or border color')
    parser.add_argument('--tolerance', type=int, default=30, help='maximum per-channel sea-color difference')
    parser.add_argument('--land-threshold', type=float, default=0.5, help='land fraction needed per hex')
    parser.add_argument('--background-name', help='background asset name; defaults to input filename stem')
    args = parser.parse_args()
    if args.columns < 1 or not 0 < args.land_threshold <= 1 or not 0 <= args.tolerance <= 255:
        parser.error('columns must be positive, land-threshold in (0, 1], and tolerance in [0, 255]')
    if args.input_image.resolve() == args.output_map.resolve():
        parser.error('input image and output map must be different files')
    with Image.open(args.input_image) as image:
        mask = land_mask(image, args.water_color, args.tolerance)
    doc = map_document(mask, args.columns, args.background_name or args.input_image.stem, args.land_threshold)
    args.output_map.write_text(json.dumps(doc, indent=2) + '\n')
    radius = mask.shape[1] / (args.columns * HEX_WIDTH)
    print(f'{args.columns} x {math.ceil(mask.shape[0] / (1.5 * radius))} hexes -> {args.output_map}')


if __name__ == '__main__':
    main()
