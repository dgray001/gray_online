import { canvasToScreen } from '../../../../util/canvas_board/canvas_board';
import type { Point2D } from '../../../../util/objects2d';
import { pointInRect } from '../../../../util/objects2d';
import type { RisqLeftPanel } from '../../canvas_components/left_panel/left_panel';
import { LeftPanelDataType } from '../../canvas_components/left_panel/left_panel_data';
import type { RisqSpace, RisqZone, UnitByTypeData } from '../../model/types';
import type { RisqViewport } from '../../rendering/board/viewport';
import { RisqViewMode } from '../../rendering/terrain';
import { UNIT_SLOT_CIRCLE_RADIUS_MULTIPLIER } from '../../rendering/zones/geometry';
import { unitVisibleInViewMode } from '../../rendering/zones/draw';
import { visibleZoneUnitSlot } from '../../rendering/zones/hit_testing';
import type { RisqOrderPlanning } from '../orders/planning';
import type { RisqSession } from '../session';
import type { RisqSelection } from './selection';

function groupByPlayer(groups: Iterable<UnitByTypeData>): Map<number, UnitByTypeData[]> {
  const units_by_player = new Map<number, UnitByTypeData[]>();
  for (const g of groups) {
    if (!units_by_player.has(g.player_id)) {
      units_by_player.set(g.player_id, []);
    }
    units_by_player.get(g.player_id)!.push(g);
  }
  return units_by_player;
}

/** Selection changes from clicking or drag-selecting on the board itself (the left panel has its own gestures) */
export class RisqBoardClicks {
  constructor(
    private session: RisqSession,
    private viewport: RisqViewport,
    private selection: RisqSelection,
    private left_panel: RisqLeftPanel,
    private planning: RisqOrderPlanning
  ) {}

  /** A released click on the zone: acts on whichever pressed part is still hovered, else opens the zone itself */
  zoneClick(space: RisqSpace, zone: RisqZone, e: MouseEvent) {
    for (const [i, part] of zone.hovered_data.entries()) {
      if (!part.clicked || !part.hovered) {
        continue;
      }
      if (i > 0 && visibleZoneUnitSlot(zone, i - 1, this.viewport.viewMode()).length === 0) {
        continue;
      }
      if (i === 0) {
        this.buildingSlotClick(space, zone);
      } else if (e.detail >= 2) {
        this.unitSlotMultiClick(space, zone, i - 1, e.detail >= 3);
      } else if (e.ctrlKey) {
        this.toggleUnitSlot(space, visibleZoneUnitSlot(zone, i - 1, this.viewport.viewMode()));
      } else {
        this.unitSlotClick(space, visibleZoneUnitSlot(zone, i - 1, this.viewport.viewMode()));
      }
      return;
    }
    if (zone.clicked) {
      this.left_panel.openPanel({ data_type: LeftPanelDataType.ZONE, data: { space, zone } }, space.visibility);
    }
  }

  /** Toggles the resource/building/foundation in the zone's building slot */
  private buildingSlotClick(space: RisqSpace, zone: RisqZone) {
    const target_id = zone.resource?.internal_id ?? zone.building?.internal_id;
    const current = this.selection.current();
    if (
      target_id !== undefined &&
      current &&
      (current.kind === 'building' || current.kind === 'resource') &&
      current.id === target_id
    ) {
      this.left_panel.close();
    } else if (!!zone.resource) {
      this.left_panel.openPanel({ data_type: LeftPanelDataType.RESOURCE, data: zone.resource }, space.visibility);
    } else if (!!zone.building) {
      this.left_panel.openPanel({ data_type: LeftPanelDataType.BUILDING, data: zone.building }, space.visibility);
    } else {
      this.foundationClick(space, zone);
    }
  }

  private foundationClick(space: RisqSpace, zone: RisqZone) {
    const local_foundation = this.planning.getLocalFoundation(zone.coordinate_key);
    const foundation = local_foundation ?? this.session.getPlayer()?.planned_foundations?.get(zone.coordinate_key);
    if (!foundation) {
      return;
    }
    const current = this.selection.current();
    if (current && current.kind === 'foundation' && current.id === zone.coordinate_key) {
      this.left_panel.close();
      return;
    }
    this.left_panel.openPanel(
      {
        data_type: LeftPanelDataType.FOUNDATION,
        data: {
          coordinate_key: zone.coordinate_key,
          building_id: foundation.building_id,
          player_id: this.session.getPlayerId(),
          is_local: !!local_foundation,
          display_name: `Planned ${foundation.display_name}`,
          zone,
        },
      },
      space.visibility
    );
  }

  /** Double click selects the slot's lead unit type within the zone; triple click, across the screen */
  private unitSlotMultiClick(space: RisqSpace, zone: RisqZone, slot_index: number, on_screen: boolean) {
    const target = visibleZoneUnitSlot(zone, slot_index, this.viewport.viewMode())[0];
    if (!target) {
      return;
    }
    const ids = on_screen
      ? this.unitsOnScreenOfType(target.player_id, target.unit_id)
      : [...zone.units.values()]
          .filter((u) => u.player_id === target.player_id && u.unit_id === target.unit_id)
          .map((u) => u.internal_id);
    this.selection.selectUnitsOfPlayer(target.player_id, ids, on_screen ? undefined : space, space.visibility);
  }

