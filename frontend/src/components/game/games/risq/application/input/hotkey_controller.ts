import type { ColorRGB } from '../../../../../../scripts/color_rgb';
import { getSettings } from '../../../../../../scripts/settings_store';
import { isDialogOpen, isTypingInInput } from '../../../../../../scripts/util';
import type { Point2D } from '../../../../util/objects2d';
import type { RisqLeftPanel } from '../../canvas_components/left_panel/left_panel';
import { RISQ_MESSAGE_WARNING_COLOR } from '../../canvas_components/message_queue';
import { RisqOrderType, RisqProducibleKind, RisqUnitStance, canAffordCost } from '../../model/types';
import type { RisqViewport } from '../../rendering/board/viewport';
import { RisqViewMode } from '../../rendering/terrain';
import type { RisqCommands } from '../commands';
import type { RisqOrdersModel } from '../orders/orders_model';
import type { RisqOrderSubmission } from '../orders/submission';
import type { RisqControlGroups } from '../selection/control_groups';
import type { RisqSelection } from '../selection/selection';
import type { RisqSession } from '../session';
import type { RisqArmedState } from './armed_state';
import type { HotkeyCombo, RisqHotkeyLookupEntry } from './hotkeys';
import {
  RisqHotkeyAction,
  buildHotkeyLookup,
  comboFromKeyboardEvent,
  comboFromMouseEvent,
  comboToString,
  hotkeyApplies,
} from './hotkeys';

/** UI actions hotkeys trigger that live on the game component */
export declare interface HotkeyHost {
  showMessage(text: string, color: ColorRGB): void;
  reviewLastTurnReport(): void;
  openTechTree(): void;
  selectNextIdleUnit(): void;
  goToCoordinate(coordinate: Point2D): void;
}

type ProducibleEntry = Extract<RisqHotkeyLookupEntry, { kind: 'create_unit' | 'research_tech' | 'build_building' }>;

const VIEW_MODE_ACTIONS: Partial<Record<RisqHotkeyAction, RisqViewMode>> = {
  [RisqHotkeyAction.VIEW_MODE_ALL]: RisqViewMode.ALL,
  [RisqHotkeyAction.VIEW_MODE_RESOURCE]: RisqViewMode.RESOURCE,
  [RisqHotkeyAction.VIEW_MODE_MILITARY]: RisqViewMode.MILITARY,
  [RisqHotkeyAction.VIEW_MODE_OWNERSHIP]: RisqViewMode.OWNERSHIP,
  [RisqHotkeyAction.VIEW_MODE_REGION]: RisqViewMode.REGION,
};

const STANCE_ACTIONS: Partial<Record<RisqHotkeyAction, RisqUnitStance>> = {
  [RisqHotkeyAction.STANCE_AGGRESSIVE]: RisqUnitStance.AGGRESSIVE,
  [RisqHotkeyAction.STANCE_DEFENSIVE]: RisqUnitStance.DEFENSIVE,
  [RisqHotkeyAction.STANCE_STAND_GROUND]: RisqUnitStance.STAND_GROUND,
  [RisqHotkeyAction.STANCE_PASSIVE]: RisqUnitStance.PASSIVE,
};

const VILLAGER_ORDER_ACTIONS: Partial<Record<RisqHotkeyAction, RisqOrderType>> = {
  [RisqHotkeyAction.GATHER]: RisqOrderType.OrderType_UnitGather,
  [RisqHotkeyAction.REPAIR]: RisqOrderType.OrderType_UnitRepair,
  [RisqHotkeyAction.RENEW]: RisqOrderType.OrderType_UnitRenew,
};

/** Keyboard and extra-mouse-button input: control groups plus configurable hotkeys, resolved against the selection */
export class RisqHotkeyController {
  constructor(
    private host: HotkeyHost,
    private session: RisqSession,
    private viewport: RisqViewport,
    private armed: RisqArmedState,
    private left_panel: RisqLeftPanel,
    private selection: RisqSelection,
    private control_groups: RisqControlGroups,
    private orders_model: RisqOrdersModel,
    private commands: RisqCommands,
    private submission: RisqOrderSubmission
  ) {}

  attach() {
    document.body.addEventListener('keydown', this.handleKeydown);
    document.body.addEventListener('keyup', this.handleKeyup);
  }

