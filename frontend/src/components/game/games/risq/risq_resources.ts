import type { RisqResource } from './risq_data';
import { RisqResourceType } from './risq_data';

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
      // wood
      case 11:
        filename = 'tree_cedar';
        break;
      case 12:
        filename = 'tree_dead';
        break;
      case 13:
        filename = 'tree_maple';
        break;
      case 14:
        filename = 'tree_oak';
        break;
      case 15:
        filename = 'tree_pine';
        break;
      case 16:
        filename = 'tree_walnut';
        break;
      // stone
      case 21:
        filename = 'stonemine';
        break;
      // gold
      case 31:
        filename = 'goldmine';
        break;
      default:
        console.error('Trying to get resource image from unknown resource id', resource.resource_id);
        return ``;
    }
  }
  return `risq/resources/${filename}`;
}

/** Returns the resource gathered from the resource type */
export function resourceType(resource: RisqResource): RisqResourceType {
  if (resource.resource_id < 11) {
    return RisqResourceType.FOOD;
  } else if (resource.resource_id < 21) {
    return RisqResourceType.WOOD;
  } else if (resource.resource_id < 31) {
    return RisqResourceType.STONE;
  } else if (resource.resource_id < 41) {
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
