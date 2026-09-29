import type { DwgRisq } from '../../../../risq';
import { RisqOrderType } from '../../../../model/types';
import { DrawRisqSpaceDetail } from '../../../../rendering/space';
import { RisqActionButton } from '../action_button';
import { getSettings } from '../../../../../../../../scripts/settings_store';
import { RisqHotkeyAction } from '../../../../application/input/hotkeys';

const ORDER_HOTKEY_ACTIONS: Partial<Record<RisqOrderType, RisqHotkeyAction>> = {
  [RisqOrderType.OrderType_UnitMoveSpace]: RisqHotkeyAction.MOVE,
  [RisqOrderType.OrderType_UnitMoveZone]: RisqHotkeyAction.MOVE,
  [RisqOrderType.OrderType_UnitAttackSpace]: RisqHotkeyAction.ATTACK,
  [RisqOrderType.OrderType_UnitAttackZone]: RisqHotkeyAction.ATTACK,
  [RisqOrderType.OrderType_UnitGather]: RisqHotkeyAction.GATHER,
  [RisqOrderType.OrderType_UnitRepair]: RisqHotkeyAction.REPAIR,
  [RisqOrderType.OrderType_UnitRenew]: RisqHotkeyAction.RENEW,
};

export declare interface OrderButtonConfig {
  row: number;
  col: number;
  order_type: RisqOrderType;
  image_path: string;
  description: string;
}

export class RisqOrderButton extends RisqActionButton {
  private risq: DwgRisq;
  private order_type: RisqOrderType;

  constructor(config: OrderButtonConfig, risq: DwgRisq, s: 0) {
    super(config, s);
    this.risq = risq;
    this.order_type = config.order_type;
  }

  override isClicking(): boolean {
    return super.isClicking() || this.isArmed();
  }

  private isArmed(): boolean {
    const armed_action = ORDER_HOTKEY_ACTIONS[this.risq.armed.getArmedOrder()];
    return armed_action !== undefined && armed_action === ORDER_HOTKEY_ACTIONS[this.order_type];
  }

  override dataRefreshed(): void {
    const hotkey_action = ORDER_HOTKEY_ACTIONS[this.order_type];
    this.setHotkeyCombo(hotkey_action !== undefined ? getSettings().risq_hotkeys.actions[hotkey_action] : undefined);
  }

  private getOrderType(): RisqOrderType {
    switch (this.order_type) {
      case RisqOrderType.OrderType_UnitMoveSpace:
      case RisqOrderType.OrderType_UnitMoveZone:
        return this.risq.viewport.drawDetail() === DrawRisqSpaceDetail.ZONE_DETAILS
          ? RisqOrderType.OrderType_UnitMoveZone
          : RisqOrderType.OrderType_UnitMoveSpace;
      case RisqOrderType.OrderType_UnitAttackSpace:
      case RisqOrderType.OrderType_UnitAttackZone:
        return this.risq.viewport.drawDetail() === DrawRisqSpaceDetail.ZONE_DETAILS
          ? RisqOrderType.OrderType_UnitAttackZone
          : RisqOrderType.OrderType_UnitAttackSpace;
      default:
        return this.order_type;
    }
  }

  protected released(): void {
    if (this.isHovering()) {
      if (this.isArmed()) {
        this.risq.armed.disarmOrder();
      } else {
        this.risq.armed.armOrder(this.getOrderType(), () => {});
      }
    }
  }
}
