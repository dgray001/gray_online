import type { ColorRGB } from '../../../../../scripts/color_rgb';
import { DialogSize } from '../../../../dialog_box/dialog_box';
import { createMessage } from '../../../../lobby/data_models';
import { RISQ_MESSAGE_WARNING_COLOR } from '../canvas_components/message_queue';
import type { UnitToggleField } from '../canvas_components/left_panel/actions/unit/unit_toggle_button';
import type { RisqGatherPoint, RisqProducible, RisqTargetCategory, RisqUnitStance } from '../model/types';
import { RisqOrderType, RisqProducibleKind, canAffordCost } from '../model/types';
import { buildingCanProduce, researchQueued } from './orders/orders_model';
import type { RisqArmedState } from './input/armed_state';
import type { RisqOrdersModel } from './orders/orders_model';
import type { RisqOrderPlanning } from './orders/planning';
import type { RisqSession } from './session';
import { RisqAutoRenewQueue } from './auto_renew_queue';
import { RisqGatherPointQueue } from './gather_point_queue';

import '../../../../dialog_box/confirm_dialog/confirm_dialog';

/** Commands issued by buttons and hotkeys: queued single orders plus immediate (non-turn) server messages */
export class RisqCommands {
  readonly auto_renew: RisqAutoRenewQueue;
  readonly gather_points: RisqGatherPointQueue;
  constructor(
    private host: HTMLElement,
    private session: RisqSession,
    private orders_model: RisqOrdersModel,
    private planning: RisqOrderPlanning,
    private armed: RisqArmedState,
    private show_message: (text: string, color: ColorRGB) => void
  ) {
    this.auto_renew = new RisqAutoRenewQueue(
      this.session,
      (building_id: number, count: number): void =>
        this.sendGameUpdate(JSON.stringify({ building_id, count }), 'set-auto-renew'),
      (): void => this.orders_model.triggerChange()
    );
    this.gather_points = new RisqGatherPointQueue(
      this.session,
      (building_id: number, point?: RisqGatherPoint): void =>
        this.sendGameUpdate(JSON.stringify({ building_id, clear: point === undefined, ...point }), 'set-gather-point'),
      (): void => this.orders_model.triggerChange()
    );
  }

  private canCommandUnits(internal_ids: number[]): boolean {
    const player = this.session.getPlayer();
    return (
      this.session.canGiveOrders() &&
      internal_ids.length > 0 &&
      internal_ids.every((id: number): boolean => !!player?.units.has(id))
    );
  }

  private canCommandBuilding(internal_id: number): boolean {
    return this.session.canGiveOrders() && !!this.session.getPlayer()?.buildings.has(internal_id);
  }

  private canQueueProduction(building_id: number, target_id: number, kind: RisqProducibleKind): boolean {
    const player = this.session.getPlayer();
    const building = player?.buildings.get(building_id);
    const producible = building?.produces.find((p: RisqProducible): boolean => p.kind === kind && p.id === target_id);
    if (!player || !building || !producible || !this.canCommandBuilding(building_id)) {
      return false;
    }
    return (
      buildingCanProduce(player, building, producible) &&
      canAffordCost(player, producible.cost) &&
      (kind !== RisqProducibleKind.TECH || !researchQueued(player, this.orders_model.uncancelledOrders(), target_id))
    );
  }

  private sendGameUpdate(content: string, kind: string): void {
    const game_update = createMessage(`player-${this.session.getPlayerId()}`, 'game-update', content, kind);
    this.host.dispatchEvent(new CustomEvent('game_update', { detail: game_update, bubbles: true }));
  }

  private confirm(question: string, on_confirmed: () => void): void {
    const dialog = document.createElement('dwg-confirm-dialog');
    dialog.setData({ question, size: DialogSize.SMALL });
    dialog.addEventListener('confirmed', on_confirmed);
    this.host.appendChild(dialog);
  }

  private addOrder(
    order_type: RisqOrderType,
    subjects: number[],
    target_id: number,
    clear_previous_orders: boolean
  ): void {
    this.orders_model.add({
      player_id: this.session.getPlayerId(),
      order_type,
      subjects,
      target_id,
      clear_previous_orders,
    });
  }

  createUnit(building_id: number, unit_id: number, ctrl_held: boolean = false): boolean {
    if (!this.canQueueProduction(building_id, unit_id, RisqProducibleKind.UNIT)) {
      return false;
    }
    this.addOrder(RisqOrderType.OrderType_BuildingCreate, [building_id], unit_id, !ctrl_held);
    return true;
  }

  researchTech(building_id: number, tech_id: number, ctrl_held: boolean = false): boolean {
    if (!this.canQueueProduction(building_id, tech_id, RisqProducibleKind.TECH)) {
      return false;
    }
    this.addOrder(RisqOrderType.OrderType_BuildingResearch, [building_id], tech_id, !ctrl_held);
    return true;
  }

  confirmDeleteUnit(internal_ids: number[]): void {
    if (!this.canCommandUnits(internal_ids)) {
      return;
    }
    this.confirm(`Are you sure you want to delete ${internal_ids.length === 1 ? 'this unit' : 'these units'}?`, () =>
      this.deleteUnit(internal_ids)
    );
  }

  private deleteUnit(internal_ids: number[]): void {
    if (!this.canCommandUnits(internal_ids)) {
      return;
    }
    this.addOrder(RisqOrderType.OrderType_UnitDelete, internal_ids, 0, true);
  }

