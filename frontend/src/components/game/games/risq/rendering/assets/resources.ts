import { err } from '../../../../../../scripts/log';
import { clampNumber, seededRandom } from '../../../../../../scripts/math';
import type { Point2D } from '../../../../util/objects2d';
import { rotatePoint } from '../../../../util/objects2d';
import type { RisqDrawHost } from '../board/board_renderer';
import type { RisqResource, RisqResourceConfig, RisqZone } from '../../model/types';
import { RisqResourceType } from '../../model/types';
import { RisqViewMode } from '../terrain';
import {
  BUILDING_CIRCLE_RADIUS_MULTIPLIER,
  CENTER_ZONE_APOTHEM_MULTIPLIER,
  buildingLocalOffset,
  clipToZone,
  edgeSlotAtAngle,
  findOuterZoneIndex,
} from '../zones/geometry';
import { PLAYER_ICON_SIZE, transformScale } from './image_cache';

// ordered smallest first; a mine of resource_id N shows the stages with resource_id <= N as it depletes
const STONE_MINE_STAGES: { resource_id: number; filename: string }[] = [
  { resource_id: 41, filename: 'stonemine_small' },
  { resource_id: 42, filename: 'stonemine_medium' },
  { resource_id: 43, filename: 'stonemine_large' },
];

function stoneMineStageImage(resource: RisqResource, configs: ReadonlyMap<number, RisqResourceConfig>): string {
  const stages = STONE_MINE_STAGES.filter((s) => s.resource_id <= resource.resource_id);
  const stage = stages.find((s) => resource.resources_left <= (configs.get(s.resource_id)?.starting_resources ?? -1));
  return (stage ?? stages[stages.length - 1]).filename;
}

/** Returns image path of the resource, which changes as the resource depletes */
export function resourceImage(resource: RisqResource, configs: ReadonlyMap<number, RisqResourceConfig>): string {
  let filename = 'error';
  if (!!resource) {
    switch (resource.resource_id) {
      // food
      case 1:
        filename = 'forage_bush';
        break;
      case 2:
        filename = 'deer';
        break;
      // wood: single tree (1x), grove (2x), forest (3x) of the same species
      case 11:
      case 21:
      case 31:
        filename = 'tree_cedar';
        break;
      case 12:
      case 22:
      case 32:
        filename = 'tree_dead';
        break;
      case 13:
      case 23:
      case 33:
        filename = 'tree_maple';
        break;
      case 14:
      case 24:
      case 34:
        filename = 'tree_oak';
        break;
      case 15:
      case 25:
      case 35:
        filename = 'tree_pine';
        break;
      case 16:
      case 26:
      case 36:
        filename = 'tree_walnut';
        break;
      case 111:
      case 121:
      case 131:
        filename = 'tree_palm';
        break;
      case 112:
      case 122:
      case 132:
        filename = 'tree_acacia';
        break;
      case 113:
      case 123:
      case 133:
        filename = 'tree_baobab';
        break;
      case 114:
      case 124:
      case 134:
        filename = 'tree_birch';
        break;
      case 115:
      case 125:
      case 135:
        filename = 'tree_juniper';
        break;
      case 116:
      case 126:
      case 136:
        filename = 'tree_spruce';
        break;
      case 117:
      case 127:
      case 137:
        filename = 'tree_willow';
        break;
      // stone
      case 41:
      case 42:
      case 43:
        filename = stoneMineStageImage(resource, configs);
        break;
      // gold
      case 51:
        filename = 'goldmine';
        break;
      default:
        err('Trying to get resource image from unknown resource id', resource.resource_id);
        return 'default';
    }
  }
  return `risq/resources/${filename}`;
}

// Forests get a scatter of tree icons drawn across their whole zone
export function isForestResource(resource: RisqResource): boolean {
  return [31, 32, 33, 34, 35, 36, 131, 132, 133, 134, 135, 136, 137].includes(resource.resource_id);
}

const GROVE_RESOURCE_IDS = new Set<number>([21, 22, 23, 24, 25, 26, 121, 122, 123, 124, 125, 126, 127]);

/** Sizes and back-tree centers as fractions of the icon box */
interface TreeClusterLayout {
  name: string;
  front_size: number;
  back_size: number;
  back_centers: Point2D[];
}

