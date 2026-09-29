import type { RisqProducible } from '../../model/types';
import { RisqOrderType } from '../../model/types';

export declare interface ArmedBuilding {
  id: number;
  display_name: string;
}

/** Identity of the armed order at one moment, so a later check can tell whether something re-armed in between */
export declare interface ArmedSnapshot {
  order: RisqOrderType;
  callback?: () => void;
}

/** The single interaction armed at a time: an order type (with optional build/mercenary target) or a building action */
export class RisqArmedState {
  private armed_order = RisqOrderType.NONE;
  private armed_building?: ArmedBuilding;
  private armed_mercenary?: RisqProducible;
  private armed_button_callback?: () => void;
  private gather_point_armed = false;
  private building_attack_armed = false;

  constructor(private on_change: () => void) {}

  armOrder(order_type: RisqOrderType, on_disarm: () => void, building?: ArmedBuilding) {
    this.armed_button_callback?.();
    this.armed_order = order_type;
    this.armed_building = building;
    this.armed_mercenary = undefined;
    this.armed_button_callback = on_disarm;
    this.gather_point_armed = false;
    this.building_attack_armed = false;
    this.on_change();
  }

  disarmOrder() {
    this.armed_button_callback?.();
    this.armed_order = RisqOrderType.NONE;
    this.armed_building = undefined;
    this.armed_mercenary = undefined;
    this.armed_button_callback = undefined;
    this.on_change();
  }

  disarmAll() {
    this.gather_point_armed = false;
    this.building_attack_armed = false;
    this.disarmOrder();
  }

  toggleOrder(order_type: RisqOrderType) {
    if (this.armed_order === order_type) {
      this.disarmOrder();
    } else {
      this.armOrder(order_type, () => {});
    }
  }

  armMercenary(mercenary: RisqProducible) {
    this.armOrder(RisqOrderType.OrderType_BuyMercenary, () => {});
    this.armed_mercenary = mercenary;
    this.on_change();
  }

  armGatherPoint() {
    this.disarmOrder();
    this.building_attack_armed = false;
    this.gather_point_armed = true;
    this.on_change();
  }

  disarmGatherPoint() {
    this.gather_point_armed = false;
    this.on_change();
  }

  armBuildingAttack() {
    this.disarmOrder();
    this.gather_point_armed = false;
    this.building_attack_armed = true;
    this.on_change();
  }

  disarmBuildingAttack() {
    this.building_attack_armed = false;
    this.on_change();
  }

  getArmedOrder(): RisqOrderType {
    return this.armed_order;
  }

  getArmedBuildingId(): number {
    return this.armed_building?.id ?? 0;
  }

  getArmedBuilding(): ArmedBuilding | undefined {
    return this.armed_building;
  }

  getArmedMercenary(): RisqProducible | undefined {
    return this.armed_mercenary;
  }

  getArmedMercenaryId(): number | undefined {
    return this.armed_mercenary?.id;
  }

  isGatherPointArmed(): boolean {
    return this.gather_point_armed;
  }

  isBuildingAttackArmed(): boolean {
    return this.building_attack_armed;
  }

  anythingArmed(): boolean {
    return this.armed_order !== RisqOrderType.NONE || this.gather_point_armed || this.building_attack_armed;
  }

  snapshot(): ArmedSnapshot {
    return { order: this.armed_order, callback: this.armed_button_callback };
  }

  unchangedSince(snapshot: ArmedSnapshot): boolean {
    return this.armed_order === snapshot.order && this.armed_button_callback === snapshot.callback;
  }
}
