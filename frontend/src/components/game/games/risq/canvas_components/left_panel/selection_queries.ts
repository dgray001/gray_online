import { equalsPoint2D } from '../../../../util/objects2d';
import type { RisqSession } from '../../application/session';
import { invertPair } from '../../model/coordinates';
import type { RisqUnit, UnitByTypeData } from '../../model/types';
import { RisqUnitType, RisqVisibilityLevel } from '../../model/types';
import { groupUnitsByType } from '../../model/unit_groups';
import type { DwgRisq } from '../../risq';
import type { LeftPanelData, PlayerUnitsDrawData, UnitsDrawData } from './left_panel_data';
import { LeftPanelDataType } from './left_panel_data';

export type UnitResolver = (player_id: number, internal_id: number) => RisqUnit | undefined;

export function unitResolver(risq: DwgRisq): UnitResolver {
  return (player_id, internal_id) => risq.getGame()?.players[player_id]?.units.get(internal_id);
}

/** Whether the selection belongs to the local player and is visible enough to take orders */
export function isOrderable(
  data: LeftPanelData | undefined,
  visibility: number | undefined,
  player_id: number
): boolean {
  if (!data || !visibility || visibility < RisqVisibilityLevel.GOOD || player_id < 0) {
    return false;
  }
  switch (data.data_type) {
    case LeftPanelDataType.BUILDING:
    case LeftPanelDataType.UNIT:
    case LeftPanelDataType.FOUNDATION:
      return data.data.player_id === player_id;
    case LeftPanelDataType.UNITS_BY_TYPE:
    case LeftPanelDataType.ECONOMIC_UNITS:
    case LeftPanelDataType.MILITARY_UNITS:
      return data.data.units.length > 0 && data.data.units[0].player_id === player_id;
    default:
      return false;
  }
}

export function isUnitSelection(data: LeftPanelData | undefined): boolean {
  switch (data?.data_type) {
    case LeftPanelDataType.MULTIPLE_PLAYERS_UNITS:
    case LeftPanelDataType.UNITS:
    case LeftPanelDataType.UNITS_BY_TYPE:
    case LeftPanelDataType.ECONOMIC_UNITS:
    case LeftPanelDataType.MILITARY_UNITS:
    case LeftPanelDataType.UNIT:
      return true;
    default:
      return false;
  }
}

function isEconomic(t: { unit_type: RisqUnitType }): boolean {
  return t.unit_type === RisqUnitType.ECONOMIC;
}

/** Whether any selected unit (for some=true) or every selected unit (some=false) matches; multi-player selections only count for some */
function unitTypesMatch(
  data: LeftPanelData | undefined,
  matches: (t: { unit_type: RisqUnitType }) => boolean,
  some: boolean
): boolean {
  switch (data?.data_type) {
    case LeftPanelDataType.UNIT:
      return matches(data.data);
    case LeftPanelDataType.MULTIPLE_PLAYERS_UNITS:
      return some && data.data.units_by_player.some(([, units]) => units.some(matches));
    case LeftPanelDataType.UNITS_BY_TYPE:
    case LeftPanelDataType.ECONOMIC_UNITS:
    case LeftPanelDataType.MILITARY_UNITS:
      return some ? data.data.units.some(matches) : data.data.units.every(matches);
    default:
      return false;
  }
}

export function hasVillager(data: LeftPanelData | undefined): boolean {
  return unitTypesMatch(data, isEconomic, true);
}

export function hasMilitary(data: LeftPanelData | undefined): boolean {
  return unitTypesMatch(data, (t) => !isEconomic(t), true);
}

export function isOnlyVillagers(data: LeftPanelData | undefined): boolean {
  return unitTypesMatch(data, isEconomic, false);
}

export function isOnlyMilitary(data: LeftPanelData | undefined): boolean {
  return unitTypesMatch(data, (t) => !isEconomic(t), false);
}

export function isOnlyGarrisoned(data: LeftPanelData | undefined, resolve: UnitResolver): boolean {
  const all_garrisoned = (player_id: number, t: UnitByTypeData): boolean =>
    [...t.units].every((id) => resolve(player_id, id)?.garrisoned_in !== undefined);
  switch (data?.data_type) {
    case LeftPanelDataType.UNIT:
      return data.data.garrisoned_in !== undefined;
    case LeftPanelDataType.MULTIPLE_PLAYERS_UNITS:
      return data.data.units_by_player.every(([player_id, units]) => units.every((t) => all_garrisoned(player_id, t)));
    case LeftPanelDataType.UNITS_BY_TYPE:
    case LeftPanelDataType.ECONOMIC_UNITS:
    case LeftPanelDataType.MILITARY_UNITS:
      return data.data.units.every((t) => all_garrisoned(t.player_id, t));
    default:
      return false;
  }
}