const FOREST_CLUSTER: TreeClusterLayout = {
  name: 'forest',
  front_size: 0.75,
  back_size: 0.6,
  back_centers: [-Math.PI / 2, (5 * Math.PI) / 6, Math.PI / 6].map((a) => ({
    x: 0.5 + 0.17 * Math.cos(a),
    y: 0.48 + 0.17 * Math.sin(a),
  })),
};

const FOREST_THREE_TREE_CLUSTER: TreeClusterLayout = {
  name: 'forest_three',
  front_size: FOREST_CLUSTER.front_size,
  back_size: FOREST_CLUSTER.back_size,
  back_centers: [(-5 * Math.PI) / 6, -Math.PI / 6].map((a) => ({
    x: 0.5 + 0.17 * Math.cos(a),
    y: 0.48 + 0.17 * Math.sin(a),
  })),
};

const GROVE_CLUSTER: TreeClusterLayout = {
  name: 'grove',
  front_size: 0.85,
  back_size: 0.65,
  back_centers: [
    { x: 0.3, y: 0.4 },
    { x: 0.7, y: 0.4 },
  ],
};

const FOREST_THREE_TREES_BELOW_RESOURCES_LEFT = 800;

function treeClusterLayout(resource: RisqResource): TreeClusterLayout | undefined {
  if (isForestResource(resource)) {
    return resource.resources_left < FOREST_THREE_TREES_BELOW_RESOURCES_LEFT
      ? FOREST_THREE_TREE_CLUSTER
      : FOREST_CLUSTER;
  }
  return GROVE_RESOURCE_IDS.has(resource.resource_id) ? GROVE_CLUSTER : undefined;
}

const DEER_RESOURCE_ID = 2;
const DEER_PAIR_MIN_RESOURCES_LEFT = 100;

const PAIR_FRONT_SIZE = 0.8;
const PAIR_BACK_SIZE = 0.68;
const TREE_PAIR_BELOW_RESOURCES_LEFT = 200;

function drawPairedIcon(ctx: CanvasRenderingContext2D, icon: HTMLImageElement, s: number): void {
  ctx.drawImage(icon, (1 - PAIR_BACK_SIZE) * s, 0.04 * s, PAIR_BACK_SIZE * s, PAIR_BACK_SIZE * s);
  ctx.drawImage(icon, 0, (1 - PAIR_FRONT_SIZE) * s, PAIR_FRONT_SIZE * s, PAIR_FRONT_SIZE * s);
}

function drawSingleDeer(ctx: CanvasRenderingContext2D, deer: HTMLImageElement, s: number): void {
  const offset = 0.5 * (1 - PAIR_FRONT_SIZE) * s;
  ctx.drawImage(deer, offset, offset, PAIR_FRONT_SIZE * s, PAIR_FRONT_SIZE * s);
}

/** Returns the resource's icon; forests and groves get a cached cluster, and a full deer gets a cached pair */
export function resourceIcon(game: RisqDrawHost, resource: RisqResource): HTMLImageElement | HTMLCanvasElement {
  const configs = game.session.getResourceConfigs();
  const image = resourceImage(resource, configs);
  const icon = game.getIcon(image);
  if (resource.resource_id === DEER_RESOURCE_ID) {
    const pair = resource.resources_left > DEER_PAIR_MIN_RESOURCES_LEFT;
    const draw = pair ? drawPairedIcon : drawSingleDeer;
    const deer = game
      .getImageCache()
      .getImage(`deer_${pair ? 'pair' : 'single'}:${image}`, PLAYER_ICON_SIZE, [icon], (ctx) => {
        draw(ctx, icon, PLAYER_ICON_SIZE);
      });
    return deer ?? icon;
  }
  const is_tree_cluster = GROVE_RESOURCE_IDS.has(resource.resource_id) || isForestResource(resource);
  if (is_tree_cluster && resource.resources_left < TREE_PAIR_BELOW_RESOURCES_LEFT) {
    const tree_pair = game.getImageCache().getImage(`tree_pair:${image}`, PLAYER_ICON_SIZE, [icon], (ctx) => {
      drawPairedIcon(ctx, icon, PLAYER_ICON_SIZE);
    });
    return tree_pair ?? icon;
  }
  const layout = treeClusterLayout(resource);
  if (!layout) {
    return icon;
  }
  const s = PLAYER_ICON_SIZE;
  const key = `${layout.name}_cluster:${image}`;
  const cluster = game.getImageCache().getImage(key, s, [icon], (ctx) => {
    const back = layout.back_size * s;
    for (const c of layout.back_centers) {
      ctx.drawImage(icon, c.x * s - 0.5 * back, c.y * s - 0.5 * back, back, back);
    }
    const front = layout.front_size * s;
    ctx.drawImage(icon, 0.5 * (s - front), s - front, front, front);
  });
  return cluster ?? icon;
}

