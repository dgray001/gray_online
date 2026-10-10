import type { BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import type { Point2D } from '../../../../util/objects2d';
import type { RisqBuilding, RisqSpace, RisqUnit, UnitByTypeData } from '../../model/types';
import { RisqUnitType } from '../../model/types';
import { groupUnitsByType } from '../../model/unit_groups';
import type { DwgRisq } from '../../risq';
import type { RisqSpaceHexagon } from './content/space_hexagon';
import type { RisqSpaceView } from './content/space_view';
import type { RisqStatsView } from './content/stats_view';
import { garrisonedUnits } from './content/unit_views';
import type { RisqLeftPanel } from './left_panel';
import type { HoveredObject, LeftPanelData } from './left_panel_data';
import { HoverableObjectType, LeftPanelDataType } from './left_panel_data';
import { unitResolver } from './selection_queries';

/** Hover and click handling for the panel's content: hex zones, unit tiles, zone rows, and stats rows */
export class RisqLeftPanelInput {
  private hovered_object?: HoveredObject;
  private group_tiles: { units: UnitByTypeData[]; p: Point2D; s: number }[] = [];
  private building_tiles: { buildings: RisqBuilding[]; p: Point2D; s: number }[] = [];
  private hovered_group = -1;
  private pressed_group = -1;

  beginGroupDraw(): void {
    this.group_tiles = [];
    this.building_tiles = [];
  }

  buildingTile(buildings: RisqBuilding[], p: Point2D, s: number): string {
    const index = this.building_tiles.push({ buildings, p, s }) - 1;
    return index === this.hovered_group
      ? this.pressed_group === index
        ? 'rgba(250, 250, 250, 0.4)'
        : 'rgba(220, 220, 220, 0.2)'
      : 'transparent';
  }

  groupTile(units: UnitByTypeData[], p: Point2D, s: number): string {
    const index = this.group_tiles.push({ units, p, s }) - 1;
    return index === this.hovered_group
      ? this.pressed_group === index
        ? 'rgba(250, 250, 250, 0.4)'
        : 'rgba(220, 220, 220, 0.2)'
      : 'transparent';
  }

  constructor(
    private risq: DwgRisq,
    private panel: RisqLeftPanel,
    private hexagon: RisqSpaceHexagon,
    private space_view: RisqSpaceView,
    private stats: RisqStatsView
  ) {}

  /** Clears the hovered object's flags; the next draw lays out fresh hover rects */
  reset(): void {
    if (this.hovered_object) {
      this.hovered_object.object.hover_data.hovered = false;
      this.hovered_object.object.hover_data.clicked = false;
    }
    this.hovered_object = undefined;
    this.group_tiles = [];
    this.building_tiles = [];
    this.hovered_group = -1;
    this.pressed_group = -1;
  }

  cancelInput(): void {
    this.reset();
    this.hexagon.clearHoveredZone();
    this.space_view.cancelInput();
  }

  private objectHoverLogic(m: Point2D, hovered: HoveredObject): void {
    const object = hovered.object;
    if (
      m.x < object.hover_data.ps.x ||
      m.y < object.hover_data.ps.y ||
      m.x > object.hover_data.pe.x ||
      m.y > object.hover_data.pe.y
    ) {
      object.hover_data.hovered = false;
      return;
    }
    const previous = this.hovered_object;
    if (!!previous && (previous.object.internal_id !== object.internal_id || previous.type !== hovered.type)) {
      previous.object.hover_data.hovered = false;
      previous.object.hover_data.clicked = false;
    }
    object.hover_data.hovered = true;
    this.hovered_object = hovered;
  }

  private hoverUnits(m: Point2D, player_id_units: [number, Iterable<number>][]): void {
    const resolve = unitResolver(this.risq);
    for (const [player_id, unit_ids] of player_id_units) {
      for (const unit_id of unit_ids) {
        const unit = resolve(player_id, unit_id);
        if (unit) {
          this.objectHoverLogic(m, { type: HoverableObjectType.UNIT, object: unit });
        }
      }
    }
  }

  mousemove(data: LeftPanelData | undefined, canvas: Point2D, screen: Point2D, transform: BoardTransformData): void {
    const m = screen;
    if (this.building_tiles.length > 0) {
      this.hovered_group = this.building_tiles.findIndex(
        ({ p, s }: { p: Point2D; s: number }): boolean => m.x >= p.x && m.x <= p.x + s && m.y >= p.y && m.y <= p.y + s
      );
      return;
    }
    if (this.group_tiles.length > 0) {
      this.hovered_group = this.group_tiles.findIndex(
        ({ p, s }: { p: Point2D; s: number }): boolean => m.x >= p.x && m.x <= p.x + s && m.y >= p.y && m.y <= p.y + s
      );
      return;
    }
    switch (data?.data_type) {
      case LeftPanelDataType.SPACE:
        this.hexagon.mousemove(m, data.data);
        this.space_view.mousemove(canvas, screen, transform);
        break;
      case LeftPanelDataType.ZONE: {
        this.hexagon.mousemove(m, data.data.space);
        const zone = data.data.zone;
        if (zone.economic_hover_data) {
          this.objectHoverLogic(m, {
            type: HoverableObjectType.ZONE_UNIT_HEADER,
            object: { internal_id: -1, unit_type: 'economic', zone, hover_data: zone.economic_hover_data },
          });
        }
        if (zone.military_hover_data) {
          this.objectHoverLogic(m, {
            type: HoverableObjectType.ZONE_UNIT_HEADER,
            object: { internal_id: -2, unit_type: 'military', zone, hover_data: zone.military_hover_data },
          });
        }
        for (const unit_id of [...zone.economic_units, ...zone.military_units]) {
          const unit = zone.units.get(unit_id);
          if (unit) {
            this.objectHoverLogic(m, { type: HoverableObjectType.UNIT, object: unit });
          }
        }
        if (zone.resource) {
          this.objectHoverLogic(m, { type: HoverableObjectType.RESOURCE, object: zone.resource });
        }
        if (zone.building) {
          this.objectHoverLogic(m, { type: HoverableObjectType.BUILDING, object: zone.building });
        }
        break;
      }
      case LeftPanelDataType.MULTIPLE_PLAYERS_UNITS:
        this.hoverUnits(
          m,
          data.data.units_by_player.flatMap(([, units]) =>
            units.map((u): [number, Set<number>] => [u.player_id, u.units])
          )
        );
        break;
      case LeftPanelDataType.UNITS_BY_TYPE:
      case LeftPanelDataType.ECONOMIC_UNITS:
      case LeftPanelDataType.MILITARY_UNITS:
        this.hoverUnits(
          m,
          data.data.units.map((u): [number, Set<number>] => [u.player_id, u.units])
        );
        break;
      case LeftPanelDataType.UNIT:
        this.stats.mousemove(m);
        break;
      case LeftPanelDataType.BUILDING:
        this.stats.mousemove(m);
        for (const unit of garrisonedUnits(this.risq, data.data)) {
          this.objectHoverLogic(m, { type: HoverableObjectType.UNIT, object: unit });
        }
        break;
      default:
        break;
    }
  }

  mousedown(data: LeftPanelData | undefined, e: MouseEvent): void {
    if (this.group_tiles.length > 0 || this.building_tiles.length > 0) {
      this.pressed_group = e.button === 0 ? this.hovered_group : -1;
      return;
    }
    const hovered_zone = this.hexagon.hoveredZone();
    switch (data?.data_type) {
      case LeftPanelDataType.SPACE:
      case LeftPanelDataType.ZONE:
        if (!!hovered_zone) {
          hovered_zone.clicked = true;
        } else if (!!this.hovered_object) {
          this.hovered_object.object.hover_data.clicked = true;
        } else if (data.data_type === LeftPanelDataType.SPACE) {
          this.space_view.mousedown(e);
        }
        break;
      case LeftPanelDataType.UNITS:
      case LeftPanelDataType.MULTIPLE_PLAYERS_UNITS:
      case LeftPanelDataType.UNITS_BY_TYPE:
      case LeftPanelDataType.ECONOMIC_UNITS:
      case LeftPanelDataType.MILITARY_UNITS:
      case LeftPanelDataType.BUILDING:
        if (!!this.hovered_object) {
          this.hovered_object.object.hover_data.clicked = true;
        }
        break;
      default:
        break;
    }
  }

  mouseup(data: LeftPanelData | undefined, visibility: number, e: MouseEvent): void {
    if (this.building_tiles.length > 0) {
      const tile = this.pressed_group === this.hovered_group ? this.building_tiles[this.pressed_group] : undefined;
      this.pressed_group = -1;
      if (tile && e.button === 0) {
        const selected = this.risq.selection.selectedBuildings();
        const buildings = e.shiftKey
          ? selected.filter((b: RisqBuilding): boolean => b.building_id === tile.buildings[0].building_id)
          : tile.buildings;
        const ids = new Set(buildings.map((b: RisqBuilding): number => b.internal_id));
        const remaining = e.ctrlKey
          ? selected.filter((b: RisqBuilding): boolean => !ids.has(b.internal_id))
          : buildings;
        this.risq.selection.selectOwnBuildings(remaining.map((b: RisqBuilding): number => b.internal_id));
      }
      return;
    }
    if (this.group_tiles.length > 0) {
      const tile = this.pressed_group === this.hovered_group ? this.group_tiles[this.pressed_group] : undefined;
      this.pressed_group = -1;
      if (tile && e.button === 0) {
        const space = data && 'space' in data.data ? data.data.space : undefined;
        const units = e.ctrlKey
          ? this.removeGroup(data, tile.units, e.shiftKey ? tile.units[0].unit_id : undefined)
          : new Map([[tile.units[0].player_id, tile.units]]);
        this.openUnitGroups(space, units, visibility);
      }
      return;
    }
    switch (data?.data_type) {
      case LeftPanelDataType.SPACE:
      case LeftPanelDataType.ZONE:
        this.worldMouseup(data, visibility, e);
        break;
      case LeftPanelDataType.MULTIPLE_PLAYERS_UNITS: {
        const unit = this.releasedUnit();
        if (unit) {
          this.unitGridClick(data.data.space, data.data.units_by_player, unit, visibility, e);
        }
        break;
      }
      case LeftPanelDataType.UNITS_BY_TYPE:
      case LeftPanelDataType.ECONOMIC_UNITS:
      case LeftPanelDataType.MILITARY_UNITS: {
        const unit = this.releasedUnit();
        if (unit) {
          this.unitGridClick(data.data.space, [[unit.player_id, data.data.units]], unit, visibility, e);
        }
        break;
      }
      case LeftPanelDataType.BUILDING: {
        const unit = this.releasedUnit();
        if (unit) {
          this.garrisonedUnitClick(data.data, unit, visibility, e);
        }
        break;
      }
      default:
        break;
    }
  }

  /** The pressed unit tile if it was released while still hovered; always clears its pressed state */
  private releasedUnit(): RisqUnit | undefined {
    if (this.hovered_object?.type !== HoverableObjectType.UNIT || !this.hovered_object.object.hover_data.clicked) {
      return undefined;
    }
    const unit = this.hovered_object.object;
    unit.hover_data.clicked = false;
    return unit.hover_data.hovered ? unit : undefined;
  }

  /** Clicking a zone in the hex opens it (or its space if it's the zone already shown); clicking a zone row object opens it */
  private worldMouseup(
    data: Extract<LeftPanelData, { data_type: LeftPanelDataType.SPACE | LeftPanelDataType.ZONE }>,
    visibility: number,
    e: MouseEvent
  ): void {
    const space: RisqSpace = data.data_type === LeftPanelDataType.SPACE ? data.data : data.data.space;
    const hovered_zone = this.hexagon.hoveredZone();
    if (!!hovered_zone && hovered_zone.clicked) {
      hovered_zone.clicked = false;
      if (!hovered_zone.hovered) {
        return;
      }
      if (data.data_type === LeftPanelDataType.ZONE && hovered_zone.coordinate_key === data.data.zone.coordinate_key) {
        this.panel.openPanel({ data_type: LeftPanelDataType.SPACE, data: space }, visibility);
      } else {
        this.panel.openPanel({ data_type: LeftPanelDataType.ZONE, data: { space, zone: hovered_zone } }, visibility);
      }
    } else if (!!this.hovered_object && this.hovered_object.object.hover_data.clicked) {
      const hovered = this.hovered_object;
      hovered.object.hover_data.clicked = false;
      if (hovered.object.hover_data.hovered) {
        this.zoneObjectClick(data, space, hovered, visibility, e);
      }
    } else if (data.data_type === LeftPanelDataType.SPACE) {
      this.space_view.mouseup(e);
    }
  }

  private zoneObjectClick(
    data: LeftPanelData,
    space: RisqSpace,
    hovered: HoveredObject,
    visibility: number,
    e: MouseEvent
  ): void {
    switch (hovered.type) {
      case HoverableObjectType.ZONE_UNIT_HEADER: {
        const units_by_player = new Map<number, UnitByTypeData[]>();
        for (const [player_id, units] of hovered.object.zone.units_by_type) {
          const type_units = [...units.values()].filter((u) =>
            hovered.object.unit_type === 'economic'
              ? u.unit_type === RisqUnitType.ECONOMIC
              : u.unit_type !== RisqUnitType.ECONOMIC
          );
          if (type_units.length > 0) {
            units_by_player.set(player_id, type_units);
          }
        }
        this.panel.openPanel({ data_type: LeftPanelDataType.UNITS, data: { space, units_by_player } }, visibility);
        break;
      }
      case HoverableObjectType.UNIT:
        if (e.shiftKey && data.data_type === LeftPanelDataType.ZONE) {
          const units_by_type = data.data.zone.units_by_type.get(hovered.object.player_id);
          if (!!units_by_type) {
            const units = [...units_by_type.values()].filter((u) => u.unit_id === hovered.object.unit_id);
            this.panel.openPanel({ data_type: LeftPanelDataType.UNITS_BY_TYPE, data: { space, units } }, visibility);
          }
        } else {
          this.panel.openPanel({ data_type: LeftPanelDataType.UNIT, data: hovered.object }, visibility);
        }
        break;
      case HoverableObjectType.BUILDING:
        this.panel.openPanel({ data_type: LeftPanelDataType.BUILDING, data: hovered.object }, visibility);
        break;
      case HoverableObjectType.RESOURCE:
        this.panel.openPanel({ data_type: LeftPanelDataType.RESOURCE, data: hovered.object }, visibility);
        break;
      default:
        break;
    }
  }

  /** Shift narrows the garrison selection to the clicked unit's type; otherwise selects just the clicked unit */
  private garrisonedUnitClick(building: RisqBuilding, hovered_unit: RisqUnit, visibility: number, e: MouseEvent): void {
    if (e.shiftKey) {
      const units_map = this.risq.getGame()?.players[building.player_id]?.units ?? new Map<number, RisqUnit>();
      const units = groupUnitsByType(units_map, building.garrisoned_units ?? []).filter(
        (u) => u.unit_id === hovered_unit.unit_id
      );
      this.panel.openPanel({ data_type: LeftPanelDataType.UNITS_BY_TYPE, data: { units } }, visibility);
    } else {
      this.panel.openPanel({ data_type: LeftPanelDataType.UNIT, data: hovered_unit }, visibility);
    }
  }

  /** Shift keeps one unit id; Ctrl removes one unit; Ctrl+Shift removes all selected units of that unit id */
  private unitGridClick(
    space: RisqSpace | undefined,
    groups: [number, UnitByTypeData[]][],
    hovered_unit: RisqUnit,
    visibility: number,
    e: MouseEvent
  ): void {
    if (e.shiftKey && !e.ctrlKey) {
      const player_units = groups.find(([player_id]) => player_id === hovered_unit.player_id)?.[1] ?? [];
      const units_by_player = new Map<number, UnitByTypeData[]>([
        [hovered_unit.player_id, player_units.filter((u) => u.unit_id === hovered_unit.unit_id)],
      ]);
      this.openUnitGroups(space, units_by_player, visibility);
    } else if (e.ctrlKey) {
      const units_by_player = new Map<number, UnitByTypeData[]>();
      for (const [player_id, units] of groups) {
        const new_units = units
          .map((u) => ({
            ...u,
            units: new Set(
              [...u.units].filter((id) =>
                e.shiftKey ? u.unit_id !== hovered_unit.unit_id : id !== hovered_unit.internal_id
              )
            ),
          }))
          .filter((u) => u.units.size > 0);
        if (new_units.length > 0) {
          units_by_player.set(player_id, new_units);
        }
      }
      this.openUnitGroups(space, units_by_player, visibility);
    } else {
      this.panel.openPanel({ data_type: LeftPanelDataType.UNIT, data: hovered_unit }, visibility);
    }
  }

  private removeGroup(
    data: LeftPanelData | undefined,
    removed: UnitByTypeData[],
    unit_id?: number
  ): Map<number, UnitByTypeData[]> {
    const groups: [number, UnitByTypeData[]][] =
      data?.data_type === LeftPanelDataType.MULTIPLE_PLAYERS_UNITS
        ? data.data.units_by_player
        : data?.data_type === LeftPanelDataType.UNITS_BY_TYPE ||
            data?.data_type === LeftPanelDataType.ECONOMIC_UNITS ||
            data?.data_type === LeftPanelDataType.MILITARY_UNITS
          ? [[data.data.units[0].player_id, data.data.units]]
          : [];
    const result = new Map<number, UnitByTypeData[]>();
    for (const [player_id, units] of groups) {
      const removed_ids = new Set(
        (unit_id === undefined
          ? removed.filter((u) => u.player_id === player_id)
          : units.filter((u) => u.unit_id === unit_id)
        ).flatMap((u: UnitByTypeData): number[] => [...u.units])
      );
      const remaining = units
        .map(
          (u: UnitByTypeData): UnitByTypeData => ({
            ...u,
            units: new Set([...u.units].filter((id: number): boolean => !removed_ids.has(id))),
          })
        )
        .filter((u: UnitByTypeData): boolean => u.units.size > 0);
      if (remaining.length > 0) {
        result.set(player_id, remaining);
      }
    }
    return result;
  }

  private openUnitGroups(
    space: RisqSpace | undefined,
    units_by_player: Map<number, UnitByTypeData[]>,
    visibility: number
  ): void {
    if (space) {
      this.panel.openPanel({ data_type: LeftPanelDataType.UNITS, data: { space, units_by_player } }, visibility);
    } else {
      const units = [...units_by_player.values()].flat();
      this.panel.openPanel({ data_type: LeftPanelDataType.UNITS_BY_TYPE, data: { units } }, visibility);
    }
  }
}
