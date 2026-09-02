import type { DwgRisq } from '../../../risq';
import { RisqOrderType } from '../../../risq_data';
import { DrawRisqSpaceDetail } from '../../../risq_space';
import { RisqActionButton } from './action_button';

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

  private getOrderType(): RisqOrderType {
    switch (this.order_type) {
      case RisqOrderType.OrderType_UnitMoveSpace:
      case RisqOrderType.OrderType_UnitMoveZone:
        return this.risq.drawDetail() === DrawRisqSpaceDetail.ZONE_DETAILS
          ? RisqOrderType.OrderType_UnitMoveZone
          : RisqOrderType.OrderType_UnitMoveSpace;
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
