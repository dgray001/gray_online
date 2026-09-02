import type { DwgRisq } from '../../../risq';
import { RisqActionButton } from './action_button';
import type { RisqProducible } from '../../../risq_data';
import { canAffordCost } from '../../../risq_data';
import { unitImage } from '../../../risq_unit';

export declare interface CreateButtonConfig {
  building_id: number;
  producible: RisqProducible;
}

export class RisqCreateButton extends RisqActionButton {
  private risq: DwgRisq;
  private building_id: number;
  private producible: RisqProducible;

  constructor(config: CreateButtonConfig, risq: DwgRisq, s: number) {
    super(
      {
        row: config.producible.row,
        col: config.producible.col,
        image_path: unitImage(config.producible.id),
        description: `Create ${config.producible.display_name}`,
      },
      s
    );
    this.risq = risq;
    this.building_id = config.building_id;
    this.producible = config.producible;
  }

  override dataRefreshed(): void {
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
      this.risq.createUnit(this.building_id, this.producible.id);
    }
  }
}
