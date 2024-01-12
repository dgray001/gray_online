import {RisqZone, UnitByTypeData} from "./risq_data";

/** Organizes units by unit id for easier processing */
export function organizeZoneUnits(zone: RisqZone) {
  zone.units_by_type = new Map<number, UnitByTypeData>();
  let added1 = false;
  let added2 = false;
  for (const unit of zone.units.values()) {
    if (!added1) {
      unit.unit_id += 1;
      added1 = true;
    } else if (!added2) {
      unit.unit_id += 2;
      added2 = true;
    }
    if (zone.units_by_type.has(unit.unit_id)) {
      zone.units_by_type.get(unit.unit_id).units.add(unit.internal_id);
    } else {
      zone.units_by_type.set(unit.unit_id, {
        unit_id: unit.unit_id,
        units: new Set<number>([unit.internal_id]),
      });
      zone.units_by_type.set(unit.unit_id+1, {
        unit_id: unit.unit_id,
        units: new Set<number>([unit.internal_id]),
      });
      zone.units_by_type.set(unit.unit_id+2, {
        unit_id: unit.unit_id,
        units: new Set<number>([unit.internal_id]),
      });
      zone.units_by_type.set(unit.unit_id+3, {
        unit_id: unit.unit_id,
        units: new Set<number>([unit.internal_id]),
      });
    }
  }
}
