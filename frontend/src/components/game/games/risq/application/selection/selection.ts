import type { RisqLeftPanel } from '../../canvas_components/left_panel/left_panel';
import { LeftPanelDataType } from '../../canvas_components/left_panel/left_panel_data';
import type {
  RisqBuilding,
  RisqFrontendOrder,
  RisqRegion,
  RisqSpace,
  RisqUnit,
  UnitByTypeData,
} from '../../model/types';
import { RisqProducibleKind, RisqVisibilityLevel } from '../../model/types';
import { groupUnitsByType } from '../../model/unit_groups';
import type { HotkeySelection } from '../input/hotkeys';
import { isBuildingOrder, isUnitOrder } from '../orders/orders_model';
import type { RisqSession } from '../session';

export type SingleSelectionKind = 'unit' | 'building' | 'resource' | 'foundation';

/** Neutral queries and changes over the current selection, which the left panel's data currently stores */
export class RisqSelection {
  constructor(
    private session: RisqSession,
    private left_panel: RisqLeftPanel
  ) {}

  selectedUnitIds(): Set<number> {
    const data = this.left_panel.getData();
    switch (data?.data_type) {
      case LeftPanelDataType.UNIT:
        return new Set([data.data.internal_id]);
      case LeftPanelDataType.UNITS_BY_TYPE:
      case LeftPanelDataType.ECONOMIC_UNITS:
      case LeftPanelDataType.MILITARY_UNITS:
        return new Set(data.data.units.flatMap((u) => [...u.units]));
      case LeftPanelDataType.MULTIPLE_PLAYERS_UNITS:
        return new Set(data.data.units_by_player.flatMap(([, units]) => units.flatMap((u) => [...u.units])));
      default:
        return new Set();
    }
  }

  /** Selected unit ids when the selection is a single player's units, as hotkeys and commands act on */
  subjectUnitIds(): number[] {
    const data = this.left_panel.getData();
    switch (data?.data_type) {
      case LeftPanelDataType.UNIT:
        return [data.data.internal_id];
      case LeftPanelDataType.UNITS_BY_TYPE:
      case LeftPanelDataType.ECONOMIC_UNITS:
      case LeftPanelDataType.MILITARY_UNITS:
        return data.data.units.flatMap((u) => [...u.units]);
      default:
        return [];
    }
  }

  selectedBuilding(): RisqBuilding | undefined {
    const data = this.left_panel.getData();
    return data?.data_type === LeftPanelDataType.BUILDING ? data.data : undefined;
  }

  current(): { kind: SingleSelectionKind; id: number } | undefined {
    const data = this.left_panel.getData();
    switch (data?.data_type) {
      case LeftPanelDataType.UNIT:
        return { kind: 'unit', id: data.data.internal_id };
      case LeftPanelDataType.BUILDING:
        return { kind: 'building', id: data.data.internal_id };
      case LeftPanelDataType.RESOURCE:
        return { kind: 'resource', id: data.data.internal_id };
      case LeftPanelDataType.FOUNDATION:
        return { kind: 'foundation', id: data.data.coordinate_key };
      case LeftPanelDataType.UNITS_BY_TYPE:
      case LeftPanelDataType.ECONOMIC_UNITS:
      case LeftPanelDataType.MILITARY_UNITS: {
        const ids = this.selectedUnitIds();
        return ids.size === 1 ? { kind: 'unit', id: [...ids][0] } : undefined;
      }
      default:
        return undefined;
    }
  }

  unitTypeGroups(): UnitByTypeData[] {
    const data = this.left_panel.getData();
    switch (data?.data_type) {
      case LeftPanelDataType.UNIT:
        return [
          {
            player_id: data.data.player_id,
            unit_id: data.data.unit_id,
            unit_type: data.data.unit_type,
            units: new Set([data.data.internal_id]),
          },
        ];
      case LeftPanelDataType.UNITS_BY_TYPE:
      case LeftPanelDataType.ECONOMIC_UNITS:
      case LeftPanelDataType.MILITARY_UNITS:
        return data.data.units;
      case LeftPanelDataType.MULTIPLE_PLAYERS_UNITS:
        return data.data.units_by_player.flatMap(([, units]) => units);
      default:
        return [];
    }
  }

  isUnitSelected(internal_id: number): boolean {
    return this.selectedUnitIds().has(internal_id);
  }

  isBuildingSelected(internal_id: number): boolean {
    return this.selectedBuilding()?.internal_id === internal_id;
  }