  /** Ctrl-click adds the whole slot to the selection, or removes it if every unit in it is already selected */
  private toggleUnitSlot(space: RisqSpace, slot_groups: UnitByTypeData[]) {
    const slot_unit_ids = slot_groups.flatMap((t) => [...t.units]);
    const selected = this.selection.selectedUnitIds();
    const deselect = slot_unit_ids.length > 0 && slot_unit_ids.every((id) => selected.has(id));
    const groups = new Map<string, UnitByTypeData>();
    for (const g of this.selection.unitTypeGroups()) {
      groups.set(`${g.player_id}:${g.unit_id}`, { ...g, units: new Set(g.units) });
    }
    for (const g of slot_groups) {
      const key = `${g.player_id}:${g.unit_id}`;
      const target = groups.get(key) ?? { ...g, units: new Set<number>() };
      for (const id of g.units) {
        if (deselect) {
          target.units.delete(id);
        } else {
          target.units.add(id);
        }
      }
      if (target.units.size > 0) {
        groups.set(key, target);
      } else {
        groups.delete(key);
      }
    }
    this.left_panel.openPanel(
      { data_type: LeftPanelDataType.UNITS, data: { space, units_by_player: groupByPlayer(groups.values()) } },
      space.visibility
    );
  }

  /** Selects the slot's units, or deselects when the slot is exactly the one already-selected unit */
  private unitSlotClick(space: RisqSpace, slot_groups: UnitByTypeData[]) {
    const slot_unit_ids = slot_groups.flatMap((t) => [...t.units]);
    const current = this.selection.current();
    if (slot_unit_ids.length === 1 && current?.kind === 'unit' && current.id === slot_unit_ids[0]) {
      this.left_panel.close();
      return;
    }
    this.left_panel.openPanel(
      { data_type: LeftPanelDataType.UNITS, data: { space, units_by_player: groupByPlayer(slot_groups) } },
      space.visibility
    );
  }

  /** A released click on a space when zones aren't interactive: opens its region if regions are, else the space */
  spaceClick(space: RisqSpace) {
    if (!space.clicked || space.visibility <= 0) {
      return;
    }
    if (this.viewport.zoneView() && this.viewport.viewMode() !== RisqViewMode.REGION) {
      return;
    }
    const region = this.viewport.regionInteractionEnabled()
      ? this.session.getRegionForSpace(space.coordinate_key)
      : undefined;
    if (region) {
      this.left_panel.openPanel({ data_type: LeftPanelDataType.REGION, data: region }, space.visibility);
    } else {
      this.left_panel.openPanel({ data_type: LeftPanelDataType.SPACE, data: space }, space.visibility);
    }
  }

  /** Selects own ungarrisoned units whose on-screen anchor lies in the dragged rectangle */
  dragSelect(start: Point2D, end: Point2D, additive: boolean) {
    const player = this.session.getPlayer();
    if (!player) {
      return;
    }
    const zone_view = this.viewport.zoneView();
    const found_ids: number[] = [];
    for (const unit of player.units.values()) {
      if (unit.garrisoned_in !== undefined || !unitVisibleInViewMode(unit.unit_type, this.viewport.viewMode())) {
        continue;
      }
      const anchor = canvasToScreen(
        this.viewport.orderPoint(unit.space_coordinate, this.viewport.unitAnchorOffset(unit), zone_view),
        this.viewport.lastTransform()
      );
      if (pointInRect(anchor, start, end)) {
        found_ids.push(unit.internal_id);
      }
    }
    if (found_ids.length === 0) {
      if (!additive) {
        this.left_panel.close();
      }
      return;
    }
    const selected_ids = additive ? new Set([...this.selection.selectedUnitIds(), ...found_ids]) : new Set(found_ids);
    this.selection.selectOwnUnits([...selected_ids]);
  }

  // space-level bounding-box check first (cheap pre-filter), then whether the unit's own rendered circle overlaps the viewport
  private unitsOnScreenOfType(player_id: number, unit_id: number): number[] {
    const ids: number[] = [];
    const bounds = this.viewport.visibleCanvasBounds();
    const radius = UNIT_SLOT_CIRCLE_RADIUS_MULTIPLIER * this.viewport.hexR();
    for (const row of this.session.getGame()?.spaces ?? []) {
      for (const space of row) {
        if (!space || !this.viewport.isSpaceOnScreen(space, bounds)) {
          continue;
        }
        for (const zone of space.zones?.flat() ?? []) {
          for (const unit of zone.units.values()) {
            if (unit.player_id !== player_id || unit.unit_id !== unit_id) {
              continue;
            }
            const canvas_pos = this.viewport.orderPoint(
              unit.space_coordinate,
              this.viewport.unitAnchorOffset(unit),
              true
            );
            if (this.viewport.isCircleOnScreen(canvas_pos, radius, bounds)) {
              ids.push(unit.internal_id);
            }
          }
        }
      }
    }
    return ids;
  }
}