/** Returns the resource gathered from the resource type */
export function resourceType(resource: RisqResource): RisqResourceType {
  if (resource.resource_id < 11) {
    return RisqResourceType.FOOD;
  } else if (resource.resource_id < 41 || (resource.resource_id > 110 && resource.resource_id < 140)) {
    return RisqResourceType.WOOD;
  } else if (resource.resource_id < 51) {
    return RisqResourceType.STONE;
  } else if (resource.resource_id < 61) {
    return RisqResourceType.GOLD;
  }
  return RisqResourceType.ERROR;
}

/** Returns the image path of the resource type */
export function resourceTypeImage(resource: RisqResource | RisqResourceType): string {
  let filename = 'error';
  const resource_type = typeof resource === 'number' ? resource : resourceType(resource);
  switch (resource_type) {
    case RisqResourceType.FOOD:
      filename = 'food';
      break;
    case RisqResourceType.WOOD:
      filename = 'wood';
      break;
    case RisqResourceType.STONE:
      filename = 'stone';
      break;
    case RisqResourceType.GOLD:
      filename = 'gold';
      break;
  }
  return `risq/resources/${filename}`;
}

const CENTER_FOREST_TREES = 8;
const EDGE_FOREST_TREES = 6;
const FOREST_TREE_RADIUS_MULTIPLIER = 0.11;
const FOREST_TREE_RING_FRACTION = 0.4;
const FOREST_MAX_PERTURBATION = 0.02;
const FOREST_OUTER_RING_BELOW_RESOURCES_LEFT = 1000;
const FOREST_INNER_RING_BELOW_RESOURCES_LEFT = 600;
const CENTER_INNER_FOREST_ANGLES = [0, Math.PI / 2, Math.PI, (3 * Math.PI) / 2];
const EDGE_INNER_FOREST_ANGLES = CENTER_INNER_FOREST_ANGLES.slice(1);

const HEXAGON_EDGE_NORMALS: Point2D[] = Array.from({ length: 6 }, (_, k) => ({
  x: Math.cos((k * Math.PI) / 3),
  y: Math.sin((k * Math.PI) / 3),
}));

/** Scales the point toward the center zone's middle until a tree centered there fits fully inside the zone */
function clampToCenterZone(p: Point2D, hex_r: number): Point2D {
  const limit = (CENTER_ZONE_APOTHEM_MULTIPLIER - FOREST_TREE_RADIUS_MULTIPLIER) * hex_r;
  const reach = Math.max(...HEXAGON_EDGE_NORMALS.map((n) => p.x * n.x + p.y * n.y));
  const scale = reach > limit ? limit / reach : 1;
  return { x: p.x * scale, y: p.y * scale };
}

/** Moves the point into the edge zone's trapezoid (local frame, zone toward +x) so a tree there fits fully inside */
function clampToEdgeZone(p: Point2D, hex_r: number): Point2D {
  const tree_r = FOREST_TREE_RADIUS_MULTIPLIER * hex_r;
  const x = clampNumber(p.x, CENTER_ZONE_APOTHEM_MULTIPLIER * hex_r + tree_r, (Math.sqrt(3) / 2) * hex_r - tree_r);
  const max_y = x * Math.tan(Math.PI / 6) - tree_r / Math.cos(Math.PI / 6);
  return { x, y: clampNumber(p.y, -max_y, max_y) };
}

declare interface ForestTreeSpots {
  inner: Point2D[];
  outer: Point2D[];
}

