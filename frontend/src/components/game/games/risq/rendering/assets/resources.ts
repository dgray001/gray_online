import { err } from '../../../../../../scripts/log';
import type { Point2D } from '../../../../util/objects2d';
import type { RisqDrawHost } from '../draw_host';
import type { RisqResource, RisqResourceConfig } from '../../model/types';
import { RisqResourceType } from '../../model/types';
import { PLAYER_ICON_SIZE } from './image_cache';

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
  return [31, 32, 33, 34, 35, 36].includes(resource.resource_id);
}

const GROVE_RESOURCE_IDS = new Set<number>([21, 22, 23, 24, 25, 26]);

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
  } else if (resource.resource_id < 41) {
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