  isResourceSelected(internal_id: number): boolean {
    const data = this.left_panel.getData();
    return data?.data_type === LeftPanelDataType.RESOURCE && data.data.internal_id === internal_id;
  }

  isRegionSelected(region: RisqRegion): boolean {
    const data = this.left_panel.getData();
    return data?.data_type === LeftPanelDataType.REGION && data.data.name === region.name;
  }

  /** The selection's shape as hotkey scoping sees it, or undefined when nothing hotkey-relevant is selected */
  hotkeySelection(): HotkeySelection | undefined {
    const building = this.selectedBuilding();
    if (building) {
      const ids = (kind: RisqProducibleKind) => building.produces.filter((p) => p.kind === kind).map((p) => p.id);
      return { kind: 'building', unit_ids: ids(RisqProducibleKind.UNIT), tech_ids: ids(RisqProducibleKind.TECH) };
    }
    if (this.subjectUnitIds().length === 0) {
      return undefined;
    }
    const lp = this.left_panel;
    const composition = lp.isOnlyVillagers() ? 'villagers' : lp.isOnlyMilitary() ? 'military' : 'mixed';
    return { kind: 'units', composition, garrisoned: lp.isOnlyGarrisoned() };
  }

  selectBuilding(building: RisqBuilding) {
    this.left_panel.openPanel({ data_type: LeftPanelDataType.BUILDING, data: building }, RisqVisibilityLevel.SPY);
  }

  selectUnit(unit: RisqUnit) {
    this.left_panel.openPanel({ data_type: LeftPanelDataType.UNIT, data: unit }, RisqVisibilityLevel.SPY);
  }

  /** Selects the given units of the local player, grouped by type, regardless of where they are */
  selectOwnUnits(internal_ids: number[]) {
    const player = this.session.getPlayer();
    if (!player) {
      return;
    }
    this.left_panel.openPanel(
      { data_type: LeftPanelDataType.UNITS_BY_TYPE, data: { units: groupUnitsByType(player.units, internal_ids) } },
      RisqVisibilityLevel.SPY
    );
  }

  /** Selects one player's units of the given ids, optionally tied to the space they were picked from */
  selectUnitsOfPlayer(player_id: number, internal_ids: number[], space: RisqSpace | undefined, visibility: number) {
    if (internal_ids.length === 0) {
      return;
    }
    const units_map =
      this.session.getGame()?.players.find((p) => p.player.player_id === player_id)?.units ??
      new Map<number, RisqUnit>();
    const units = groupUnitsByType(units_map, internal_ids);
    this.left_panel.openPanel({ data_type: LeftPanelDataType.UNITS_BY_TYPE, data: { space, units } }, visibility);
  }

  selectOrderSubjects(order: RisqFrontendOrder) {
    const player = this.session.getPlayer();
    if (!player || !this.session.getGame()) {
      return;
    }
    if (isBuildingOrder(order.order_type)) {
      const building = player.buildings.get(order.subjects[0]);
      if (building) {
        this.selectBuilding(building);
      }
      return;
    }
    if (!isUnitOrder(order.order_type)) {
      return;
    }
    const units_by_type = groupUnitsByType(player.units, order.subjects);
    const first = order.subjects.map((id) => player.units.get(id)).find((u) => !!u);
    const location = first ? this.session.unitLocation(first) : undefined;
    if (units_by_type.length === 0 || !location) {
      return;
    }
    const space = this.session.spaceAt(location.space_coordinate);
    if (!space) {
      return;
    }
    this.left_panel.openPanel(
      {
        data_type: LeftPanelDataType.UNITS,
        data: { space, units_by_player: new Map([[player.player.player_id, units_by_type]]) },
      },
      RisqVisibilityLevel.SPY
    );
  }

  /** Selects the next completed building of this type after the currently selected one, returning it */
  cycleBuildingOfType(building_id: number): RisqBuilding | undefined {
    const player = this.session.getPlayer();
    if (!player) {
      return undefined;
    }
    const buildings = [...player.buildings.values()]
      .filter((b) => b.building_id === building_id && !b.under_construction)
      .sort((a, b) => a.internal_id - b.internal_id);
    if (buildings.length === 0) {
      return undefined;
    }
    const current_index = buildings.findIndex((b) => b.internal_id === this.selectedBuilding()?.internal_id);
    const next = buildings[(current_index + 1) % buildings.length];
    this.selectBuilding(next);
    return next;
  }
}
