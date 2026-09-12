import type { DwgRisq } from '../../../risq';
import { RisqOrderType } from '../../../risq_data';
import { RisqActionButton } from './action_button';

export declare interface GarrisonButtonConfig {
  row: number;
  col: number;
  unit_internal_ids: number[];
}

export class RisqGarrisonButton extends RisqActionButton {
  private risq: DwgRisq;
  private unit_internal_ids: number[];
  private ungarrison: boolean;
  private armed = false;

  constructor(config: GarrisonButtonConfig, risq: DwgRisq, s: number) {
    const ungarrison = risq.getLeftPanel().isOnlyGarrisoned();
    super(
      {
        row: config.row,
        col: config.col,
        image_path: ungarrison ? 'icons/ungarrison128' : 'icons/garrison128',
        description: ungarrison ? 'Ungarrison' : 'Garrison',
      },
      s
    );
    this.risq = risq;
    this.unit_internal_ids = config.unit_internal_ids;
    this.ungarrison = ungarrison;
  }

  override isClicking(): boolean {
    return super.isClicking() || this.armed;
  }

  override dataRefreshed(): void {
    this.armed = !this.ungarrison && this.risq.getArmedOrder() === RisqOrderType.OrderType_UnitGarrison;
  }

  protected released(): void {
    if (!this.isHovering()) {
      return;
    }
    if (this.ungarrison) {
      this.risq.ungarrisonUnits(this.unit_internal_ids);
      return;
    }
    if (this.armed) {
      this.armed = false;
      this.risq.disarmOrder();
    } else {
      this.armed = true;
      this.risq.armOrder(RisqOrderType.OrderType_UnitGarrison, () => {
        this.armed = false;
      });
    }
  }
}
