import {RisqUnit} from "./risq_data";

/** Returns image path of the unit */
export function unitImage(unit: RisqUnit): string {
  let filename = '';
  switch(unit.unit_id) {
    case 1:
      filename = 'villager';
      break;
    case 11:
      filename = 'swordsman';
      break;
    default:
      console.error('Trying to get unit image from unknown unit id', unit.unit_id);
      return '';
  }
  return `risq/units/${filename}`;
}
