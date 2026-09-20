#!/usr/bin/env python3
"""Crops a texture to the largest pointy-top hexagon that fits inside it, then
resizes it to a fixed pixel size so every output image is identically sized.

Pointy-top means a vertex at top/bottom and flat vertical edges left/right,
matching the hexagon orientation risq draws on the canvas board. Output is a
732x846 RGBA PNG (matching risq's drawHexagon sqrt(3):2 ratio) with alpha
transparency outside the hexagon and no source metadata carried over. Does
not modify the input file.

Usage:
  cut_hex_texture.py <input_image> <output_image>   cut a single file
  cut_hex_texture.py                                cut every png in
                                                      DEFAULT_SOURCE_DIR into
                                                      DEFAULT_OUTPUT_DIR
"""

import argparse
import math
from pathlib import Path

from PIL import Image, ImageDraw

SUPERSAMPLE = 4
OUTPUT_WIDTH = 732
OUTPUT_HEIGHT = 846

REPO_ROOT = Path(__file__).resolve().parent.parent
DEFAULT_SOURCE_DIR = REPO_ROOT.parent.parent / 'assets' / 'images' / 'textures'
DEFAULT_OUTPUT_DIR = REPO_ROOT / 'backend' / 'static' / 'images' / 'risq' / 'terrains'


def hex_vertices(cx: float, cy: float, r: float) -> list[tuple[float, float]]:
    return [
        (cx + r * math.cos(math.radians(deg)), cy + r * math.sin(math.radians(deg)))
        for deg in (30, 90, 150, 210, 270, 330)
    ]


def cut_hex_texture(input_path: str, output_path: str) -> None:
    source = Image.open(input_path).convert('RGBA')
    r = min(source.height / 2, source.width / math.sqrt(3))
    bbox_w, bbox_h = math.sqrt(3) * r, 2 * r
    crop_x = (source.width - bbox_w) / 2
    crop_y = (source.height - bbox_h) / 2
    cropped = source.crop((round(crop_x), round(crop_y), round(crop_x + bbox_w), round(crop_y + bbox_h)))
    resized = cropped.resize((OUTPUT_WIDTH, OUTPUT_HEIGHT), Image.LANCZOS)

    mask_size = (OUTPUT_WIDTH * SUPERSAMPLE, OUTPUT_HEIGHT * SUPERSAMPLE)
    mask = Image.new('L', mask_size, 0)
    vertices = hex_vertices(mask_size[0] / 2, mask_size[1] / 2, mask_size[1] / 2)
    ImageDraw.Draw(mask).polygon(vertices, fill=255)
    mask = mask.resize((OUTPUT_WIDTH, OUTPUT_HEIGHT), Image.LANCZOS)

    resized.putalpha(mask)
    output = Image.new('RGBA', (OUTPUT_WIDTH, OUTPUT_HEIGHT))
    output.paste(resized, (0, 0))
    output.save(output_path, format='PNG')


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument('input_image', nargs='?', help='single input file; omit to use --names batch mode')
    parser.add_argument('output_image', nargs='?', help='single output file; omit to use --names batch mode')
    parser.add_argument('--names', nargs='+', help='explicit filenames (relative to --source-dir) to batch-cut into --output-dir')
    parser.add_argument('--source-dir', default=str(DEFAULT_SOURCE_DIR), help='batch mode source directory')
    parser.add_argument('--output-dir', default=str(DEFAULT_OUTPUT_DIR), help='batch mode output directory')
    args = parser.parse_args()
    if args.names:
        source_dir = Path(args.source_dir)
        output_dir = Path(args.output_dir)
        output_dir.mkdir(parents=True, exist_ok=True)
        for name in args.names:
            cut_hex_texture(str(source_dir / name), str(output_dir / name))
            print(f'cut {name}')
    elif args.input_image and args.output_image:
        cut_hex_texture(args.input_image, args.output_image)
    else:
        parser.error('provide input_image and output_image, or --names for batch mode')
