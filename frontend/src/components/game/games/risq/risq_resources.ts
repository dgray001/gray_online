import {capitalize} from '../../../../scripts/util';
import {RisqResource} from './risq_data';

/** Returns image of the resource */
export function resourceImage(resource: RisqResource): HTMLImageElement {
  let filename = 'empty_plot';
  if (!!resource) {
    switch(resource.resource_id) {
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
        return new Image();
    }
  }
  const img = new Image();
  img.src = `/images/risq/resources/${filename}.png`;
  img.alt = filename.split('_').map(word => capitalize(word)).join(' ');
  return img;
}
