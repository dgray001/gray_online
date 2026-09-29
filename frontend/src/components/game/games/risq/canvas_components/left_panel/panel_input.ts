import type { BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import type { Point2D } from '../../../../util/objects2d';
import type { RisqBuilding, RisqSpace, RisqUnit, UnitByTypeData } from '../../model/types';
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

  constructor(
    private risq: DwgRisq,
    private panel: RisqLeftPanel,
    private hexagon: RisqSpaceHexagon,
    private space_view: RisqSpaceView,
    private stats: RisqStatsView
  ) {}

  /** Clears the hovered object's flags; the next draw lays out fresh hover rects */
  reset() {
    if (this.hovered_object) {
      this.hovered_object.object.hover_data.hovered = false;
      this.hovered_object.object.hover_data.clicked = false;
    }
    this.hovered_object = undefined;
  }

  private objectHoverLogic(m: Point2D, hovered: HoveredObject) {
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

  private hoverUnits(m: Point2D, player_id_units: [number, Iterable<number>][]) {
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

  mousemove(data: LeftPanelData | undefined, canvas: Point2D, screen: Point2D, transform: BoardTransformData) {
    const m = screen;
    switch (data?.data_type) {
      case LeftPanelDataType.SPACE:
        this.hexagon.mousemove(m, data.data);
        this.space_view.mousemove(canvas, screen, transform);
        break;
      case LeftPanelDataType.ZONE: {
        this.hexagon.mousemove(m, data.data.space);
        const zone = data.data.zone;
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

  mousedown(data: LeftPanelData | undefined, e: MouseEvent) {
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

  mouseup(data: LeftPanelData | undefined, visibility: number, e: MouseEvent) {
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
  ) {
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
  ) {
    switch (hovered.type) {
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
  private garrisonedUnitClick(building: RisqBuilding, hovered_unit: RisqUnit, visibility: number, e: MouseEvent) {
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

  /** Shift keeps only the clicked unit's type; Ctrl removes the clicked unit; a plain click selects just that unit */
  private unitGridClick(
    space: RisqSpace | undefined,
    groups: [number, UnitByTypeData[]][],
    hovered_unit: RisqUnit,
    visibility: number,
    e: MouseEvent
  ) {
    if (e.shiftKey) {
      const player_units = groups.find(([player_id]) => player_id === hovered_unit.player_id)?.[1] ?? [];
      const units_by_player = new Map<number, UnitByTypeData[]>([
        [hovered_unit.player_id, player_units.filter((u) => u.unit_id === hovered_unit.unit_id)],
      ]);
      this.openUnitGroups(space, units_by_player, visibility);
    } else if (e.ctrlKey) {
      const units_by_player = new Map<number, UnitByTypeData[]>();
      for (const [player_id, units] of groups) {
        const new_units = units
          .map((u) => ({ ...u, units: new Set([...u.units].filter((id) => id !== hovered_unit.internal_id)) }))
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

  private openUnitGroups(
    space: RisqSpace | undefined,
    units_by_player: Map<number, UnitByTypeData[]>,
    visibility: number
  ) {
    if (space) {
      this.panel.openPanel({ data_type: LeftPanelDataType.UNITS, data: { space, units_by_player } }, visibility);
    } else {
      const units = [...units_by_player.values()].flat();
      this.panel.openPanel({ data_type: LeftPanelDataType.UNITS_BY_TYPE, data: { units } }, visibility);
    }
  }
}
