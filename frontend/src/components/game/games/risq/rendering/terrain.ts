import type { ColorRGB } from '../../../../../scripts/color_rgb';
import { err } from '../../../../../scripts/log';
import type { RisqBuilding, RisqPlayer } from '../model/types';
import { RisqAttackType, RisqProducibleKind, RisqUnitType } from '../model/types';

export enum RisqViewMode {
  ALL = 0,
  RESOURCE = 1,
  MILITARY = 2,
  OWNERSHIP = 3,
}

/** The resource view needs an economy and the all view needs more than military units to show */
export function viewModeAllowed(mode: RisqViewMode, has_economy: boolean, military_only: boolean): boolean {
  if (mode === RisqViewMode.RESOURCE) {
    return has_economy;
  }
  return mode !== RisqViewMode.ALL || !military_only;
}

export function nextViewMode(mode: RisqViewMode, has_economy = true, military_only = false): RisqViewMode {
  let next = ((mode + 1) % 4) as RisqViewMode;
  while (!viewModeAllowed(next, has_economy, military_only)) {
    next = ((next + 1) % 4) as RisqViewMode;
  }
  return next;
}

export function unitVisibleInViewMode(unit_type: RisqUnitType, view_mode: RisqViewMode): boolean {
  if (view_mode === RisqViewMode.OWNERSHIP) {
    return false;
  }
  if (view_mode === RisqViewMode.MILITARY) {
    return unit_type !== RisqUnitType.ECONOMIC;
  }
  if (view_mode === RisqViewMode.RESOURCE) {
    return unit_type === RisqUnitType.ECONOMIC;
  }
  return true;
}

export function buildingVisibleInViewMode(building: RisqBuilding, view_mode: RisqViewMode): boolean {
  const produces_visible_unit = building.produces.some(
    (p) =>
      p.kind === RisqProducibleKind.UNIT && p.unit_type !== undefined && unitVisibleInViewMode(p.unit_type, view_mode)
  );
  if (view_mode === RisqViewMode.RESOURCE) {
    return produces_visible_unit || building.gather_capacity !== undefined;
  }
  if (view_mode === RisqViewMode.MILITARY) {
    return (
      produces_visible_unit ||
      building.combat_stats.attack_type !== RisqAttackType.NONE ||
      building.garrison_capacity > 0
    );
  }
  return true;
}

export function buildingOrdersVisibleInViewMode(building: RisqBuilding, view_mode: RisqViewMode): boolean {
  return view_mode !== RisqViewMode.RESOURCE && buildingVisibleInViewMode(building, view_mode);
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
      filename = 'grass_patchy';
      break;
    case 4:
      filename = 'grass_dead1';
      break;
    case 5:
      filename = 'grass_dead2';
      break;
    case 6:
      filename = 'grass_forest';
      break;
    case 7:
      filename = 'cobblestone_irregular';
      break;
    case 8:
      filename = 'cobblestone_regular';
      break;
    case 9:
      filename = 'dirt1';
      break;
    case 10:
      filename = 'dirt2';
      break;
    case 11:
      filename = 'dirt3';
      break;
    case 12:
      filename = 'dirt4';
      break;
    case 13:
      filename = 'dirt5';
      break;
    case 14:
      filename = 'gravel_compact';
      break;
    case 15:
      filename = 'leaves';
      break;
    case 16:
      filename = 'sand1';
      break;
    case 17:
      filename = 'sand2';
      break;
    case 18:
      filename = 'sand3';
      break;
    case 19:
      filename = 'sand4';
      break;
    case 20:
      filename = 'sand5';
      break;
    case 21:
      filename = 'snow1';
      break;
    case 22:
      filename = 'snow2';
      break;
    case 23:
      filename = 'straw1';
      break;
    case 24:
      filename = 'straw2';
      break;
    case 25:
      filename = 'farm_tilled';
      break;
    case 26:
      filename = 'farm_tilled_dead';
      break;
    case 51:
      filename = 'grass_green1_hills';
      break;
    case 52:
      filename = 'grass_green2_hills';
      break;
    case 53:
      filename = 'grass_patchy_hills';
      break;
    case 54:
      filename = 'grass_dead1_hills';
      break;
    case 55:
      filename = 'grass_dead2_hills';
      break;
    case 56:
      filename = 'grass_forest_hills';
      break;
    case 59:
      filename = 'dirt1_hills';
      break;
    case 60:
      filename = 'dirt2_hills';
      break;
    case 61:
      filename = 'dirt3_hills';
      break;
    case 62:
      filename = 'dirt4_hills';
      break;
    case 63:
      filename = 'dirt5_hills';
      break;
    case 66:
      filename = 'sand1_hills';
      break;
    case 67:
      filename = 'sand2_hills';
      break;
    case 68:
      filename = 'sand3_hills';
      break;
    case 69:
      filename = 'sand4_hills';
      break;
    case 70:
      filename = 'sand5_hills';
      break;
    case 71:
      filename = 'snow1_hills';
      break;
    case 72:
      filename = 'snow2_hills';
      break;
    case 75:
      filename = 'farm_tilled_hills';
      break;
    case 76:
      filename = 'farm_tilled_dead_hills';
      break;
    case 101:
      filename = 'grass_green1_mountains';
      break;
    case 102:
      filename = 'grass_green2_mountains';
      break;
    case 103:
      filename = 'grass_patchy_mountains';
      break;
    case 104:
      filename = 'grass_dead1_mountains';
      break;
    case 105:
      filename = 'grass_dead2_mountains';
      break;
    case 106:
      filename = 'grass_forest_mountains';
      break;
    case 109:
      filename = 'dirt1_mountains';
      break;
    case 110:
      filename = 'dirt2_mountains';
      break;
    case 111:
      filename = 'dirt3_mountains';
      break;
    case 112:
      filename = 'dirt4_mountains';
      break;
    case 113:
      filename = 'dirt5_mountains';
      break;
    case 121:
      filename = 'snow1_mountains';
      break;
    case 122:
      filename = 'snow2_mountains';
      break;
    case 151:
      filename = 'swamp_dead';
      break;
    case 152:
      filename = 'swamp_hybrid';
      break;
    case 153:
      filename = 'swamp_mossy';
      break;
    case 201:
      filename = 'shallows_boggy';
      break;
    case 202:
      filename = 'shallows_gravel';
      break;
    case 203:
      filename = 'shallows_icy';
      break;
    case 204:
      filename = 'shallows_muddy';
      break;
    case 205:
      filename = 'shallows_plant';
      break;
    case 206:
      filename = 'shallows_sand';
      break;
    case 251:
      filename = 'water_coastal';
      break;
    case 252:
      filename = 'water_emerald';
      break;
    case 253:
      filename = 'water_freshwater';
      break;
    case 254:
      filename = 'water_glacial';
      break;
    case 301:
      filename = 'water_abyssal';
      break;
    case 302:
      filename = 'water_arctic';
      break;
    case 303:
      filename = 'water_deep_ocean1';
      break;
    case 304:
      filename = 'water_deep_ocean2';
      break;
    case 305:
      filename = 'water_deep_ocean3';
      break;
    case 306:
      filename = 'water_stormy';
      break;
    default:
      err('Trying to get terrain image from unknown terrain_id', terrain_id);
      return 'default';
  }
  return `risq/terrains/${filename}`;
}

/** Image path of the fog-of-war overlay texture, hex-cut to match a space's shape */
export const FOG_OVERLAY_IMAGE = 'risq/terrains/fog';
