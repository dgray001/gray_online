import {RisqBuilding} from './risq_data';

/** Returns image path of the building */
export function buildingImage(building: RisqBuilding): string {
  let filename = 'empty_plot';
  if (!!building) {
    switch(building.building_id) {
      case 1:
        filename = 'village_center';
        break;
      case 2:
        filename = 'housing';
        break;
      default:
        console.error('Trying to get building image from unknown building id', building.building_id);
        return '';
    }
  }
  return `risq/buildings/${filename}`;
}
