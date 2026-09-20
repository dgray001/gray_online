#!/usr/bin/env python3
"""Turns a flat hex-cut terrain texture into its hilly variant.

Reverse-engineered from the existing risq terrain art: a hills texture keeps
the flat texture's hue and saturation, but replaces its value/luminance with
a shared rock-relief image's value (i.e. a "Luminosity" blend of the relief
photo over the flat texture, brightness-leveled to suit the flat texture).

With --detail-strength > 0, some of the flat texture's own fine luminance
detail (e.g. tilled-row furrows) is preserved by multiplying it back on top
of the leveled relief, so it isn't fully erased by the relief's own shading.

Usage: apply_hill_overlay.py <flat_texture> <output_image> [--relief PATH] [--level FLOAT] [--detail-strength FLOAT]
"""

import argparse

import numpy as np
from PIL import Image, ImageFilter

DEFAULT_RELIEF = 'backend/static/images/risq/terrains/hill_relief_base.png'
DEFAULT_LEVEL = 0.78  # hills-mean-V / flat-mean-V, matching the grass/dirt originals
DETAIL_BLUR_RADIUS = 10


def apply_hill_overlay(
    flat_path: str, output_path: str, relief_path: str, level: float, detail_strength: float
) -> None:
    flat = Image.open(flat_path).convert('RGBA')
    alpha = flat.split()[-1]
    flat_hsv = np.asarray(flat.convert('RGB').convert('HSV'), dtype=np.float64)
    h, s, flat_v = flat_hsv[..., 0], flat_hsv[..., 1], flat_hsv[..., 2]

    relief = Image.open(relief_path).convert('L')
    if relief.size != flat.size:
        relief = relief.resize(flat.size, Image.LANCZOS)
    relief_v = np.asarray(relief, dtype=np.float64)

    target_mean = level * flat_v.mean()
    leveled_relief_v = relief_v * (target_mean / relief_v.mean())

    detail = 1.0
    if detail_strength > 0:
        flat_v_blur = np.asarray(
            Image.fromarray(flat_v.astype(np.uint8)).filter(ImageFilter.GaussianBlur(radius=DETAIL_BLUR_RADIUS)),
            dtype=np.float64,
        )
        raw_detail = flat_v / np.maximum(flat_v_blur, 1)
        detail = 1 + (raw_detail - 1) * detail_strength

    new_v = np.clip(leveled_relief_v * detail, 0, 255)

    hills_hsv = np.stack([h, s, new_v], axis=-1).astype(np.uint8)
    hills_rgb = Image.fromarray(hills_hsv, 'HSV').convert('RGB')
    hills_rgb.putalpha(alpha)
    hills_rgb.save(output_path)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument('flat_texture')
    parser.add_argument('output_image')
    parser.add_argument('--relief', default=DEFAULT_RELIEF)
    parser.add_argument('--level', type=float, default=DEFAULT_LEVEL)
    parser.add_argument('--detail-strength', type=float, default=0.0)
    args = parser.parse_args()
    apply_hill_overlay(args.flat_texture, args.output_image, args.relief, args.level, args.detail_strength)
