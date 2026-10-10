import type { RisqLeftPanel } from '../../canvas_components/left_panel/left_panel';
import type { Point2D } from '../../../../util/objects2d';
import { LeftPanelDataType } from '../../canvas_components/left_panel/left_panel_data';
import type { RisqSession } from '../session';
import type { RisqSelection } from './selection';

export interface ControlGroup {
  kind: 'unit' | 'building';
  ids: number[];
}

/** Control groups 1-10 ('0' is group 10), pruned of dead members when recalled */
export class RisqControlGroups {
  private groups = new Map<number, ControlGroup>();
  private group_revision = 0;

  get revision(): number {
    return this.group_revision;
  }

  memberships(kind: ControlGroup['kind'], ids: Iterable<number>): number[] {
    const members = new Set(ids);
    return [...this.groups.entries()]
      .filter(([, group]) => group.kind === kind && group.ids.some((id) => members.has(id)))
      .map(([number]) => number)
      .sort((a, b) => a - b);
  }

  unitIds(number: number, unit_id: number): number[] {
    const group = this.groups.get(number);
    const units = this.session.getPlayer()?.units;
    return group?.kind === 'unit' ? group.ids.filter((id) => units?.get(id)?.unit_id === unit_id) : [];
  }

  removeGroup(number: number): void {
    if (this.groups.delete(number)) {
      this.group_revision++;
    }
  }

  removeMembers(number: number, internal_ids: Iterable<number>): void {
    const removed = new Set(internal_ids);
    const group = this.groups.get(number);
    if (!group?.ids.some((id) => removed.has(id))) {
      return;
    }
    const live_ids = this.entries().find(([candidate]) => candidate === number)?.[1].ids ?? [];
    group.ids = live_ids.filter((id) => !removed.has(id));
    if (group.ids.length === 0) {
      this.groups.delete(number);
    }
    this.group_revision++;
  }

  entries(): [number, ControlGroup][] {
    const player = this.session.getPlayer();
    return [...this.groups.entries()]
      .flatMap(([number, group]) => {
        const ids = group.ids.filter((id) =>
          group.kind === 'unit' ? player?.units.has(id) : player?.buildings.has(id)
        );
        return ids.length ? [[number, { kind: group.kind, ids }] as [number, ControlGroup]] : [];
      })
      .sort(([a], [b]) => a - b);
  }

  constructor(
    private session: RisqSession,
    private left_panel: RisqLeftPanel,
    private selection: RisqSelection
  ) {}

  assign(group: number): void {
    if (!this.left_panel.isOrderable()) {
      return;
    }
    const data = this.left_panel.getData();
    switch (data?.data_type) {
      case LeftPanelDataType.UNIT:
        this.assignGroup(group, { kind: 'unit', ids: [data.data.internal_id] });
        break;
      case LeftPanelDataType.BUILDING:
        this.assignGroup(group, { kind: 'building', ids: [data.data.internal_id] });
        break;
      case LeftPanelDataType.BUILDINGS:
        this.assignGroup(group, { kind: 'building', ids: this.selection.selectedBuildingIds() });
        break;
      case LeftPanelDataType.UNITS_BY_TYPE:
      case LeftPanelDataType.ECONOMIC_UNITS:
      case LeftPanelDataType.MILITARY_UNITS:
        this.assignGroup(group, { kind: 'unit', ids: data.data.units.flatMap((u) => [...u.units]) });
        break;
      default:
        break;
    }
  }

  private assignGroup(number: number, assigned: ControlGroup): void {
    const ids = new Set(assigned.ids);
    for (const [previous, group] of this.entries()) {
      if (
        previous !== number &&
        group.kind === assigned.kind &&
        group.ids.length === ids.size &&
        group.ids.every((id) => ids.has(id))
      ) {
        this.groups.delete(previous);
      }
    }
    this.groups.set(number, assigned);
    this.group_revision++;
  }

  recall(group: number): Point2D | undefined {
    const group_data = this.groups.get(group);
    const player = this.session.getPlayer();
    if (!group_data || !player) {
      return;
    }
    if (group_data.kind === 'building') {
      const ids = group_data.ids.filter((id: number): boolean => player.buildings.has(id));
      if (ids.length === 0) {
        this.groups.delete(group);
        return;
      }
      group_data.ids = ids;
      this.selection.selectOwnBuildings(ids);
      return this.selection.coordinate();
    }
    const alive_ids = group_data.ids.filter((id) => player.units.has(id));
    if (alive_ids.length === 0) {
      this.groups.delete(group);
      return;
    }
    group_data.ids = alive_ids;
    this.selection.selectOwnUnits(alive_ids);
    const coordinates = alive_ids.flatMap((id) => {
      const location = this.session.unitLocation(player.units.get(id)!);
      return location ? [location.space_coordinate] : [];
    });
    if (coordinates.length === 0) {
      return;
    }
    return {
      x: coordinates.reduce((sum, coordinate) => sum + coordinate.x, 0) / coordinates.length,
      y: coordinates.reduce((sum, coordinate) => sum + coordinate.y, 0) / coordinates.length,
    };
  }
}
