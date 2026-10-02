import type { DwgRisq } from '../../../../risq';
import { RisqRichTooltipActionButton } from '../action_button';
import type { RisqProducible } from '../../../../model/types';
import { canAffordCost } from '../../../../model/types';
import { unitImage } from '../../../../rendering/assets/unit';
import { RISQ_MESSAGE_WARNING_COLOR } from '../../../message_queue';
import type { RisqTooltipData } from '../../../risq_tooltip';
import { getSettings } from '../../../../../../../../scripts/settings_store';
import { hotkeyDisplayString } from '../../../../application/input/hotkeys';

export declare interface CreateButtonConfig {
  building_id: number;
  producible: RisqProducible;
}

export class RisqCreateButton extends RisqRichTooltipActionButton {
  private risq: DwgRisq;
  private building_id: number;
  private producible: RisqProducible;
  private ctrl_held = false;

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
    if (!!player && this.risq.session.givingOrders() && !player.orders_submitted) {
      this.enable();
      this.dimmed = !canAffordCost(player, this.producible.cost);
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
      if (this.dimmed) {
        this.risq.showMessage('Not enough resources', RISQ_MESSAGE_WARNING_COLOR);
        return;
      }
      this.risq.commands.createUnit(this.building_id, this.producible.id, this.ctrl_held);
    }
  }

  protected override getTooltipData(): RisqTooltipData {
    return {
      title: this.description,
      description: this.producible.description,
      cost: this.producible.cost,
      stamina_cost: this.producible.stamina_cost,
      hotkey: hotkeyDisplayString(getSettings().risq_hotkeys.create_unit[this.producible.id]),
    };
  }
}
