import type { RisqUnit, UnitByTypeData } from './types';
import { RisqUnitType } from './types';
/** Organizes units by unit id for easier processing */
export function organizeZoneUnits(units: Map<number, RisqUnit>): Map<number, Map<number, UnitByTypeData>> {
  const units_by_type = new Map<number, Map<number, UnitByTypeData>>();
  for (const unit of units.values()) {
    if (!units_by_type.has(unit.player_id)) {
      units_by_type.set(unit.player_id, new Map<number, UnitByTypeData>());
    }
    const units_by_type_data = units_by_type.get(unit.player_id)?.get(unit.unit_id);
    if (units_by_type_data) {
      units_by_type_data.units.add(unit.internal_id);
    } else {
      units_by_type.get(unit.player_id)!.set(unit.unit_id, {
        unit_id: unit.unit_id,
        unit_type: unit.unit_type,
        player_id: unit.player_id,
        units: new Set<number>([unit.internal_id]),
      });
    }
  }
  return units_by_type;
}

/** Groups the input units by player, keeping only economic or military types */
export function unitsByPlayerFiltered(units: Map<number, RisqUnit>, economic: boolean): Map<number, UnitByTypeData[]> {
  const units_by_type = organizeZoneUnits(units);
  const result = new Map<number, UnitByTypeData[]>();
  for (const [player_id, player_units] of units_by_type.entries()) {
    const filtered = [...player_units.values()].filter((u) => (u.unit_type === RisqUnitType.ECONOMIC) === economic);
    if (filtered.length > 0) {
      result.set(player_id, filtered);
    }
  }
  return result;
}

/** Groups an arbitrary, possibly multi-space list of a single player's unit ids by unit type */
export function groupUnitsByType(units: Map<number, RisqUnit>, ids: number[]): UnitByTypeData[] {
  const by_type = new Map<number, UnitByTypeData>();
  for (const id of ids) {
    const unit = units.get(id);
    if (!unit) {
      continue;
    }
    const existing = by_type.get(unit.unit_id);
    if (existing) {
      existing.units.add(id);
    } else {
      by_type.set(unit.unit_id, {
        player_id: unit.player_id,
        unit_id: unit.unit_id,
        unit_type: unit.unit_type,
        units: new Set([id]),
      });
    }
  }
  return [...by_type.values()];
}
