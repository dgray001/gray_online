import type { DwgRisq } from '../../risq';
import type { RisqOrderType } from '../../risq_data';
import { RisqActionButton } from './action_button';

/** Declarative placement + look of one order button in the action grid */
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

  constructor(risq: DwgRisq, config: OrderButtonConfig, s: number) {
    super(config, s);
    this.risq = risq;
    this.order_type = config.order_type;
  }

  override isClicking(): boolean {
    return super.isClicking() || this.armed;
  }

  protected released(): void {
    if (this.isHovering()) {
      if (this.armed) {
        this.armed = false;
        this.risq.disarmOrder();
      } else {
        this.armed = true;
        this.risq.armOrder(this.order_type, () => {
          this.armed = false;
        });
      }
    }
  }
}