/** The subjects whose orders the panel's order list shows; empty when the selection has none of the local player's */
export function orderListSubjects(
  data: LeftPanelData | undefined,
  own_player_id: number | undefined
): { ids: number[]; kind?: 'unit' | 'building' } {
  switch (data?.data_type) {
    case LeftPanelDataType.UNIT:
    case LeftPanelDataType.BUILDING:
      if (data.data.player_id !== own_player_id) {
        return { ids: [] };
      }
      return { ids: [data.data.internal_id], kind: data.data_type === LeftPanelDataType.UNIT ? 'unit' : 'building' };
    case LeftPanelDataType.UNITS_BY_TYPE:
    case LeftPanelDataType.ECONOMIC_UNITS:
    case LeftPanelDataType.MILITARY_UNITS:
      if (!data.data.units.some((u) => u.player_id === own_player_id)) {
        return { ids: [] };
      }
      return { ids: data.data.units.flatMap((u) => [...u.units]), kind: 'unit' };
    default:
      return { ids: [] };
  }
}

/** Collapses a UNITS selection into its most specific data type; undefined when nothing selectable remains */
export function normalizeSelection(data: LeftPanelData, resolve: UnitResolver): LeftPanelData | undefined {
  switch (data.data_type) {
    case LeftPanelDataType.UNITS_BY_TYPE:
    case LeftPanelDataType.ECONOMIC_UNITS:
    case LeftPanelDataType.MILITARY_UNITS:
      return normalizePlayerUnits(data.data, resolve);
    case LeftPanelDataType.UNITS:
      return normalizeUnits(data.data, resolve);
    default:
      return data;
  }
}

function normalizeUnits(data: UnitsDrawData, resolve: UnitResolver): LeftPanelData | undefined {
  if (!data || !data.space || !data.units_by_player || data.units_by_player.size < 1) {
    return undefined;
  }
  const new_data: [number, UnitByTypeData[]][] = [];
  for (const [player_id, units_by_type] of data.units_by_player.entries()) {
    if (player_id < 0 || units_by_type.length < 1) {
      continue;
    }
    new_data.push([player_id, units_by_type.filter((u) => u.units.size > 0)]);
  }
  if (new_data.length < 1) {
    return undefined;
  }
  if (new_data.length > 1) {
    return {
      data_type: LeftPanelDataType.MULTIPLE_PLAYERS_UNITS,
      data: { space: data.space, units_by_player: new_data },
    };
  }
  return normalizePlayerUnits({ space: data.space, units: new_data[0][1] }, resolve);
}

/** A single unit becomes UNIT; otherwise the type reflects whether the units are economic, military, or mixed */
function normalizePlayerUnits(data: PlayerUnitsDrawData, resolve: UnitResolver): LeftPanelData | undefined {
  if (data.units.length === 1 && data.units[0].units.size === 1) {
    const unit = resolve(data.units[0].player_id, [...data.units[0].units.values()][0]);
    return unit ? { data_type: LeftPanelDataType.UNIT, data: unit } : undefined;
  }
  const new_data = { space: data.space, units: data.units };
  const has_economic_units = data.units.some(isEconomic);
  const has_military_units = data.units.some((u) => !isEconomic(u));
  if (has_economic_units && has_military_units) {
    return { data_type: LeftPanelDataType.UNITS_BY_TYPE, data: new_data };
  } else if (has_economic_units) {
    return { data_type: LeftPanelDataType.ECONOMIC_UNITS, data: new_data };
  } else if (has_military_units) {
    return { data_type: LeftPanelDataType.MILITARY_UNITS, data: new_data };
  }
  return undefined;
}

