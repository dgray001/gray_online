import type { DwgRisq } from '../../../risq';
import { RisqRichTooltipActionButton } from './action_button';
import type { RisqProducible } from '../../../risq_data';
import { canAffordCost } from '../../../risq_data';
import { unitImage } from '../../../risq_unit';
import { RISQ_MESSAGE_WARNING_COLOR } from '../../message_queue';
import type { RisqTooltipData } from '../../risq_tooltip';

export declare interface CreateButtonConfig {
  building_id: number;
  producible: RisqProducible;
}

export class RisqCreateButton extends RisqRichTooltipActionButton {
  private risq: DwgRisq;
  private building_id: number;
  private producible: RisqProducible;

  constructor(config: CreateButtonConfig, risq: DwgRisq, s: number) {
    super(
      {
        row: config.producible.row,
        col: config.producible.col,
        image_path: unitImage(config.producible.id, true),
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
    if (!!player && this.risq.givingOrders() && !player.orders_submitted) {
      this.enable();
      this.dimmed = !canAffordCost(player, this.producible.cost);
    } else {
      this.disable();
    }
  }

  protected released(): void {
    if (this.isHovering()) {
      if (this.dimmed) {
        this.risq.showMessage('Not enough resources', RISQ_MESSAGE_WARNING_COLOR);
        return;
      }
      this.risq.createUnit(this.building_id, this.producible.id);
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
