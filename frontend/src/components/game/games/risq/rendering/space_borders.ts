import { drawLine } from '../../../util/canvas_util';
import type { Point2D } from '../../../util/objects2d';
import { AXIAL_DIRECTION_VECTORS, addPoint2D } from '../../../util/objects2d';
import { coordinateToIndex, getSpace } from '../model/coordinates';
import type { RisqRegion, RisqSpace } from '../model/types';
import type { RisqDrawHost } from './draw_host';
import type { DrawRisqSpaceConfig } from './space';
import { SPACE_BORDER_REFERENCE_RADIUS, borderStrokeStyle, isOverviewViewMode, spaceBorderWidth } from './space';
import type { RisqViewMode } from './terrain';
import { spaceOwnerColor } from './terrain';

function regionBorderAddition(region: RisqRegion | undefined, game: RisqDrawHost, overview: boolean): number {
  if (!region) {
    return 0;
  }
  return (
    (overview ? 0.6 : 0.4) +
    (region === game.hover.hoveredRegion() ? 0.2 : 0) +
    (game.selection.isRegionSelected(region) ? 0.2 : 0)
  );
}

function sharedEdgeStyle(
  ctx: CanvasRenderingContext2D,
  center: Point2D,
  normal: Point2D,
  width: number,
  color: string,
  neighbor_color: string
): CanvasGradient {
  const gradient = ctx.createLinearGradient(
    center.x - normal.x * width,
    center.y - normal.y * width,
    center.x + normal.x * width,
    center.y + normal.y * width
  );
  gradient.addColorStop(0, color);
  gradient.addColorStop(0.5, color);
  gradient.addColorStop(0.5, neighbor_color);
  gradient.addColorStop(1, neighbor_color);
  return gradient;
}

export function hexEdgeForDirectionIndex(c: Point2D, r: number, direction_index: number): [Point2D, Point2D] {
  const angle = Math.PI / 6 + (5 - direction_index) * (Math.PI / 3);
  return [
    { x: c.x + r * Math.cos(angle), y: c.y + r * Math.sin(angle) },
    { x: c.x + r * Math.cos(angle + Math.PI / 3), y: c.y + r * Math.sin(angle + Math.PI / 3) },
  ];
}

export function drawRisqSpaceBorders(
  ctx: CanvasRenderingContext2D,
  game: RisqDrawHost,
  spaces: RisqSpace[],
  config: Pick<DrawRisqSpaceConfig, 'hex_r' | 'draw_detail'> & {
    view_mode?: RisqViewMode;
    regions_only?: boolean;
    black_regions?: boolean;
    min_width?: number;
  },
  center_of: (space: RisqSpace) => Point2D = (space) => space.center
): void {
  const game_data = game.getGame();
  if (!game_data) {
    return;
  }
  const visible_keys = new Set(spaces.map((space) => space.coordinate_key));
  const width = spaceBorderWidth(config);
  const overview = isOverviewViewMode(config.view_mode);
  const neighbor_at = (space: RisqSpace, direction: number): RisqSpace | undefined =>
    getSpace(
      game_data,
      coordinateToIndex(game_data.board_size, addPoint2D(space.coordinate, AXIAL_DIRECTION_VECTORS[direction]))
    );
  ctx.save();
  ctx.lineCap = 'butt';
  for (const space of spaces) {
    const color = borderStrokeStyle(spaceOwnerColor(space.ownership, game_data.players), 1);
    for (const direction of AXIAL_DIRECTION_VECTORS.keys()) {
      const neighbor = neighbor_at(space, direction);
      if (neighbor && visible_keys.has(neighbor.coordinate_key) && space.coordinate_key > neighbor.coordinate_key) {
        continue;
      }
      const region = game.session.getRegionForSpace(space.coordinate_key);
      const neighbor_region = neighbor ? game.session.getRegionForSpace(neighbor.coordinate_key) : undefined;
      const region_edge = region !== neighbor_region;
      if (config.regions_only && !region_edge) {
        continue;
      }
      const addition = region_edge
        ? Math.max(regionBorderAddition(region, game, overview), regionBorderAddition(neighbor_region, game, overview))
        : 0;
      const scaled_width = width + addition * (config.hex_r / SPACE_BORDER_REFERENCE_RADIUS);
      const edge_width = Math.max(scaled_width, config.min_width ?? 0);
      const [hex_start, hex_end] = hexEdgeForDirectionIndex(center_of(space), config.hex_r, direction);
      const along = { x: (hex_end.x - hex_start.x) / config.hex_r, y: (hex_end.y - hex_start.y) / config.hex_r };
      const miter = edge_width / (2 * Math.sqrt(3));
      const convex_start = !neighbor && !neighbor_at(space, (direction + 1) % 6);
      const convex_end = !neighbor && !neighbor_at(space, (direction + 5) % 6);
      const start = convex_start ? { x: hex_start.x - miter * along.x, y: hex_start.y - miter * along.y } : hex_start;
      const end = convex_end ? { x: hex_end.x + miter * along.x, y: hex_end.y + miter * along.y } : hex_end;
      const angle = (-direction * Math.PI) / 3;
      const normal = { x: Math.cos(angle), y: Math.sin(angle) };
      const center = { x: (start.x + end.x) / 2, y: (start.y + end.y) / 2 };
      const neighbor_color = neighbor
        ? borderStrokeStyle(spaceOwnerColor(neighbor.ownership, game_data.players), 1)
        : undefined;
      ctx.lineWidth = edge_width;
      ctx.strokeStyle =
        region_edge && addition > 0 && config.black_regions !== false
          ? 'black'
          : neighbor_color && neighbor_color !== color
            ? sharedEdgeStyle(ctx, center, normal, edge_width / 2, color, neighbor_color)
            : color;
      drawLine(ctx, start, end);
    }
  }
  ctx.restore();
}
