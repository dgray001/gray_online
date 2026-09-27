import type { DwgRisq } from '../../risq';
import { RisqActionButton } from '../left_panel/action_button/action_button';
import { getSettings } from '../../../../../../scripts/settings_store';
import { RisqHotkeyAction } from '../../risq_hotkeys';

export class RisqTechTreeButton extends RisqActionButton {
  private risq: DwgRisq;

  constructor(risq: DwgRisq, s: number) {
    super({ row: 0, col: 0, image_path: 'icons/research64', description: 'Open tech tree' }, s);
    this.risq = risq;
  }

  override dataRefreshed(): void {
    this.setHotkeyCombo(getSettings().risq_hotkeys.actions[RisqHotkeyAction.TECH_TREE]);
  }

  protected released(): void {
    if (!this.isHovering()) {
      return;
    }
    const dialog = document.createElement('dwg-risq-tech-tree-dialog');
    dialog.setData({ risq: this.risq });
    this.risq.appendChild(dialog);
  }
}
