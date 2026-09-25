import { err } from '../../../../scripts/log';

/** Returns image path of the building; pass plain to get the uncolored source (no player-color marker pixels) */
export function buildingImage(building_id: number | undefined, under_construction?: boolean, plain = false): string {
  if (building_id !== undefined && under_construction) {
    return 'risq/buildings/construction';
  }
  let filename = 'empty_plot';
  if (building_id !== undefined) {
    switch (building_id) {
      case 1:
        filename = 'village_center';
        break;
      case 2:
        filename = 'housing';
        break;
      case 21:
        filename = 'outpost';
        break;
      case 23:
        filename = 'redoubt';
        break;
      case 11:
        filename = 'blacksmith';
        break;
      case 22:
        filename = 'barracks';
        break;
      case 3:
        filename = 'farm';
        break;
      default:
        err('Trying to get building image from unknown building id', building_id);
        return '';
    }
    if (plain && filename !== 'empty_plot') {
      filename += '_plain';
    }
  }
  return `risq/buildings/${filename}`;
}
