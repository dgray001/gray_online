import type { RisqBuilding, RisqCost, RisqPlayer, UnitByTypeData } from './types';
import { RisqProducibleKind, RisqRange, RisqResourceType } from './types';

export interface UnitDisplayBlock {
  player_id: number;
  unit_id?: number;
  units: UnitByTypeData[];
  count: number;
}

export function unitDisplayBlocks(groups: [number, UnitByTypeData[]][], capacity: number): UnitDisplayBlock[] {
  const total = groups.reduce((sum, [, units]) => sum + units.reduce((count, u) => count + u.units.size, 0), 0);
  if (total <= capacity) {
    return groups.flatMap(([player_id, units]) =>
      units.flatMap((u) =>
        [...u.units].map((id) => ({
          player_id,
          unit_id: u.unit_id,
          units: [{ ...u, units: new Set([id]) }],
          count: 1,
        }))
      )
    );
  }
  const ids = groups.flatMap(([player_id, units]) =>
    units
      .filter((u) => u.units.size > 0)
      .map((u) => ({
        player_id,
        unit_id: u.unit_id,
        units: [u],
        count: u.units.size,
      }))
  );
  if (ids.length <= capacity) {
    return ids;
  }
  return categoryBlocks(groups, capacity);
}

function categoryBlocks(groups: [number, UnitByTypeData[]][], capacity: number): UnitDisplayBlock[] {
  const categories = new Map<string, UnitDisplayBlock>();
  for (const [player_id, units] of groups) {
    for (const u of units.filter((u) => u.units.size > 0)) {
      const key = `${player_id}:${u.unit_type}`;
      const block = categories.get(key) ?? { player_id, unit_id: u.unit_id, units: [], count: 0 };
      block.units.push(u);
      block.count += u.units.size;
      categories.set(key, block);
    }
  }
  return categories.size <= capacity || groups.length === 1
    ? [...categories.values()]
    : groups
        .map(([player_id, units]) => ({
          player_id,
          units,
          count: units.reduce((sum, u) => sum + u.units.size, 0),
        }))
        .filter((block) => block.count > 0);
}

export function canHaveGatherPoint(building: RisqBuilding): boolean {
  return building.garrison_capacity > 0 || building.produces.some((p) => p.kind === RisqProducibleKind.UNIT);
}

export function canAffordCost(player: RisqPlayer, cost: RisqCost): boolean {
  const affords = (type: RisqResourceType, amount: number): boolean => {
    if (amount <= 0) {
      return true;
    }
    const pr = player.resources.get(type);
    return (pr ? pr.amount - pr.spending : 0) >= amount;
  };
  return (
    affords(RisqResourceType.FOOD, cost.food) &&
    affords(RisqResourceType.WOOD, cost.wood) &&
    affords(RisqResourceType.STONE, cost.stone) &&
    affords(RisqResourceType.GOLD, cost.gold)
  );
}

export function meetsTechRequirement(player: RisqPlayer, required_tech_id: number): boolean {
  return required_tech_id === 0 || !!player.researched_techs.get(required_tech_id);
}

export function rangeLabel(range: RisqRange): string {
  const labels: Record<RisqRange, string> = {
    [RisqRange.NONE]: 'none',
    [RisqRange.ZONE]: 'zone (melee)',
    [RisqRange.WALL]: 'wall',
    [RisqRange.SPACE]: 'space',
    [RisqRange.TWOZONE]: '2 zones',
    [RisqRange.THREEZONE]: '3 zones',
    [RisqRange.ADJACENT]: '1 space',
    [RisqRange.FIVEZONE]: '5 zones',
    [RisqRange.SECONDARY]: '2 spaces',
  };
  return labels[range];
}

export function rangeDistance(range: RisqRange): number | null {
  switch (range) {
    case RisqRange.WALL:
    case RisqRange.SPACE:
    case RisqRange.TWOZONE:
    case RisqRange.THREEZONE:
      return 0;
    case RisqRange.ADJACENT:
    case RisqRange.FIVEZONE:
      return 1;
    case RisqRange.SECONDARY:
      return 2;
    default:
      return null;
  }
}

export function zoneRangeDistance(range: RisqRange): number {
  if (range >= RisqRange.FIVEZONE) {
    return 5;
  }
  if (range >= RisqRange.THREEZONE) {
    return 3;
  }
  if (range >= RisqRange.TWOZONE) {
    return 2;
  }
  return range >= RisqRange.WALL ? 1 : 0;
}
