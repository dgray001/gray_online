import type { RisqBuilding, RisqPlayer, RisqProducible } from '../../../model/types';
import {
  RisqAttackType,
  RisqOrderType,
  RisqProducibleKind,
  RisqUnitStance,
  meetsTechRequirement,
} from '../../../model/types';
import type { DwgRisq } from '../../../risq';
import { RisqTargetPriorityControl } from '../controls/target_priority_control';
import { ACTION_GRID_COLS } from '../layout';
import type { LeftPanelData } from '../left_panel_data';
import { LeftPanelDataType } from '../left_panel_data';
import { isOnlyMilitary, isOnlyVillagers, unitResolver } from '../selection_queries';
import type { RisqActionButton } from './action_button';
import { RisqBuildingAttackButton, RisqBuildingGatherPointButton } from './building/building_armed_buttons';
import { RisqBuildingDeleteButton } from './building/building_delete_button';
import { RisqBuildingUngarrisonButton } from './building/building_ungarrison_button';
import { RisqCreateButton } from './building/create_button';
import { RisqResearchButton } from './building/research_button';
import { RisqDeleteFoundationButton } from './shared/delete_foundation_button';
import { RisqOrderButton } from './shared/order_button';
import { RisqStopButton } from './shared/stop_button';
import { RisqBuildButton } from './unit/build_button';
import { RisqDeleteButton } from './unit/delete_button';
import { RisqGarrisonButton } from './unit/garrison_button';
import { RisqStanceButton } from './unit/stance_button';
import { RisqUnitToggleButton } from './unit/unit_toggle_button';
import type { UnitToggleField } from './unit/unit_toggle_button';

export declare interface PanelActions {
  buttons: RisqActionButton[];
  target_priority_control?: RisqTargetPriorityControl;
}

const LAST_COL = ACTION_GRID_COLS - 1;

const STANCES: [RisqUnitStance, string, string][] = [
  [RisqUnitStance.AGGRESSIVE, 'icons/swords128', 'Aggressive'],
  [RisqUnitStance.DEFENSIVE, 'icons/shield128', 'Defensive'],
  [RisqUnitStance.STAND_GROUND, 'icons/stand_ground128', 'Stand Ground'],
  [RisqUnitStance.PASSIVE, 'icons/passive128', 'Passive'],
];

const TOGGLES: [UnitToggleField, string, string][] = [
  ['interrupt_current', 'icons/interrupt_current128', 'Interrupt Current'],
  ['attack_back', 'icons/attack_back128', 'Attack Back'],
];

const VILLAGER_ORDERS: [RisqOrderType, string, string][] = [
  [RisqOrderType.OrderType_UnitGather, 'icons/gather128', 'Gather'],
  [RisqOrderType.OrderType_UnitRepair, 'icons/repair128', 'Repair'],
  [RisqOrderType.OrderType_UnitRenew, 'icons/wheat128', 'Renew'],
];

/** Action buttons (and the target-priority control for military) for an orderable selection */
export function buildPanelActions(risq: DwgRisq, data: LeftPanelData | undefined): PanelActions {
  const player = risq.getPlayer();
  switch (data?.data_type) {
    case LeftPanelDataType.UNIT:
    case LeftPanelDataType.UNITS_BY_TYPE:
    case LeftPanelDataType.ECONOMIC_UNITS:
    case LeftPanelDataType.MILITARY_UNITS:
      return unitActions(risq, data, player);
    case LeftPanelDataType.BUILDING:
      return { buttons: buildingActions(risq, data.data, player) };
    case LeftPanelDataType.FOUNDATION:
      if (data.data.player_id !== player?.player.player_id || !risq.session.givingOrders()) {
        return { buttons: [] };
      }
      return {
        buttons: [
          new RisqDeleteFoundationButton(
            {
              row: 0,
              col: LAST_COL,
              image_path: 'icons/skull128',
              description: 'Cancel Foundation',
              foundation_data: data.data,
            },
            risq,
            0
          ),
        ],
      };
    default:
      return { buttons: [] };
  }
}

