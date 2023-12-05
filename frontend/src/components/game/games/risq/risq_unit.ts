import {capitalize} from "../../../../scripts/util";
import {RisqUnit} from "./risq_data";

/** Returns image of the unit */
export function unitImage(unit: RisqUnit): HTMLImageElement {
  let filename = '';
  switch(unit.unit_id) {
    case 1:
      filename = 'villager';
      break;
    default:
      console.error('Trying to get unit image from unknown unit id', unit.unit_id);
      return new Image();
  }
  const img = new Image();
  img.src = `/images/risq/units/${filename}.png`;
  img.alt = filename.split('_').map(word => capitalize(word)).join(' ');
  // TODO: add player color
  return img;
}