  confirmDeleteBuilding(internal_ids: number[]): void {
    if (internal_ids.length === 0 || !internal_ids.every((id: number): boolean => this.canCommandBuilding(id))) {
      return;
    }
    this.confirm(
      `Are you sure you want to delete ${internal_ids.length === 1 ? 'this building' : 'these buildings'}?`,
      (): void => this.deleteBuilding(internal_ids)
    );
  }

  private deleteBuilding(internal_ids: number[]): void {
    if (internal_ids.length === 0 || !internal_ids.every((id: number): boolean => this.canCommandBuilding(id))) {
      return;
    }
    this.addOrder(RisqOrderType.OrderType_BuildingDelete, internal_ids, 0, true);
  }

  stopUnit(internal_ids: number[]): void {
    if (!this.canCommandUnits(internal_ids)) {
      return;
    }
    this.orders_model.batch((): void => {
      for (const internal_id of internal_ids) {
        this.orders_model.cancelForSubject(internal_id, 'unit');
      }
    });
  }

  stopBuilding(internal_id: number): void {
    if (!this.canCommandBuilding(internal_id)) {
      return;
    }
    this.orders_model.cancelForSubject(internal_id, 'building');
  }

  ungarrisonUnits(internal_ids: number[]): void {
    if (!this.canCommandUnits(internal_ids)) {
      return;
    }
    this.addOrder(RisqOrderType.OrderType_UnitUngarrison, internal_ids, 0, true);
  }

  ungarrisonBuilding(building_id: number): void {
    if (!this.canCommandBuilding(building_id)) {
      return;
    }
    this.ungarrisonUnits(this.session.getPlayer()?.buildings.get(building_id)?.garrisoned_units ?? []);
  }

  toggleBuildingAttack(): void {
    if (this.armed.isBuildingAttackArmed()) {
      this.armed.disarmBuildingAttack();
    } else {
      this.armed.armBuildingAttack();
    }
  }

  toggleBuildingGatherPoint(building_ids: number[]): void {
    if (!building_ids.length || !building_ids.every((id: number): boolean => this.canCommandBuilding(id))) {
      return;
    }
    if (this.armed.isGatherPointArmed()) {
      this.armed.disarmGatherPoint();
      for (const id of building_ids) {
        this.clearGatherPoint(id);
      }
    } else {
      this.armed.armGatherPoint();
    }
  }

  toggleMercenary(mercenary: RisqProducible): void {
    if (this.armed.getArmedMercenaryId() === mercenary.id) {
      this.armed.disarmOrder();
      return;
    }
    const unavailable_reason = this.planning.mercenaryUnavailableReason(mercenary);
    if (unavailable_reason) {
      this.show_message(unavailable_reason, RISQ_MESSAGE_WARNING_COLOR);
      return;
    }
    this.armed.armMercenary(mercenary);
  }

  setGatherPoint(building_id: number, point: RisqGatherPoint): void {
    if (!this.canCommandBuilding(building_id)) {
      return;
    }
    this.gather_points.change(building_id, point);
  }

  clearGatherPoint(building_id: number): void {
    const building = this.session.getPlayer()?.buildings.get(building_id);
    if (!this.canCommandBuilding(building_id) || !building || !this.gather_points.point(building)) {
      return;
    }
    this.gather_points.change(building_id);
  }

  private sendUnitBehavior(internal_ids: number[], fields: Record<string, unknown>): void {
    if (!this.canCommandUnits(internal_ids)) {
      return;
    }
    this.sendGameUpdate(JSON.stringify({ internal_ids, ...fields }), 'set-unit-behavior');
  }

  setUnitStance(internal_ids: number[], stance: RisqUnitStance): void {
    this.sendUnitBehavior(internal_ids, { stance });
  }

  setDefaultUnitBehavior(
    fields: Partial<{
      stance: RisqUnitStance;
      attack_back: boolean;
      interrupt_current: boolean;
      target_priority: RisqTargetCategory[];
    }>
  ): void {
    if (this.session.canGiveOrders()) {
      this.sendGameUpdate(JSON.stringify(fields), 'set-default-unit-behavior');
    }
  }

  setUnitToggle(internal_ids: number[], field: UnitToggleField, value: boolean): void {
    this.sendUnitBehavior(internal_ids, { [field]: value });
  }

  /** Turns the flag on for all the units unless every one of them already has it on */
  toggleUnitFlag(internal_ids: number[], field: UnitToggleField): void {
    const player = this.session.getPlayer();
    const values = internal_ids.map((id) => player?.units.get(id)?.[field]);
    const active = values.length > 0 && values.every((v) => v === true);
    this.setUnitToggle(internal_ids, field, !active);
  }

  setUnitTargetPriority(internal_ids: number[], target_priority: RisqTargetCategory[]): void {
    this.sendUnitBehavior(internal_ids, { target_priority });
  }

  changeAutoRenew(internal_id: number, change: number): void {
    const player = this.session.getPlayer();
    const building = player?.buildings.get(internal_id);
    if (!player || !building?.renew_cost || building.under_construction || !this.canCommandBuilding(internal_id)) {
      return;
    }
    if (!this.auto_renew.change(building.building_id, change, building.renew_cost)) {
      this.show_message('Not enough resources', RISQ_MESSAGE_WARNING_COLOR);
      return;
    }
  }
}
