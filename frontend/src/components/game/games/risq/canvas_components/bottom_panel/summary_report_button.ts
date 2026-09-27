import type { DwgRisq } from '../../risq';
import { RisqActionButton } from '../left_panel/action_button/action_button';
import { RISQ_MESSAGE_WARNING_COLOR } from '../message_queue';
import { getSettings } from '../../../../../../scripts/settings_store';
import { RisqHotkeyAction } from '../../risq_hotkeys';

export class RisqSummaryReportButton extends RisqActionButton {
  private risq: DwgRisq;

  constructor(risq: DwgRisq, s: number) {
    super({ row: 0, col: 0, image_path: 'icons/report64', description: 'Review last turn report' }, s);
    this.risq = risq;
  }

  override dataRefreshed(): void {
    this.dimmed = !this.risq.getLastTurnReport();
    this.setHotkeyCombo(getSettings().risq_hotkeys.actions[RisqHotkeyAction.SUMMARY_REPORT]);
  }

  protected released(): void {
    if (!this.isHovering()) {
      return;
    }
    if (this.dimmed) {
      this.risq.showMessage('No previous turn report to review', RISQ_MESSAGE_WARNING_COLOR);
    } else {
      this.risq.reopenLastTurnReport();
    }
  }
}
