import { drawLine, drawText } from '../../util/canvas_util';
import type { Point2D } from '../../util/objects2d';
import { AXIAL_DIRECTION_VECTORS, addPoint2D } from '../../util/objects2d';
import { coordinateToIndex, getSpace } from './risq_coordinates';
import type { GameRisq, RisqRegion, RisqSpace } from './risq_data';
import type { DwgRisq } from './risq';
import { spaceOwnerColor } from './risq_terrain';

/** Maps each explored space's coordinate key to the region it belongs to */
export function buildRegionLookup(regions: RisqRegion[]): Map<number, RisqRegion> {
  const lookup = new Map<number, RisqRegion>();
  for (const region of regions) {
    for (const key of region.spaces) {
      lookup.set(key, region);
    }
  }
  return lookup;
}

/** Endpoints of the hex edge facing the AXIAL_DIRECTION_VECTORS[direction_index] neighbor, matching drawHexagon's vertex layout */
function hexEdgeForDirectionIndex(c: Point2D, r: number, direction_index: number): [Point2D, Point2D] {
  const i0 = 5 - direction_index;
  const angle0 = Math.PI / 6 + i0 * (Math.PI / 3);
  const angle1 = angle0 + Math.PI / 3;
  return [
    { x: c.x + r * Math.cos(angle0), y: c.y + r * Math.sin(angle0) },
    { x: c.x + r * Math.cos(angle1), y: c.y + r * Math.sin(angle1) },
  ];
}

/** Direction indices at which this space's hex borders a different (or unexplored/off-board) region */
function regionBorderDirections(
  space: RisqSpace,
  region: RisqRegion,
  region_lookup: Map<number, RisqRegion>,
  game: GameRisq
): number[] {
  const directions: number[] = [];
  for (const [i, dv] of AXIAL_DIRECTION_VECTORS.entries()) {
    const neighbor = getSpace(game, coordinateToIndex(game.board_size, addPoint2D(space.coordinate, dv)));
    if (region_lookup.get(neighbor?.coordinate_key ?? -1) !== region) {
      directions.push(i);
    }
  }
  return directions;
}

/** Draws each on-screen region's perimeter: thicker and owner-colored when fully owned, thin white otherwise */
export function drawRisqRegionBorders(
  ctx: CanvasRenderingContext2D,
  game: DwgRisq,
  spaces: RisqSpace[],
  hex_r: number,
  center_of: (space: RisqSpace) => Point2D = (space) => space.center
) {
  const region_lookup = game.getRegionLookup();
  const game_data = game.getGame();
  if (!game_data) {
    return;
  }
  const hovered_region = game.hoveredRegion();
  for (const space of spaces) {
    const region = region_lookup.get(space.coordinate_key);
    if (!region) {
      continue;
    }
    const hovered = region === hovered_region;
    const owner_color = region.owner >= 0 ? spaceOwnerColor(region.owner, game_data.players) : undefined;
    ctx.strokeStyle = owner_color
      ? `rgba(${owner_color.getR()}, ${owner_color.getG()}, ${owner_color.getB()}, ${hovered ? 1 : 0.95})`
      : `rgba(255, 255, 255, ${hovered ? 0.85 : 0.55})`;
    ctx.lineWidth = (owner_color ? 3 : 1.5) + (hovered ? 1 : 0) + (game.isRegionSelected(region) ? 2 : 0);
    for (const i of regionBorderDirections(space, region, region_lookup, game_data)) {
      const [p0, p1] = hexEdgeForDirectionIndex(center_of(space), hex_r, i);
      drawLine(ctx, p0, p1);
    }
  }
}

/** Draws each known region's name and gold bonus at its centroid; used by the REGION view mode */
export function drawRisqRegionLabels(ctx: CanvasRenderingContext2D, game: DwgRisq) {
  const game_data = game.getGame();
  if (!game_data) {
    return;
  }
  const drawn = new Set<RisqRegion>();
  for (const region of game.getRegionLookup().values()) {
    if (drawn.has(region)) {
      continue;
    }
    drawn.add(region);
    const p = game.regionLabelPosition(region);
    const owner_color = region.owner >= 0 ? spaceOwnerColor(region.owner, game_data.players) : undefined;
    drawText(ctx, region.name, {
      p: { x: p.x, y: p.y - 8 },
      w: 220,
      fill_style: owner_color ? owner_color.getString() : 'white',
      stroke_style: 'black',
      stroke_width: 3,
      align: 'center',
      baseline: 'middle',
      font: 'bold 16px serif',
    });
    drawText(ctx, `+${region.gold_bonus} gold/turn`, {
      p: { x: p.x, y: p.y + 10 },
      w: 220,
      fill_style: 'rgb(255, 215, 0)',
      stroke_style: 'black',
      stroke_width: 2,
      align: 'center',
      baseline: 'middle',
      font: '13px serif',
    });
  }
}
