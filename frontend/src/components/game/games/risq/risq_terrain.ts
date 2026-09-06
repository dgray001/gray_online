import type { ColorRGB } from '../../../../scripts/color_rgb';
import { capitalize } from '../../../../scripts/util';
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

/** All the terrain types */
export enum RisqTerrainType {
  FLATLANDS = 0,
  HILLY = 1,
  MOUNTAINOUS = 2,
  VALLEY = 3,
  SWAMP = 4,
  SHALLOWS = 5,
  WATER = 6,
  DEEP_WATER = 7,
}

/** Converts terrain type to a string */
export function risqTerrainName(terrain: RisqTerrainType): string {
  return capitalize(RisqTerrainType[terrain].replace('_', ' ').toLowerCase());
}

/** Returns image path of the terrain */
export function terrainImage(terrain: RisqTerrainType): string {
  switch (terrain) {
    case RisqTerrainType.FLATLANDS:
      return 'risq/terrains/grass';
    default:
      console.error('Trying to get terrain image from unknown terrain type', terrain);
      return 'risq/terrains/grass';
  }
}

/** Image path of the fog-of-war overlay texture, hex-cut to match a space's shape */
export const FOG_OVERLAY_IMAGE = 'risq/terrains/fog';
