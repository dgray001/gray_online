import { err } from '../../../../scripts/log';

/** Fill style for healthbar background rect */
export const UNIT_HEALTHBAR_COLOR_BACKGROUND = 'black';

/** Fill style for healthbar health rect */
export const UNIT_HEALTHBAR_COLOR_HEALTH = 'rgb(100, 250, 100)';

/** Returns image path of the unit; pass plain to get the uncolored source (no player-color marker pixels) */
export function unitImage(unit_id: number, plain = false): string {
  let filename = '';
  switch (unit_id) {
    case 1:
      filename = 'villager';
      break;
    case 11:
    case 12:
    case 13:
      filename = 'swordsman';
      break;
    default:
      err('Trying to get unit image from unknown unit id', unit_id);
      return '';
  }
  if (plain) {
    filename += '_plain';
  }
  return `risq/units/${filename}`;
}

export const COMBO_UNIT_ICON_SIZE = 64;

export function comboUnitIconKey(white: boolean): string {
  return white ? 'combo_unit_white' : 'combo_unit';
}

export function drawComboUnitIcon(
  ctx: CanvasRenderingContext2D,
  villager_icon: HTMLImageElement,
  unit_icon: HTMLImageElement
) {
  const icon_size = 0.72 * COMBO_UNIT_ICON_SIZE;
  ctx.drawImage(villager_icon, 0, 0, icon_size, icon_size);
  ctx.drawImage(unit_icon, COMBO_UNIT_ICON_SIZE - icon_size, COMBO_UNIT_ICON_SIZE - icon_size, icon_size, icon_size);
}
