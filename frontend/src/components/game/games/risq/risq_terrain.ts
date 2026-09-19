import type { ColorRGB } from '../../../../scripts/color_rgb';
import type { RisqPlayer } from './risq_data';

export enum RisqViewMode {
  ALL = 0,
  RESOURCE = 1,
  MILITARY = 2,
  OWNERSHIP = 3,
}

export function nextViewMode(mode: RisqViewMode): RisqViewMode {
  return ((mode + 1) % 4) as RisqViewMode;
}

export function spaceOwnerColor(ownership: number | undefined, players: RisqPlayer[]): ColorRGB | undefined {
  if (ownership === undefined || ownership < 0 || ownership >= players.length) {
    return undefined;
  }
  return players[ownership].color;
}

/** All the terrain types */
export enum RisqTerrainType {
  NONE = 0,
  FLATLANDS = 1,
  HILLY = 2,
  MOUNTAINOUS = 3,
  SWAMP = 4,
  SHALLOWS = 5,
  WATER = 6,
  DEEP_WATER = 7,
}

export function terrainTypeLabel(terrain_type: RisqTerrainType): string {
  return (RisqTerrainType[terrain_type] ?? '').toLowerCase().replace(/_/g, ' ');
}

/** Returns image path of the terrain */
export function terrainImage(terrain_id: number): string {
  let filename = '';
  switch (terrain_id) {
    case 1:
      filename = 'grass_green1';
      break;
    case 2:
      filename = 'grass_green2';
      break;
    case 3:
      filename = 'grass_dead1';
      break;
    case 4:
      filename = 'grass_dead2';
      break;
    case 5:
      filename = 'grass_forest';
      break;
    case 6:
      filename = 'cobblestone_irregular';
      break;
    case 7:
      filename = 'cobblestone_regular';
      break;
    case 8:
      filename = 'dirt1';
      break;
    case 9:
      filename = 'dirt2';
      break;
    case 10:
      filename = 'dirt3';
      break;
    case 11:
      filename = 'dirt4';
      break;
    case 12:
      filename = 'dirt5';
      break;
    case 13:
      filename = 'gravel_compact';
      break;
    case 14:
      filename = 'leaves1';
      break;
    case 15:
      filename = 'sand1';
      break;
    case 16:
      filename = 'sand2';
      break;
    case 17:
      filename = 'sand3';
      break;
    case 18:
      filename = 'sand4';
      break;
    case 19:
      filename = 'sand5';
      break;
    case 20:
      filename = 'snow1';
      break;
    case 21:
      filename = 'snow2';
      break;
    case 22:
      filename = 'straw1';
      break;
    case 23:
      filename = 'straw2';
      break;
    case 51:
      filename = 'grass_green1_hills';
      break;
    case 52:
      filename = 'grass_green2_hills';
      break;
    case 53:
      filename = 'grass_dead1_hills';
      break;
    case 54:
      filename = 'grass_dead2_hills';
      break;
    case 55:
      filename = 'grass_forest_hills';
      break;
    case 56:
      filename = 'dirt1_hills';
      break;
    case 57:
      filename = 'dirt2_hills';
      break;
    case 58:
      filename = 'dirt3_hills';
      break;
    case 59:
      filename = 'dirt4_hills';
      break;
    case 60:
      filename = 'dirt5_hills';
      break;
    case 61:
      filename = 'snow1_hills';
      break;
    case 62:
      filename = 'snow2_hills';
      break;
    default:
      console.error('Trying to get terrain image from unknown terrain_id', terrain_id);
      filename = 'grass_green1';
  }
  return `risq/terrains/${filename}`;
}

/** Image path of the fog-of-war overlay texture, hex-cut to match a space's shape */
export const FOG_OVERLAY_IMAGE = 'risq/terrains/fog';
