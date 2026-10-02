import type { DwgRisq } from '../../../../risq';
import { RisqRichTooltipActionButton } from '../action_button';
import type { RisqProducible } from '../../../../model/types';
import { RisqOrderType, canAffordCost } from '../../../../model/types';
import { techImage } from '../../../../rendering/assets/buildings';
import { RISQ_MESSAGE_WARNING_COLOR } from '../../../message_queue';
import type { RisqTooltipData } from '../../../risq_tooltip';
import { getSettings } from '../../../../../../../../scripts/settings_store';
import { hotkeyDisplayString } from '../../../../application/input/hotkeys';

export declare interface ResearchButtonConfig {
  building_id: number;
  producible: RisqProducible;
}

export class RisqResearchButton extends RisqRichTooltipActionButton {
  private risq: DwgRisq;
  private building_id: number;
  private producible: RisqProducible;
  private already_queued = false;
  private ctrl_held = false;

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
    if (!!player && this.risq.session.givingOrders() && !player.orders_submitted) {
      this.enable();
      this.already_queued = this.risq.orders_model
        .all()
        .some((o) => o.order_type === RisqOrderType.OrderType_BuildingResearch && o.target_id === this.producible.id);
      this.dimmed = this.already_queued || !canAffordCost(player, this.producible.cost);
    } else {
      this.disable();
    }
  }

  override mouseup(e: MouseEvent): void {
    this.ctrl_held = e.ctrlKey;
    super.mouseup(e);
  }

  protected released(): void {
    if (this.isHovering()) {
      if (this.already_queued) {
        this.risq.showMessage('Already queued for research', RISQ_MESSAGE_WARNING_COLOR);
        return;
      }
      if (this.dimmed) {
        this.risq.showMessage('Not enough resources', RISQ_MESSAGE_WARNING_COLOR);
        return;
      }
      this.risq.commands.researchTech(this.building_id, this.producible.id, this.ctrl_held);
    }
  }

  protected override getTooltipData(): RisqTooltipData {
    return {
      title: this.description,
      description: this.producible.description,
      cost: this.producible.cost,
      stamina_cost: this.producible.stamina_cost,
      hotkey: hotkeyDisplayString(getSettings().risq_hotkeys.research_tech[this.producible.id]),
    };
  }
}