  detach() {
    document.body.removeEventListener('keydown', this.handleKeydown);
    document.body.removeEventListener('keyup', this.handleKeyup);
  }

  private handleKeydown = (e: KeyboardEvent) => {
    if (isTypingInInput() || isDialogOpen()) {
      return;
    }
    if (/^[0-9]$/.test(e.key)) {
      const group = e.key === '0' ? 10 : parseInt(e.key, 10);
      if (e.ctrlKey) {
        this.control_groups.assign(group);
      } else {
        this.control_groups.recall(group);
      }
      return;
    }
    this.pressed(this.resolveEntry(comboFromKeyboardEvent(e)));
  };

  private handleKeyup = (e: KeyboardEvent) => {
    if (isTypingInInput() || isDialogOpen()) {
      return;
    }
    this.released(this.resolveEntry(comboFromKeyboardEvent(e)));
  };

  extraMouseButton(e: MouseEvent) {
    if (isTypingInInput() || isDialogOpen()) {
      return;
    }
    const entry = this.resolveEntry(comboFromMouseEvent(e));
    if (e.type === 'mousedown') {
      this.pressed(entry);
    } else {
      this.released(entry);
    }
  }

  private resolveEntry(combo: HotkeyCombo): RisqHotkeyLookupEntry | undefined {
    const entries = buildHotkeyLookup(getSettings().risq_hotkeys).get(comboToString(combo)) ?? [];
    if (entries.length <= 1) {
      return entries[0];
    }
    const hotkey_selection = this.selection.hotkeySelection();
    return hotkey_selection ? entries.find((entry) => hotkeyApplies(entry, hotkey_selection)) : undefined;
  }

  private pressed(entry: RisqHotkeyLookupEntry | undefined) {
    if (entry?.kind !== 'cycle_building') {
      return;
    }
    const next = this.selection.cycleBuildingOfType(entry.id);
    if (next) {
      this.host.goToCoordinate(next.space_coordinate);
    }
  }

  private released(entry: RisqHotkeyLookupEntry | undefined) {
    if (!entry) {
      return;
    }
    if (entry.kind === 'action') {
      this.action(entry.action);
    } else if (entry.kind === 'create_unit' || entry.kind === 'research_tech' || entry.kind === 'build_building') {
      this.producible(entry);
    } else if (entry.kind === 'hire_mercenary') {
      const mercenary = this.session.getPlayer()?.available_mercenaries.find((m) => m.id === entry.id);
      if (mercenary) {
        this.commands.toggleMercenary(mercenary);
      }
    }
  }

  private action(action: RisqHotkeyAction) {
    const building = this.selection.selectedBuilding();
    const subject_ids = this.selection.subjectUnitIds();
    const zone_view = this.viewport.zoneView();
    const view_mode = VIEW_MODE_ACTIONS[action];
    const stance = STANCE_ACTIONS[action];
    const villager_order = VILLAGER_ORDER_ACTIONS[action];
    if (view_mode !== undefined) {
      this.viewport.setViewMode(view_mode);
    } else if (stance !== undefined) {
      if (this.left_panel.isOnlyMilitary()) {
        this.commands.setUnitStance(subject_ids, stance);
      }
    } else if (villager_order !== undefined) {
      if (this.left_panel.isOnlyVillagers()) {
        this.armed.toggleOrder(villager_order);
      }
    } else {
      this.otherAction(action, building?.internal_id, subject_ids, zone_view);
    }
  }

