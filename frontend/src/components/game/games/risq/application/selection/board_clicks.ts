import { canvasToScreen } from '../../../../util/canvas_board/canvas_board';
import type { Point2D } from '../../../../util/objects2d';
import { pointInRect } from '../../../../util/objects2d';
import type { RisqLeftPanel } from '../../canvas_components/left_panel/left_panel';
import { LeftPanelDataType } from '../../canvas_components/left_panel/left_panel_data';
import { RisqVisibilityLevel } from '../../model/types';
import type {
  RisqBuilding,
  RisqPlayer,
  RisqResource,
  RisqSpace,
  RisqUnit,
  RisqZone,
  UnitByTypeData,
} from '../../model/types';
import { groupUnitsByType, organizeZoneUnits } from '../../model/unit_groups';
import type { RisqViewport } from '../../rendering/board/viewport';
import { unitVisibleInViewMode } from '../../rendering/terrain';

import { visibleZoneUnitSlot } from '../../rendering/zones/hit_testing';
import { zoneCenterOffset } from '../../rendering/zones/geometry';
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
  private click_selection: UnitByTypeData[] = [];
  constructor(
    private session: RisqSession,
    private viewport: RisqViewport,
    private selection: RisqSelection,
    private left_panel: RisqLeftPanel,
    private planning: RisqOrderPlanning
  ) {}

  /** A released click on the zone: acts on whichever pressed part is still hovered, else opens the zone itself */
  zoneClick(space: RisqSpace, zone: RisqZone, e: MouseEvent): void {
    if (e.detail === 1) {
      this.click_selection = this.selection.unitTypeGroups();
    }
    for (const [i, part] of zone.hovered_data.entries()) {
      if (!part.clicked || !part.hovered) {
        continue;
      }
      if (i > 0 && visibleZoneUnitSlot(zone, i - 1, this.viewport.viewMode()).length === 0) {
        continue;
      }
      if (i === 0) {
        this.buildingSlotClick(space, zone, e.ctrlKey);
      } else if (e.detail >= 2) {
        this.unitSlotMultiClick(space, zone, i - 1, e.detail >= 3, e.ctrlKey);
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
  private buildingSlotClick(space: RisqSpace, zone: RisqZone, additive: boolean): void {
    if (additive && !zone.resource && zone.building?.player_id === this.session.getPlayerId()) {
      const ids = new Set(this.selection.selectedBuildings().map((b: RisqBuilding): number => b.internal_id));
      if (ids.has(zone.building.internal_id)) {
        ids.delete(zone.building.internal_id);
      } else {
        ids.add(zone.building.internal_id);
      }
      this.selection.selectOwnBuildings([...ids]);
      return;
    }
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

  private foundationClick(space: RisqSpace, zone: RisqZone): void {
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
  private unitSlotMultiClick(
    space: RisqSpace,
    zone: RisqZone,
    slot_index: number,
    on_screen: boolean,
    additive: boolean
  ): void {
    const target = visibleZoneUnitSlot(zone, slot_index, this.viewport.viewMode())[0];
    if (!target) {
      return;
    }
    const ids = on_screen
      ? this.unitsOnScreenOfType(target.player_id, target.unit_id)
      : [...zone.units.values()]
          .filter((u) => u.player_id === target.player_id && u.unit_id === target.unit_id)
          .map((u) => u.internal_id);
    if (additive) {
      const units = this.session
        .getGame()!
        .players.find((player: RisqPlayer): boolean => player.player.player_id === target.player_id)!.units;
      this.toggleUnitSlot(space, [...this.click_selection, ...groupUnitsByType(units, ids)], false);
      return;
    }
    this.selection.selectUnitsOfPlayer(target.player_id, ids, on_screen ? undefined : space, space.visibility);
  }

  /** Ctrl-click adds the whole slot to the selection, or removes it if every unit in it is already selected */
  private toggleUnitSlot(space: RisqSpace, slot_groups: UnitByTypeData[], toggle: boolean = true): void {
    const slot_unit_ids = slot_groups.flatMap((t) => [...t.units]);
    const selected = this.selection.selectedUnitIds();
    const deselect = toggle && slot_unit_ids.length > 0 && slot_unit_ids.every((id) => selected.has(id));
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
  private unitSlotClick(space: RisqSpace, slot_groups: UnitByTypeData[]): void {
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
  spaceClick(space: RisqSpace): void {
    if (!space.clicked || space.visibility <= 0) {
      return;
    }
    if (this.viewport.zoneView()) {
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

  /** Selects own ungarrisoned units in the rectangle, falling back to buildings when no units match */
  dragSelect(start: Point2D, end: Point2D, additive: boolean): void {
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
      const buildings = [...player.buildings.values()].filter((building: RisqBuilding): boolean => {
        const anchor = canvasToScreen(
          this.viewport.orderPoint(
            building.space_coordinate,
            zoneCenterOffset(building.zone_coordinate, this.viewport.hexR()),
            zone_view
          ),
          this.viewport.lastTransform()
        );
        return pointInRect(anchor, start, end);
      });
      if (buildings.length > 0) {
        this.selection.selectOwnBuildings([
          ...new Set([
            ...(additive ? this.selection.selectedBuildings().map((b: RisqBuilding): number => b.internal_id) : []),
            ...buildings.map((b: RisqBuilding): number => b.internal_id),
          ]),
        ]);
      } else {
        this.dragSelectResources(start, end, additive, zone_view);
      }
      return;
    }
    const selected_ids = additive ? new Set([...this.selection.selectedUnitIds(), ...found_ids]) : new Set(found_ids);
    this.selection.selectOwnUnits([...selected_ids]);
  }

  private dragSelectResources(start: Point2D, end: Point2D, additive: boolean, zone_view: boolean): void {
    if (additive) {
      return;
    }
    const center = { x: (start.x + end.x) / 2, y: (start.y + end.y) / 2 };
    const resources = this.resourcesInRect(start, end, zone_view, center);
    if (resources.length > 0) {
      resources.sort((a, b): number => a.dist - b.dist);
      const target = resources[0];
      this.left_panel.openPanel({ data_type: LeftPanelDataType.RESOURCE, data: target.resource }, target.visibility);
    } else {
      this.dragSelectEnemies(start, end, zone_view);
    }
  }

  private dragSelectEnemies(start: Point2D, end: Point2D, zone_view: boolean): void {
    const units = new Map<number, RisqUnit>();
    const buildings: RisqBuilding[] = [];
    const inside = (entity: RisqUnit | RisqBuilding, offset: Point2D | undefined): boolean =>
      entity.player_id !== this.session.getPlayerId() &&
      pointInRect(
        canvasToScreen(
          this.viewport.orderPoint(entity.space_coordinate, offset, zone_view),
          this.viewport.lastTransform()
        ),
        start,
        end
      );
    for (const space of this.session.getGame()?.spaces.flat() ?? []) {
      if (!space || space.visibility < RisqVisibilityLevel.GOOD) {
        continue;
      }
      for (const unit of space.units?.values() ?? []) {
        if (
          unit.garrisoned_in === undefined &&
          unitVisibleInViewMode(unit.unit_type, this.viewport.viewMode()) &&
          inside(unit, this.viewport.unitAnchorOffset(unit))
        ) {
          units.set(unit.internal_id, unit);
        }
      }
      for (const building of space.buildings?.values() ?? []) {
        if (inside(building, zoneCenterOffset(building.zone_coordinate, this.viewport.hexR()))) {
          buildings.push(building);
        }
      }
    }
    if (units.size > 0) {
      const first = units.values().next().value!;
      const space = this.session.spaceAt(first.space_coordinate)!;
      const units_by_player = new Map(
        [...organizeZoneUnits(units)].map(([id, groups]): [number, UnitByTypeData[]] => [id, [...groups.values()]])
      );
      this.left_panel.openPanel(
        { data_type: LeftPanelDataType.UNITS, data: { space, units_by_player } },
        space.visibility
      );
    } else if (buildings.length > 0) {
      this.left_panel.openPanel(
        buildings.length === 1
          ? { data_type: LeftPanelDataType.BUILDING, data: buildings[0] }
          : { data_type: LeftPanelDataType.BUILDINGS, data: { buildings } },
        RisqVisibilityLevel.GOOD
      );
    } else {
      this.left_panel.close();
    }
  }

  private resourcesInRect(
    start: Point2D,
    end: Point2D,
    zone_view: boolean,
    center: Point2D
  ): { resource: RisqResource; visibility: RisqVisibilityLevel; dist: number }[] {
    const bounds = this.viewport.visibleCanvasBounds();
    return (this.session.getGame()?.spaces ?? []).flat().flatMap((s) => {
      if (!s || s.visibility < RisqVisibilityLevel.FOG || !this.viewport.isSpaceOnScreen(s, bounds)) {
        return [];
      }
      return [...(s.resources?.values() ?? [])].flatMap((r) => {
        const a = this.resourceAnchor(r, zone_view);
        return pointInRect(a, start, end)
          ? [{ resource: r, visibility: s.visibility, dist: (a.x - center.x) ** 2 + (a.y - center.y) ** 2 }]
          : [];
      });
    });
  }

  private resourceAnchor(resource: RisqResource, zone_view: boolean): Point2D {
    const offset = zoneCenterOffset(resource.zone_coordinate, this.viewport.hexR());
    return canvasToScreen(
      this.viewport.orderPoint(resource.space_coordinate, offset, zone_view),
      this.viewport.lastTransform()
    );
  }

  // space-level bounding-box check first (cheap pre-filter), then whether the unit's own rendered circle overlaps the viewport
  private unitsOnScreenOfType(player_id: number, unit_id: number): number[] {
    const ids: number[] = [];
    const bounds = this.viewport.visibleCanvasBounds();
    const radius = this.viewport.unitRadius();
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
