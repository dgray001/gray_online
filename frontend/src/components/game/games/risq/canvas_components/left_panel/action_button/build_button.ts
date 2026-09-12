import type { DwgRisq } from '../../../risq';
import { RisqOrderType } from '../../../risq_data';
import type { RisqProducible } from '../../../risq_data';
import { canAffordCost } from '../../../risq_data';
import { buildingImage } from '../../../risq_buildings';
import { RisqActionButton } from './action_button';
import type { RisqTooltipData } from '../../risq_tooltip';

export declare interface BuildButtonConfig {
  producible: RisqProducible;
}

export class RisqBuildButton extends RisqActionButton {
  private risq: DwgRisq;
  private producible: RisqProducible;
  private armed = false;

  constructor(config: BuildButtonConfig, risq: DwgRisq, s: number) {
    super(
      {
        row: config.producible.row,
        col: config.producible.col,
        image_path: buildingImage(config.producible.id, false, true),
        description: `Build ${config.producible.display_name}`,
      },
      s
    );
    this.risq = risq;
    this.producible = config.producible;
  }

  override isClicking(): boolean {
    return super.isClicking() || this.armed;
  }

  override dataRefreshed(): void {
    this.armed =
      this.risq.getArmedOrder() === RisqOrderType.OrderType_UnitBuild &&
      this.risq.getArmedBuildingId() === this.producible.id;
    const player = this.risq.getPlayer();
    if (
      !!player &&
      this.risq.givingOrders() &&
      !player.orders_submitted &&
      canAffordCost(player, this.producible.cost)
    ) {
      this.enable();
    } else {
      this.disable();
    }
  }

  protected released(): void {
    if (this.isHovering()) {
      if (this.armed) {
        this.armed = false;
        this.risq.disarmOrder();
      } else {
        this.armed = true;
        this.risq.armOrder(
          RisqOrderType.OrderType_UnitBuild,
          () => {
            this.armed = false;
          },
          { id: this.producible.id, display_name: this.producible.display_name }
        );
      }
    }
  }

  protected override getTooltipData(): RisqTooltipData {
    return {
      title: this.description,
      description: this.producible.description,
      cost: this.producible.cost,
      stamina_cost: this.producible.stamina_cost,
    };
  }
}
