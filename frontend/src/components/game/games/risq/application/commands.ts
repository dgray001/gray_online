import type { ColorRGB } from '../../../../../scripts/color_rgb';
import { DialogSize } from '../../../../dialog_box/dialog_box';
import { createMessage } from '../../../../lobby/data_models';
import { RISQ_MESSAGE_WARNING_COLOR } from '../canvas_components/message_queue';
import type { UnitToggleField } from '../canvas_components/left_panel/actions/unit/unit_toggle_button';
import type { RisqGatherPoint, RisqProducible, RisqTargetCategory, RisqUnitStance } from '../model/types';
import { RisqOrderType, RisqProducibleKind, canAffordCost } from '../model/types';
import { buildingCanProduce, researchQueued } from './orders/eligibility';
import type { RisqArmedState } from './input/armed_state';
import type { RisqOrdersModel } from './orders/orders_model';
import type { RisqOrderPlanning } from './orders/planning';
import type { RisqSession } from './session';

import '../../../../dialog_box/confirm_dialog/confirm_dialog';

/** Commands issued by buttons and hotkeys: queued single orders plus immediate (non-turn) server messages */
export class RisqCommands {
  constructor(
    private host: HTMLElement,
    private session: RisqSession,
    private orders_model: RisqOrdersModel,
    private planning: RisqOrderPlanning,
    private armed: RisqArmedState,
    private show_message: (text: string, color: ColorRGB) => void
  ) {}

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
      (kind !== RisqProducibleKind.TECH || !researchQueued(player, this.orders_model.all(), target_id))
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

  createUnit(building_id: number, unit_id: number, ctrl_held: boolean = false): void {
    if (!this.canQueueProduction(building_id, unit_id, RisqProducibleKind.UNIT)) {
      return;
    }
    this.addOrder(RisqOrderType.OrderType_BuildingCreate, [building_id], unit_id, !ctrl_held);
  }

  researchTech(building_id: number, tech_id: number, ctrl_held: boolean = false): void {
    if (!this.canQueueProduction(building_id, tech_id, RisqProducibleKind.TECH)) {
      return;
    }
    this.addOrder(RisqOrderType.OrderType_BuildingResearch, [building_id], tech_id, !ctrl_held);
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

  confirmDeleteBuilding(internal_id: number): void {
    if (!this.canCommandBuilding(internal_id)) {
      return;
    }
    this.confirm('Are you sure you want to delete this building?', () => this.deleteBuilding(internal_id));
  }

  private deleteBuilding(internal_id: number): void {
    if (!this.canCommandBuilding(internal_id)) {
      return;
    }
    this.addOrder(RisqOrderType.OrderType_BuildingDelete, [internal_id], 0, true);
  }

  stopUnit(internal_ids: number[]): void {
    if (!this.canCommandUnits(internal_ids)) {
      return;
    }
    for (const internal_id of internal_ids) {
      this.orders_model.cancelForSubject(internal_id, 'unit');
    }
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

  toggleBuildingGatherPoint(building_id: number): void {
    if (!this.canCommandBuilding(building_id)) {
      return;
    }
    if (this.armed.isGatherPointArmed()) {
      this.armed.disarmGatherPoint();
      this.clearGatherPoint(building_id);
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
    this.sendGameUpdate(JSON.stringify({ building_id, clear: false, ...point }), 'set-gather-point');
  }

  clearGatherPoint(building_id: number): void {
    if (!this.canCommandBuilding(building_id) || !this.session.getPlayer()?.buildings.get(building_id)?.gather_point) {
      return;
    }
    this.sendGameUpdate(JSON.stringify({ building_id, clear: true }), 'set-gather-point');
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

  auto_renew_pending = false;

  changeAutoRenew(internal_id: number, change: number): void {
    const player = this.session.getPlayer();
    const building = player?.buildings.get(internal_id);
    if (
      !player ||
      !building?.renew_cost ||
      building.under_construction ||
      !this.canCommandBuilding(internal_id) ||
      this.auto_renew_pending
    ) {
      return;
    }
    const count = (player.auto_renewals.get(building.building_id) ?? 0) + change;
    if (count < 0) {
      return;
    }
    if (change > 0 && !canAffordCost(player, building.renew_cost)) {
      this.show_message('Not enough resources', RISQ_MESSAGE_WARNING_COLOR);
      return;
    }
    this.auto_renew_pending = true;
    this.sendGameUpdate(JSON.stringify({ building_id: building.building_id, count }), 'set-auto-renew');
  }
}
