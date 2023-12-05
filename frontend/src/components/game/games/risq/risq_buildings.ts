import {capitalize} from "../../../../scripts/util";
import {RisqBuilding} from "./risq_data";

/** Returns image of the building */
export function buildingImage(building: RisqBuilding): HTMLImageElement {
  let filename = '';
  switch(building.building_id) {
    case 1:
      filename = 'village_center';
      break;
    case 2:
      filename = 'housing';
      break;
    default:
      console.error('Trying to get building image from unknown building id', building.building_id);
      return new Image();
  }
  const img = new Image();
  img.src = `/images/risq/buildings/${filename}.png`;
  img.alt = filename.split('_').map(word => capitalize(word)).join(' ');
  // TODO: add player color
  return img;
}
