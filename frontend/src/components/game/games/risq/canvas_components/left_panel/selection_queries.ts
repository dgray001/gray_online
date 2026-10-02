import type { RisqUnit, UnitByTypeData } from '../../model/types';
import { RisqUnitType, RisqVisibilityLevel } from '../../model/types';
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
