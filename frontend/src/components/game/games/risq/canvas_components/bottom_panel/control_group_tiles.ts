import type { RisqUnit } from '../../model/types';
import { groupUnitsByType } from '../../model/unit_groups';
import { unitDisplayBlocks } from '../../model/unit_display';

export interface ControlGroupTile {
  unit: RisqUnit;
  count: number;
  ids: number[];
}

export function controlGroupTiles(units: Map<number, RisqUnit>, ids: number[], capacity: number): ControlGroupTile[] {
  const groups = groupUnitsByType(units, ids);
  return unitDisplayBlocks(groups.length ? [[groups[0].player_id, groups]] : [], capacity).map((block) => ({
    unit: units.get([...block.units[0].units][0])!,
    count: block.count,
    ids: block.units.flatMap((group) => [...group.units]),
  }));
}
