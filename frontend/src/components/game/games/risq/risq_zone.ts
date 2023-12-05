import {RisqZone, UnitByTypeData} from "./risq_data";

/** Organizes units by unit id for easier processing */
export function organizeZoneUnits(zone: RisqZone) {
  zone.units_by_type = new Map<number, UnitByTypeData>();
  for (const unit of zone.units.values()) {
    if (zone.units_by_type.has(unit.unit_id)) {
      zone.units_by_type.get(unit.unit_id).units.add(unit.internal_id);
    } else {
      zone.units_by_type.set(unit.unit_id, {
        unit_id: unit.unit_id,
        units: new Set<number>([unit.internal_id]),
      });
    }
  }
}
