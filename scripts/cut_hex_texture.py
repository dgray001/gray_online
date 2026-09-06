#!/usr/bin/env python3
"""Crops a texture to the largest pointy-top hexagon that fits inside it.

Pointy-top means a vertex at top/bottom and flat vertical edges left/right,
matching the hexagon orientation risq draws on the canvas board. Output is a
PNG with alpha transparency outside the hexagon, sized so that width:height
is exactly sqrt(3):2 (the same ratio risq's drawHexagon uses), so it can be
drawn with ctx.drawImage(img, x, y, sqrt(3) * hex_r, 2 * hex_r) with no
distortion. Does not modify the input file.

Usage: cut_hex_texture.py <input_image> <output_image>
"""

import argparse
import math

from PIL import Image, ImageDraw

SUPERSAMPLE = 4


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

    mask_size = (cropped.width * SUPERSAMPLE, cropped.height * SUPERSAMPLE)
    mask = Image.new('L', mask_size, 0)
    vertices = hex_vertices(mask_size[0] / 2, mask_size[1] / 2, mask_size[1] / 2)
    ImageDraw.Draw(mask).polygon(vertices, fill=255)
    mask = mask.resize(cropped.size, Image.LANCZOS)

    cropped.putalpha(mask)
    cropped.save(output_path)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument('input_image')
    parser.add_argument('output_image')
    args = parser.parse_args()
    cut_hex_texture(args.input_image, args.output_image)
