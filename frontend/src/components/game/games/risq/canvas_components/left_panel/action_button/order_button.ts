import type { DwgRisq } from '../../../risq';
import { RisqOrderType } from '../../../risq_data';
import { DrawRisqSpaceDetail } from '../../../risq_space';
import { RisqActionButton } from './action_button';
import { getSettings } from '../../../../../../../scripts/settings_store';
import { RisqHotkeyAction } from '../../../risq_hotkeys';

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
  private armed = false;

  constructor(config: OrderButtonConfig, risq: DwgRisq, s: 0) {
    super(config, s);
    this.risq = risq;
    this.order_type = config.order_type;
  }

  override isClicking(): boolean {
    return super.isClicking() || this.armed;
  }

  override dataRefreshed(): void {
    this.armed = this.risq.getArmedOrder() === this.getOrderType();
    const hotkey_action = ORDER_HOTKEY_ACTIONS[this.order_type];
    this.setHotkeyCombo(hotkey_action !== undefined ? getSettings().risq_hotkeys.actions[hotkey_action] : undefined);
  }

  private getOrderType(): RisqOrderType {
    switch (this.order_type) {
      case RisqOrderType.OrderType_UnitMoveSpace:
      case RisqOrderType.OrderType_UnitMoveZone:
        return this.risq.drawDetail() === DrawRisqSpaceDetail.ZONE_DETAILS
          ? RisqOrderType.OrderType_UnitMoveZone
          : RisqOrderType.OrderType_UnitMoveSpace;
      case RisqOrderType.OrderType_UnitAttackSpace:
      case RisqOrderType.OrderType_UnitAttackZone:
        return this.risq.drawDetail() === DrawRisqSpaceDetail.ZONE_DETAILS
          ? RisqOrderType.OrderType_UnitAttackZone
          : RisqOrderType.OrderType_UnitAttackSpace;
      default:
        return this.order_type;
    }
  }

  protected released(): void {
    if (this.isHovering()) {
      if (this.armed) {
        this.armed = false;
        this.risq.disarmOrder();
      } else {
        this.armed = true;
        this.risq.armOrder(this.getOrderType(), () => {
          this.armed = false;
        });
      }
    }
  }
}
