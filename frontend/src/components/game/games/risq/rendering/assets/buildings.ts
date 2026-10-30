import { err } from '../../../../../../scripts/log';
import type { RisqCombatStats } from '../../model/types';

export function damageLevel(combat_stats: RisqCombatStats): 0 | 1 | 2 {
  const health_ratio = combat_stats.health / combat_stats.max_health;
  return health_ratio < 0.2 ? 2 : health_ratio < 0.6 ? 1 : 0;
}

/** Returns image path of the building; pass plain to get the uncolored source (no player-color marker pixels) */
export function buildingImage(
  building_id: number | undefined,
  under_construction?: boolean,
  plain = false,
  combat_stats?: RisqCombatStats
): string {
  if (building_id !== undefined && under_construction) {
    return 'risq/buildings/construction';
  }
  let filename = 'empty_plot';
  if (building_id !== undefined) {
    switch (building_id) {
      case 1:
        filename = 'village_center';
        break;
      case 2:
        filename = 'housing';
        break;
      case 21:
        filename = 'outpost';
        break;
      case 23:
        filename = 'redoubt';
        break;
      case 11:
        filename = 'blacksmith';
        break;
      case 22:
        filename = 'barracks';
        break;
      case 3:
        filename = 'farm';
        break;
      case 24:
        filename = 'wall';
        break;
      default:
        err('Trying to get building image from unknown building id', building_id);
        return 'default';
    }
    if (combat_stats && damageLevel(combat_stats) > 0) {
      filename += `_damaged${damageLevel(combat_stats)}`;
    }
    if (plain && filename !== 'empty_plot') {
      filename += '_plain';
    }
  }
  return `risq/buildings/${filename}`;
}

export function rubbleImage(building_id: number, turns: number): string {
  return `${buildingImage(building_id)}_rubble${turns}`;
}

export function techImage(tech_id: number): string {
  switch (tech_id) {
    case 1:
      return 'risq/techs/farm';
    case 2:
      return 'risq/techs/bow_saw';
    case 3:
      return 'risq/techs/prospecting';
    case 11:
      return 'risq/techs/fertilizer';
    case 61:
      return 'risq/techs/infantry_attack';
    case 62:
      return 'risq/techs/infantry_armor';
    case 63:
      return 'risq/techs/pikes';
    case 64:
      return 'risq/techs/ballistics';
    case 91:
      return 'risq/techs/bivouacs';
    case 92:
      return 'risq/techs/escalades';
    case 101:
      return 'risq/techs/mercenary_contracts';
    case 102:
      return 'risq/techs/shrapnel';
    default:
      return 'icons/research64';
  }
}
