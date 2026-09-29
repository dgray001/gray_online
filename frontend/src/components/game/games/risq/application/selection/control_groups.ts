import type { RisqLeftPanel } from '../../canvas_components/left_panel/left_panel';
import { LeftPanelDataType } from '../../canvas_components/left_panel/left_panel_data';
import type { RisqSession } from '../session';
import type { RisqSelection } from './selection';

interface ControlGroup {
  kind: 'unit' | 'building';
  ids: number[];
}

/** Control groups 1-10 ('0' is group 10), pruned of dead members when recalled */
export class RisqControlGroups {
  private groups = new Map<number, ControlGroup>();

  constructor(
    private session: RisqSession,
    private left_panel: RisqLeftPanel,
    private selection: RisqSelection
  ) {}

  assign(group: number) {
    if (!this.left_panel.isOrderable()) {
      return;
    }
    const data = this.left_panel.getData();
    switch (data?.data_type) {
      case LeftPanelDataType.UNIT:
        this.groups.set(group, { kind: 'unit', ids: [data.data.internal_id] });
        break;
      case LeftPanelDataType.BUILDING:
        this.groups.set(group, { kind: 'building', ids: [data.data.internal_id] });
        break;
      case LeftPanelDataType.UNITS_BY_TYPE:
      case LeftPanelDataType.ECONOMIC_UNITS:
      case LeftPanelDataType.MILITARY_UNITS:
        this.groups.set(group, { kind: 'unit', ids: data.data.units.flatMap((u) => [...u.units]) });
        break;
      default:
        break;
    }
  }

  recall(group: number) {
    const group_data = this.groups.get(group);
    const player = this.session.getPlayer();
    if (!group_data || !player) {
      return;
    }
    if (group_data.kind === 'building') {
      const building = player.buildings.get(group_data.ids[0]);
      if (!building) {
        this.groups.delete(group);
        return;
      }
      this.selection.selectBuilding(building);
      return;
    }
    const alive_ids = group_data.ids.filter((id) => player.units.has(id));
    if (alive_ids.length === 0) {
      this.groups.delete(group);
      return;
    }
    group_data.ids = alive_ids;
    this.selection.selectOwnUnits(alive_ids);
  }
}
