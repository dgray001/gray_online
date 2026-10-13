import type { RisqBuilding, RisqUnit } from '../../model/types';
import { displayGroups } from '../../model/display_groups';
import { buildingImage } from '../../rendering/assets/buildings';
import { groupUnitsByType } from '../../model/unit_groups';
import { unitDisplayBlocks } from '../../model/unit_display';

export interface ControlGroupTile {
  unit: RisqUnit;
  count: number;
  ids: number[];
}

export function buildingControlGroupTiles(
  buildings: Map<number, RisqBuilding>,
  ids: number[],
  capacity: number
): { image: string; count: number; ids: number[] }[] {
  return displayGroups(
    ids.map((id: number): RisqBuilding => buildings.get(id)!),
    capacity,
    (building: RisqBuilding): string => String(building.building_id)
  ).map((group: RisqBuilding[]): { image: string; count: number; ids: number[] } => ({
    image: buildingImage(group[0].building_id, group[0].under_construction),
    count: group.length,
    ids: group.map((building: RisqBuilding): number => building.internal_id),
  }));
}

export function controlGroupTiles(units: Map<number, RisqUnit>, ids: number[], capacity: number): ControlGroupTile[] {
  const groups = groupUnitsByType(units, ids);
  return unitDisplayBlocks(groups.length ? [[groups[0].player_id, groups]] : [], capacity).map((block) => ({
    unit: units.get([...block.units[0].units][0])!,
    count: block.count,
    ids: block.units.flatMap((group) => [...group.units]),
  }));
}
