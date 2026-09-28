import type { DwgRisq } from '../../risq';
import type { RisqProducible } from '../../risq_data';
import { unitImage } from '../../risq_unit';
import { RisqActionButton } from '../left_panel/action_button/action_button';
import type { RisqTooltipData } from '../risq_tooltip';
import { getSettings } from '../../../../../../scripts/settings_store';
import { hotkeyDisplayString } from '../../risq_hotkeys';

export class RisqMercenaryButton extends RisqActionButton {
  private risq: DwgRisq;
  private mercenary: RisqProducible;

  constructor(risq: DwgRisq, mercenary: RisqProducible, s: number) {
    const description = `Hire ${mercenary.display_name}`;
    super({ row: 0, col: 0, image_path: unitImage(mercenary.id, true), description }, s);
    this.risq = risq;
    this.mercenary = mercenary;
  }

  override isClicking(): boolean {
    return super.isClicking() || this.risq.getArmedMercenaryId() === this.mercenary.id;
  }

  override dataRefreshed(): void {
    const player = this.risq.getPlayer();
    if (!!player && this.risq.givingOrders() && !player.orders_submitted) {
      this.enable();
      this.dimmed = !!this.risq.mercenaryUnavailableReason(this.mercenary);
    } else {
      this.disable();
    }
  }

  protected released(): void {
    if (this.isHovering()) {
      this.risq.toggleMercenary(this.mercenary);
    }
  }

  protected override getTooltipData(): RisqTooltipData {
    return {
      title: this.description,
      description: this.mercenary.description,
      cost: this.mercenary.cost,
      hotkey: hotkeyDisplayString(getSettings().risq_hotkeys.hire_mercenary[this.mercenary.id]),
    };
  }
}