/** Local-frame tree spots: a ring (center) or inward arc (edge), plus an inner square/right triangle at half that distance */
function forestTreeLocalOffsets(is_center: boolean, hex_r: number, seed: number): ForestTreeSpots {
  const random = seededRandom(seed);
  const outer_step = is_center ? (2 * Math.PI) / CENTER_FOREST_TREES : Math.PI / (EDGE_FOREST_TREES - 1);
  const inner_step = Math.PI / 2;
  // rings rotate within one step; arcs slide by up to half a step either way so their ends stay near the zone
  const jitter = (step: number): number => (is_center ? random() * step : (random() - 0.5) * step);
  const outer_offset = jitter(outer_step);
  const inner_offset = jitter(inner_step);
  const building = buildingLocalOffset(is_center, hex_r);
  const gap = CENTER_ZONE_APOTHEM_MULTIPLIER - BUILDING_CIRCLE_RADIUS_MULTIPLIER;
  const center_ring_r = (BUILDING_CIRCLE_RADIUS_MULTIPLIER + FOREST_TREE_RING_FRACTION * gap) * hex_r;
  const ring_at = (angle: number): Point2D =>
    is_center
      ? { x: center_ring_r * Math.cos(angle), y: center_ring_r * Math.sin(angle) }
      : edgeSlotAtAngle(angle, hex_r, FOREST_TREE_RING_FRACTION);
  const ring_angles = (
    is_center
      ? Array.from({ length: CENTER_FOREST_TREES }, (_, i) => (2 * Math.PI * i) / CENTER_FOREST_TREES)
      : Array.from({ length: EDGE_FOREST_TREES }, (_, i) => Math.PI / 2 + (i * Math.PI) / (EDGE_FOREST_TREES - 1))
  ).map((angle) => angle + outer_offset);
  const inner = (is_center ? CENTER_INNER_FOREST_ANGLES : EDGE_INNER_FOREST_ANGLES).map((angle) => {
    const p = ring_at(angle + inner_offset);
    return { x: 0.5 * (building.x + p.x), y: 0.5 * (building.y + p.y) };
  });
  const perturbation = (): number => (random() - 0.5) * 2 * FOREST_MAX_PERTURBATION * hex_r;
  const clamp_to_zone = is_center ? clampToCenterZone : clampToEdgeZone;
  const place = (p: Point2D): Point2D => clamp_to_zone({ x: p.x + perturbation(), y: p.y + perturbation() }, hex_r);
  return { inner: inner.map(place), outer: ring_angles.map(ring_at).map(place) };
}

/** Draws a forest zone's trees upright around its space's center (the current origin), clipped to the zone */
export function drawForestTrees(
  ctx: CanvasRenderingContext2D,
  game: RisqDrawHost,
  zone: RisqZone,
  view_mode: RisqViewMode,
  hex_r: number,
  map_rotation: number
): void {
  const resource = zone.resource;
  const hidden_view = view_mode === RisqViewMode.MILITARY || view_mode === RisqViewMode.OWNERSHIP;
  if (!resource || !isForestResource(resource) || hidden_view) {
    return;
  }
  const direction = findOuterZoneIndex(zone.coordinate);
  const zone_rotation = direction === -1 ? 0 : (Math.PI / 3) * (direction + 1);
  const icon_r = FOREST_TREE_RADIUS_MULTIPLIER * hex_r;
  const icon = game
    .getImageCache()
    .getScaledSprite(
      game.getIcon(resourceImage(resource, game.session.getResourceConfigs())),
      2 * icon_r * transformScale(ctx)
    );
  ctx.save();
  clipToZone(ctx, { x: 0, y: 0 }, hex_r, zone.coordinate);
  const spots = forestTreeLocalOffsets(direction === -1, hex_r, zone.coordinate_key);
  const visible_spots = [
    ...(resource.resources_left < FOREST_INNER_RING_BELOW_RESOURCES_LEFT ? [] : spots.inner),
    ...(resource.resources_left < FOREST_OUTER_RING_BELOW_RESOURCES_LEFT ? [] : spots.outer),
  ];
  for (const offset of visible_spots) {
    const p = rotatePoint(offset, zone_rotation);
    ctx.translate(p.x, p.y);
    ctx.rotate(-map_rotation);
    ctx.drawImage(icon, -icon_r, -icon_r, 2 * icon_r, 2 * icon_r);
    ctx.rotate(map_rotation);
    ctx.translate(-p.x, -p.y);
  }
  ctx.restore();
}