export function resolveSelectionData(
  data: LeftPanelData,
  session: RisqSession
): { data: LeftPanelData; visibility: number } | undefined {
  switch (data.data_type) {
    case LeftPanelDataType.UNIT: {
      const unit = session.findUnitById(data.data.internal_id);
      if (!unit) {
        return undefined;
      }
      const loc = session.unitLocation(unit);
      const space = loc ? session.spaceAt(loc.space_coordinate) : undefined;
      const vis = unit.player_id === session.getPlayerId() ? RisqVisibilityLevel.GOOD : (space?.visibility ?? 0);
      return vis >= RisqVisibilityLevel.GOOD
        ? { data: { data_type: LeftPanelDataType.UNIT, data: unit }, visibility: vis }
        : undefined;
    }
    case LeftPanelDataType.BUILDING: {
      const building = session.findBuildingById(data.data.internal_id);
      if (!building) {
        return undefined;
      }
      const space = session.spaceAt(building.space_coordinate);
      const vis = building.player_id === session.getPlayerId() ? RisqVisibilityLevel.GOOD : (space?.visibility ?? 0);
      return vis >= RisqVisibilityLevel.GOOD
        ? { data: { data_type: LeftPanelDataType.BUILDING, data: building }, visibility: vis }
        : undefined;
    }
    case LeftPanelDataType.SPACE: {
      const space = session.spaceAt(data.data.coordinate);
      const vis = space?.visibility ?? 0;
      return space && vis >= RisqVisibilityLevel.FOG
        ? { data: { data_type: LeftPanelDataType.SPACE, data: space }, visibility: vis }
        : undefined;
    }
    case LeftPanelDataType.ZONE: {
      const space = session.spaceAt(data.data.space.coordinate);
      const zone = space?.zones?.flat().find((z) => equalsPoint2D(z.coordinate, data.data.zone.coordinate));
      const vis = space?.visibility ?? 0;
      return space && zone && vis >= RisqVisibilityLevel.FOG
        ? { data: { data_type: LeftPanelDataType.ZONE, data: { space, zone } }, visibility: vis }
        : undefined;
    }
    case LeftPanelDataType.RESOURCE: {
      const space = session.spaceAt(data.data.space_coordinate);
      const zone = space?.zones?.flat().find((z) => equalsPoint2D(z.coordinate, data.data.zone_coordinate));
      const vis = space?.visibility ?? 0;
      return zone?.resource && vis >= RisqVisibilityLevel.FOG
        ? { data: { data_type: LeftPanelDataType.RESOURCE, data: zone.resource }, visibility: vis }
        : undefined;
    }
    case LeftPanelDataType.REGION: {
      const region = session.getGame()?.regions.find((r) => r.name === data.data.name);
      return region
        ? { data: { data_type: LeftPanelDataType.REGION, data: region }, visibility: RisqVisibilityLevel.GOOD }
        : undefined;
    }
    case LeftPanelDataType.FOUNDATION: {
      const player = session.getPlayer();
      const coord_key = data.data.coordinate_key;
      const planned = player?.planned_foundations.get(coord_key);
      const space_coord = invertPair(coord_key);
      const space = session.spaceAt(space_coord);
      const zone = space?.zones?.flat().find((z) => z.coordinate_key === coord_key);
      if (planned && space && zone) {
        return {
          data: {
            data_type: LeftPanelDataType.FOUNDATION,
            data: {
              coordinate_key: coord_key,
              building_id: planned.building_id,
              player_id: session.getPlayerId(),
              is_local: data.data.is_local,
              display_name: `Planned ${planned.display_name}`,
              zone,
            },
          },
          visibility: space.visibility,
        };
      }
      if (zone?.building) {
        return {
          data: { data_type: LeftPanelDataType.BUILDING, data: zone.building },
          visibility: space?.visibility ?? RisqVisibilityLevel.GOOD,
        };
      }
      return undefined;
    }
    case LeftPanelDataType.UNITS_BY_TYPE:
    case LeftPanelDataType.ECONOMIC_UNITS:
    case LeftPanelDataType.MILITARY_UNITS: {
      const all_ids = data.data.units.flatMap((u) => [...u.units.values()]);
      const surviving = all_ids.map((id) => session.findUnitById(id)).filter((u): u is RisqUnit => !!u);
      if (surviving.length === 0) {
        return undefined;
      }
      const first = surviving[0];
      const space = data.data.space ? session.spaceAt(data.data.space.coordinate) : undefined;
      const owner = session.getGame()?.players.find((p) => p.player.player_id === first.player_id);
      if (!owner) {
        return undefined;
      }
      const grouped = groupUnitsByType(
        owner.units,
        surviving.map((u) => u.internal_id)
      );
      const normalized = normalizePlayerUnits({ space, units: grouped }, (p, id) =>
        session.getGame()?.players[p]?.units.get(id)
      );
      if (!normalized) {
        return undefined;
      }
      const vis = first.player_id === session.getPlayerId() ? RisqVisibilityLevel.GOOD : (space?.visibility ?? 0);
      return vis >= RisqVisibilityLevel.GOOD ? { data: normalized, visibility: vis } : undefined;
    }
    case LeftPanelDataType.UNITS:
    case LeftPanelDataType.MULTIPLE_PLAYERS_UNITS: {
      const space = session.spaceAt(data.data.space.coordinate);
      if (!space || space.visibility < RisqVisibilityLevel.GOOD) {
        return undefined;
      }
      const surviving_map = new Map<number, UnitByTypeData[]>();
      const raw_entries: [number, UnitByTypeData[]][] =
        data.data_type === LeftPanelDataType.UNITS
          ? [...data.data.units_by_player.entries()]
          : data.data.units_by_player;
      for (const [pid, types] of raw_entries) {
        const ids = types.flatMap((t) => [...t.units.values()]);
        const surviving = ids.map((id) => session.findUnitById(id)).filter((u): u is RisqUnit => !!u);
        const owner = session.getGame()?.players.find((p) => p.player.player_id === pid);
        if (owner && surviving.length > 0) {
          surviving_map.set(
            pid,
            groupUnitsByType(
              owner.units,
              surviving.map((u) => u.internal_id)
            )
          );
        }
      }
      const normalized = normalizeUnits({ space, units_by_player: surviving_map }, (p, id) =>
        session.getGame()?.players[p]?.units.get(id)
      );
      return normalized ? { data: normalized, visibility: space.visibility } : undefined;
    }
    default:
      return undefined;
  }
}
