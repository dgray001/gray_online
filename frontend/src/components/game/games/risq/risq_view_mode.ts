import type { ColorRGB } from '../../../../scripts/color_rgb';
import type { RisqPlayer, RisqSpace } from './risq_data';

export enum RisqViewMode {
  ALL = 0,
  RESOURCE = 1,
  MILITARY = 2,
  OWNERSHIP = 3,
}

export function nextViewMode(mode: RisqViewMode): RisqViewMode {
  return ((mode + 1) % 4) as RisqViewMode;
}

export function spaceOwnerColor(space: RisqSpace, players: RisqPlayer[]): ColorRGB | undefined {
  const ownership = space.ownership;
  if (ownership === undefined || ownership < 0 || ownership >= players.length) {
    return undefined;
  }
  return players[ownership].color;
}