  private otherAction(
    action: RisqHotkeyAction,
    building_id: number | undefined,
    subject_ids: number[],
    zone_view: boolean
  ) {
    switch (action) {
      case RisqHotkeyAction.CYCLE_VIEW_MODE:
        this.viewport.cycleViewMode();
        break;
      case RisqHotkeyAction.SUMMARY_REPORT:
        this.host.reviewLastTurnReport();
        break;
      case RisqHotkeyAction.TECH_TREE:
        this.host.openTechTree();
        break;
      case RisqHotkeyAction.NEXT_IDLE:
        this.host.selectNextIdleUnit();
        break;
      case RisqHotkeyAction.SUBMIT_ORDERS:
        this.submission.confirmSubmit();
        break;
      case RisqHotkeyAction.MOVE:
        this.armed.toggleOrder(
          zone_view ? RisqOrderType.OrderType_UnitMoveZone : RisqOrderType.OrderType_UnitMoveSpace
        );
        break;
      case RisqHotkeyAction.ATTACK:
        this.armed.toggleOrder(
          zone_view ? RisqOrderType.OrderType_UnitAttackZone : RisqOrderType.OrderType_UnitAttackSpace
        );
        break;
      case RisqHotkeyAction.STOP:
        if (subject_ids.length) {
          this.commands.stopUnit(subject_ids);
        }
        break;
      case RisqHotkeyAction.DELETE:
        if (building_id !== undefined) {
          this.commands.confirmDeleteBuilding(building_id);
        } else if (subject_ids.length) {
          this.commands.confirmDeleteUnit(subject_ids);
        }
        break;
      case RisqHotkeyAction.GARRISON:
        if (this.left_panel.isOnlyGarrisoned()) {
          this.commands.ungarrisonUnits(subject_ids);
        } else {
          this.armed.toggleOrder(RisqOrderType.OrderType_UnitGarrison);
        }
        break;
      case RisqHotkeyAction.UNGARRISON:
        if (building_id !== undefined) {
          this.commands.ungarrisonBuilding(building_id);
        } else if (subject_ids.length) {
          this.commands.ungarrisonUnits(subject_ids);
        }
        break;
      case RisqHotkeyAction.TOGGLE_INTERRUPT_CURRENT:
      case RisqHotkeyAction.TOGGLE_ATTACK_BACK:
        if (this.left_panel.isOnlyMilitary()) {
          const field = action === RisqHotkeyAction.TOGGLE_ATTACK_BACK ? 'attack_back' : 'interrupt_current';
          this.commands.toggleUnitFlag(subject_ids, field);
        }
        break;
      default:
        if (building_id !== undefined) {
          this.buildingAction(action, building_id);
        }
        break;
    }
  }

  private buildingAction(action: RisqHotkeyAction, building_id: number) {
    switch (action) {
      case RisqHotkeyAction.BUILDING_ATTACK:
        this.commands.toggleBuildingAttack();
        break;
      case RisqHotkeyAction.BUILDING_GATHER_POINT:
        this.commands.toggleBuildingGatherPoint(building_id);
        break;
      case RisqHotkeyAction.BUILDING_CLEAR_GATHER_POINT:
        this.armed.disarmGatherPoint();
        this.commands.clearGatherPoint(building_id);
        break;
      case RisqHotkeyAction.BUILDING_DELETE:
        this.commands.confirmDeleteBuilding(building_id);
        break;
      default:
        break;
    }
  }

  private producible(entry: ProducibleEntry) {
    const building = this.selection.selectedBuilding();
    const player = this.session.getPlayer();
    if (!player) {
      return;
    }
    if (entry.kind === 'build_building') {
      this.armBuild(entry.id);
      return;
    }
    const kind = entry.kind === 'create_unit' ? RisqProducibleKind.UNIT : RisqProducibleKind.TECH;
    const producible = building?.produces.find((p) => p.kind === kind && p.id === entry.id);
    if (!building || !producible) {
      return;
    }
    const already_queued =
      entry.kind === 'research_tech' &&
      this.orders_model
        .all()
        .some((o) => o.order_type === RisqOrderType.OrderType_BuildingResearch && o.target_id === entry.id);
    if (already_queued) {
      this.host.showMessage('Already queued for research', RISQ_MESSAGE_WARNING_COLOR);
    } else if (!canAffordCost(player, producible.cost)) {
      this.host.showMessage('Not enough resources', RISQ_MESSAGE_WARNING_COLOR);
    } else if (entry.kind === 'create_unit') {
      this.commands.createUnit(building.internal_id, entry.id);
    } else {
      this.commands.researchTech(building.internal_id, entry.id);
    }
  }

  private armBuild(building_id: number) {
    const player = this.session.getPlayer();
    const producible = this.selection
      .subjectUnitIds()
      .map((id) => player?.units.get(id)?.builds.find((p) => p.id === building_id))
      .find((p) => !!p);
    if (!producible) {
      return;
    }
    if (
      this.armed.getArmedOrder() === RisqOrderType.OrderType_UnitBuild &&
      this.armed.getArmedBuildingId() === building_id
    ) {
      this.armed.disarmOrder();
    } else {
      this.armed.armOrder(RisqOrderType.OrderType_UnitBuild, () => {}, {
        id: building_id,
        display_name: producible.display_name,
      });
    }
  }
}
