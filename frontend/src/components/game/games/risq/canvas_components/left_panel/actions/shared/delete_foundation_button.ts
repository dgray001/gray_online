import type { DwgRisq } from '../../../../risq';
import { RisqActionButton } from '../action_button';
import type { RisqActionButtonConfig } from '../action_button';
import type { FoundationDrawData } from '../../left_panel_data';
import { RisqHotkeyAction } from '../../../../application/input/hotkeys';
import { RisqOrderType } from '../../../../model/types';
import { invertPair } from '../../../../model/coordinates';

export declare interface DeleteFoundationButtonConfig extends RisqActionButtonConfig {
  foundation_data: FoundationDrawData;
}

export class RisqDeleteFoundationButton extends RisqActionButton {
  private risq: DwgRisq;
  private foundation_data: FoundationDrawData;

  constructor(config: DeleteFoundationButtonConfig, risq: DwgRisq, s: number) {
    super(config, s);
    this.risq = risq;
    this.foundation_data = config.foundation_data;
  }

  override dataRefreshed(): void {
    const player = this.risq.getPlayer();
    if (!!player && this.risq.session.givingOrders() && !player.orders_submitted) {
      this.enable();
    } else {
      this.disable();
    }
  }

  override matchesAction(action: RisqHotkeyAction): boolean {
    return action === RisqHotkeyAction.DELETE;
  }

  override execute(): void {
    if (this.foundation_data.is_local) {
      const build_orders = this.risq.orders_model.all().filter((o) => {
        if (o.order_type !== RisqOrderType.OrderType_UnitBuild) {
          return false;
        }
        const outer = invertPair(o.target_id);
        return outer.y === this.foundation_data.coordinate_key;
      });
      for (const order of build_orders) {
        this.risq.orders_model.cancel(order);
      }
    } else {
      this.risq.orders_model.add({
        player_id: this.risq.getPlayerId(),
        order_type: RisqOrderType.OrderType_CancelFoundation,
        subjects: [],
        target_id: this.foundation_data.coordinate_key,
        clear_previous_orders: false,
      });
    }
    this.risq.left_panel.close();
  }
}
