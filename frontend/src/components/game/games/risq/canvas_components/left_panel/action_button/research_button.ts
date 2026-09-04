import type { DwgRisq } from '../../../risq';
import { RisqActionButton } from './action_button';
import type { RisqProducible } from '../../../risq_data';
import { canAffordCost } from '../../../risq_data';
import { techImage } from '../../../risq_techs';
import type { RisqTooltipData } from '../../risq_tooltip';

export declare interface ResearchButtonConfig {
  building_id: number;
  producible: RisqProducible;
}

export class RisqResearchButton extends RisqActionButton {
  private risq: DwgRisq;
  private building_id: number;
  private producible: RisqProducible;

  constructor(config: ResearchButtonConfig, risq: DwgRisq, s: number) {
    super(
      {
        row: config.producible.row,
        col: config.producible.col,
        image_path: techImage(config.producible.id),
        description: `Research ${config.producible.display_name}`,
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
      this.risq.researchTech(this.building_id, this.producible.id);
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
