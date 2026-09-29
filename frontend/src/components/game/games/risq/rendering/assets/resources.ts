import { err } from '../../../../../../scripts/log';
import type { Point2D } from '../../../../util/objects2d';
import type { DwgRisq } from '../../risq';
import type { RisqResource } from '../../model/types';
import { RisqResourceType } from '../../model/types';
import { PLAYER_ICON_SIZE } from './image_cache';

/** Returns image path of the resource */
export function resourceImage(resource: RisqResource): string {
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
        filename = 'stonemine';
        break;
      // gold
      case 51:
        filename = 'goldmine';
        break;
      default:
        err('Trying to get resource image from unknown resource id', resource.resource_id);
        return ``;
    }
  }
  return `risq/resources/${filename}`;
}

// resource_ids for dense "forest" resource variants (as opposed to "grove" or single "tree" resources of the same
// species), which get a scatter of tree icons drawn across their whole zone instead of just their own icon.
// Add each forest resource_id here once it's added to config/resources.json.
const FOREST_RESOURCE_IDS = new Set<number>([31, 32, 33, 34, 35, 36]);

export function isForestResource(resource: RisqResource): boolean {
  return FOREST_RESOURCE_IDS.has(resource.resource_id);
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

const GROVE_CLUSTER: TreeClusterLayout = {
  name: 'grove',
  front_size: 0.85,
  back_size: 0.65,
  back_centers: [
    { x: 0.3, y: 0.4 },
    { x: 0.7, y: 0.4 },
  ],
};

function treeClusterLayout(resource: RisqResource): TreeClusterLayout | undefined {
  if (isForestResource(resource)) {
    return FOREST_CLUSTER;
  }
  return GROVE_RESOURCE_IDS.has(resource.resource_id) ? GROVE_CLUSTER : undefined;
}

/** Returns the resource's icon; forests and groves get a cached cluster of their tree in front of smaller copies */
export function resourceIcon(game: DwgRisq, resource: RisqResource): HTMLImageElement | HTMLCanvasElement {
  const icon = game.getIcon(resourceImage(resource));
  const layout = treeClusterLayout(resource);
  if (!layout) {
    return icon;
  }
  const s = PLAYER_ICON_SIZE;
  const key = `${layout.name}_cluster:${resourceImage(resource)}`;
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
