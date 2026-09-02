/** Fill style for healthbar background rect */
export const UNIT_HEALTHBAR_COLOR_BACKGROUND = 'black';

/** Fill style for healthbar health rect */
export const UNIT_HEALTHBAR_COLOR_HEALTH = 'rgb(100, 250, 100)';

/** Returns image path of the unit */
export function unitImage(unit_id: number): string {
  let filename = '';
  switch (unit_id) {
    case 1:
      filename = 'villager';
      break;
    case 11:
      filename = 'swordsman';
      break;
    case 12:
      filename = 'villager_aoe';
      break;
    default:
      console.error('Trying to get unit image from unknown unit id', unit_id);
      return '';
  }
  return `risq/units/${filename}`;
}