function unitActions(risq: DwgRisq, data: LeftPanelData, player: RisqPlayer | undefined): PanelActions {
  let unit_internal_ids: number[];
  let builds: RisqProducible[];
  if (data.data_type === LeftPanelDataType.UNIT) {
    unit_internal_ids = [data.data.internal_id];
    builds = data.data.builds;
  } else if (
    data.data_type === LeftPanelDataType.UNITS_BY_TYPE ||
    data.data_type === LeftPanelDataType.ECONOMIC_UNITS ||
    data.data_type === LeftPanelDataType.MILITARY_UNITS
  ) {
    unit_internal_ids = data.data.units.flatMap((u) => [...u.units]);
    const representative_id = data.data.units[0]?.units.values().next().value;
    const representative =
      representative_id === undefined
        ? undefined
        : unitResolver(risq)(player?.player.player_id ?? -1, representative_id);
    builds = representative?.builds ?? [];
  } else {
    return { buttons: [] };
  }
  const buttons: RisqActionButton[] = [
    new RisqOrderButton(
      {
        row: 0,
        col: 0,
        order_type: RisqOrderType.OrderType_UnitMoveSpace,
        image_path: 'icons/move128',
        description: 'Move',
      },
      risq,
      0
    ),
    new RisqOrderButton(
      {
        row: 0,
        col: 1,
        order_type: RisqOrderType.OrderType_UnitAttackSpace,
        image_path: 'icons/sword128',
        description: 'Attack',
      },
      risq,
      0
    ),
    new RisqGarrisonButton({ row: 0, col: 2, unit_internal_ids }, risq, 0),
    new RisqStopButton(
      { row: 0, col: 3, image_path: 'icons/hand_stop128', description: 'Stop', unit_internal_ids },
      risq,
      0
    ),
    new RisqDeleteButton(
      { row: 0, col: LAST_COL, image_path: 'icons/skull128', description: 'Delete', unit_internal_ids },
      risq,
      0
    ),
  ];
  if (isOnlyVillagers(data)) {
    for (const [col, [order_type, image_path, description]] of VILLAGER_ORDERS.entries()) {
      buttons.push(new RisqOrderButton({ row: 1, col, order_type, image_path, description }, risq, 0));
    }
    for (const producible of builds) {
      if (!player || !meetsTechRequirement(player, producible.required_tech_id)) {
        continue;
      }
      buttons.push(new RisqBuildButton({ producible }, risq, 0));
    }
  }
  if (!isOnlyMilitary(data)) {
    return { buttons };
  }
  for (const [col, [stance, image_path, description]] of STANCES.entries()) {
    buttons.push(new RisqStanceButton({ row: 1, col, unit_internal_ids, stance, image_path, description }, risq, 0));
  }
  for (const [col, [field, image_path, description]] of TOGGLES.entries()) {
    buttons.push(new RisqUnitToggleButton({ row: 2, col, unit_internal_ids, field, image_path, description }, risq, 0));
  }
  return { buttons, target_priority_control: new RisqTargetPriorityControl(risq, unit_internal_ids) };
}

function buildingActions(risq: DwgRisq, building: RisqBuilding, player: RisqPlayer | undefined): RisqActionButton[] {
  const building_id = building.internal_id;
  const buttons: RisqActionButton[] = [];
  if (building.combat_stats.attack_type !== RisqAttackType.NONE || building.garrison_capacity > 0) {
    buttons.push(
      new RisqBuildingAttackButton({ row: 0, col: 0, image_path: 'icons/sword128', description: 'Attack' }, risq, 0)
    );
  }
  const can_have_gather_point =
    building.produces.some((p) => p.kind === RisqProducibleKind.UNIT) || building.garrison_capacity > 0;
  if (can_have_gather_point) {
    buttons.push(
      new RisqBuildingGatherPointButton(
        { row: 0, col: 1, image_path: '', description: 'Set/Unset Gather Point', building_id },
        risq,
        0
      )
    );
  }
  if (building.garrison_capacity > 0) {
    buttons.push(
      new RisqBuildingUngarrisonButton(
        { row: 0, col: 2, image_path: 'icons/ungarrison128', description: 'Ungarrison', building_id },
        risq,
        0
      )
    );
  }
  buttons.push(
    new RisqStopButton(
      { row: 0, col: 3, image_path: 'icons/hand_stop128', description: 'Stop', unit_internal_ids: [building_id] },
      risq,
      0
    ),
    new RisqBuildingDeleteButton(
      { row: 0, col: LAST_COL, image_path: 'icons/skull128', description: 'Delete', building_id },
      risq,
      0
    )
  );
  if (building.under_construction) {
    return buttons;
  }
  for (const producible of building.produces) {
    if (!player || !meetsTechRequirement(player, producible.required_tech_id)) {
      continue;
    }
    if (producible.kind === RisqProducibleKind.UNIT) {
      buttons.push(new RisqCreateButton({ building_id, producible }, risq, 0));
    } else if (producible.kind === RisqProducibleKind.TECH && !player.researched_techs.get(producible.id)) {
      buttons.push(new RisqResearchButton({ building_id, producible }, risq, 0));
    }
  }
  return buttons;
}
