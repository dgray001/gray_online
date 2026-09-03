import type { Point2D } from '../../util/objects2d';
import type { GameRisq, RisqSpace } from './risq_data';

/** Returns the space from the input index, if the space exists */
export function getSpace(game: GameRisq, index: Point2D): RisqSpace | undefined {
  if (index.x < 0 || index.x >= game.spaces.length) {
    return undefined;
  }
  const row = game.spaces[index.x];
  if (!row) {
    return undefined;
  }
  if (index.y < 0 || index.y >= row.length) {
    return undefined;
  }
  return row[index.y];
}

/** Transforms the input coordinate in axial space to index space */
export function coordinateToIndex(board_size: number, coordinate: Point2D): Point2D {
  return {
    x: coordinate.y + board_size,
    y: coordinate.x - Math.max(-board_size, -(board_size + coordinate.y)),
  };
}

/** Transforms the input coordinate in index space to axial space */
export function indexToCoordinate(board_size: number, index: Point2D): Point2D {
  const cy = index.x - board_size;
  return {
    x: index.y + Math.max(-board_size, -(board_size + cy)),
    y: cy,
  };
}

/** Cantor pairing function adapted to work with negatives, mirroring backend's util.Pair */
export function cantorPair(i: number, j: number): number {
  const ni = i < 0 ? -2 * i - 1 : 2 * i;
  const nj = j < 0 ? -2 * j - 1 : 2 * j;
  return ((ni + nj) * (ni + nj + 1)) / 2 + nj;
}

function natToInt(n: number): number {
  return n % 2 === 0 ? n / 2 : -(n + 1) / 2;
}

/** Inverts cantorPair, mirroring backend's util.InvertPair */
export function invertPair(z: number): Point2D {
  const w = Math.floor((Math.sqrt(8 * z + 1) - 1) / 2);
  const t = (w * w + w) / 2;
  const ny = z - t;
  const nx = w - ny;
  return { x: natToInt(nx), y: natToInt(ny) };
}

/** Decodes a zone target key into its space and zone-local axial coordinates, mirroring backend's invertZoneKey */
export function invertZoneKey(key: number): { space: Point2D; zone: Point2D } {
  const outer = invertPair(key);
  return { space: invertPair(outer.x), zone: invertPair(outer.y) };
}

/** Decodes a build target key into the building id plus its zone's coordinates, mirroring backend's invertBuildKey */
export function invertBuildKey(key: number): { building_id: number; space: Point2D; zone: Point2D } {
  const outer = invertPair(key);
  const { space, zone } = invertZoneKey(outer.y);
  return { building_id: outer.x, space, zone };
}
