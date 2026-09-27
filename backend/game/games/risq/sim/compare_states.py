#!/usr/bin/env python3
"""Compares two game-state JSON files (e.g. per-turn dumps from a debug.log,
or two results.json entries) and reports whether they're identical.

Usage: compare_states.py <state1.json> <state2.json>

Numbers within RESOURCE_TOLERANCE of each other count as equal (floating-point
resource accumulation isn't expected to match to the last bit). Everything
else must match exactly.
"""
import json
import sys

RESOURCE_TOLERANCE = 0.0001


def diff(path, a, b, out):
    if isinstance(a, (int, float)) and isinstance(b, (int, float)) and not isinstance(a, bool) and not isinstance(b, bool):
        if abs(a - b) > RESOURCE_TOLERANCE:
            out.append(f"{path}: {a} != {b}")
        return
    if isinstance(a, dict) and isinstance(b, dict):
        for key in sorted(set(a) | set(b)):
            if key not in a:
                out.append(f"{path}.{key}: missing in first")
            elif key not in b:
                out.append(f"{path}.{key}: missing in second")
            else:
                diff(f"{path}.{key}", a[key], b[key], out)
        return
    if isinstance(a, list) and isinstance(b, list):
        if len(a) != len(b):
            out.append(f"{path}: length {len(a)} != {len(b)}")
            return
        for i, (av, bv) in enumerate(zip(a, b)):
            diff(f"{path}[{i}]", av, bv, out)
        return
    if a != b:
        out.append(f"{path}: {a!r} != {b!r}")


def main():
    if len(sys.argv) != 3:
        print(__doc__)
        sys.exit(1)
    a = json.load(open(sys.argv[1]))
    b = json.load(open(sys.argv[2]))
    differences = []
    diff("state", a, b, differences)
    if not differences:
        print("IDENTICAL")
        return
    print(f"DIFFERENT ({len(differences)} difference(s)):")
    for d in differences:
        print(" ", d)
    sys.exit(1)


if __name__ == "__main__":
    main()
