import type { DwgRisq } from '../../../risq';
import { RisqActionButton, type RisqActionButtonConfig } from './action_button';
import type { FoundationDrawData } from '../left_panel_data';
import { RisqOrderType } from '../../../risq_data';
import { invertPair } from '../../../risq_coordinates';

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
    if (!!player && this.risq.givingOrders() && !player.orders_submitted) {
      this.enable();
    } else {
      this.disable();
    }
  }

  protected released(): void {
    if (this.isHovering()) {
      if (this.foundation_data.is_local) {
        // Find and cancel local pending build order
        const build_orders = this.risq
          .getOrdersModel()
          .all()
          .filter((o) => {
            if (o.order_type !== RisqOrderType.OrderType_UnitBuild) {
              return false;
            }
            const outer = invertPair(o.target_id);
            return outer.y === this.foundation_data.coordinate_key;
          });
        for (const order of build_orders) {
          this.risq.getOrdersModel().cancel(order);
        }
      } else {
        // Queue player-level cancel foundation order
        this.risq.getOrdersModel().add({
          player_id: this.risq.getPlayerId(),
          order_type: RisqOrderType.OrderType_CancelFoundation,
          subjects: [],
          target_id: this.foundation_data.coordinate_key,
          clear_previous_orders: false,
        });
      }
      this.risq.getLeftPanel().close();
    }
  }
}
