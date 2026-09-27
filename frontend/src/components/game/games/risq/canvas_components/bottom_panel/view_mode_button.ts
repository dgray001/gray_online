import type { DwgRisq } from '../../risq';
import { RisqActionButton } from '../left_panel/action_button/action_button';
import { getSettings } from '../../../../../../scripts/settings_store';
import { RisqHotkeyAction } from '../../risq_hotkeys';

export class RisqViewModeButton extends RisqActionButton {
  private risq: DwgRisq;

  constructor(risq: DwgRisq, s: number) {
    super({ row: 0, col: 0, image_path: 'icons/eye64', description: 'Toggle view mode' }, s);
    this.risq = risq;
  }

  override dataRefreshed(): void {
    this.setHotkeyCombo(getSettings().risq_hotkeys.actions[RisqHotkeyAction.CYCLE_VIEW_MODE]);
  }

  protected released(): void {
    if (this.isHovering()) {
      this.risq.cycleViewMode();
    }
  }
}
