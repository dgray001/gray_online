import type { DwgRisq } from '../../risq';
import { RisqActionButton } from '../left_panel/action_button/action_button';

export class RisqViewModeButton extends RisqActionButton {
  private risq: DwgRisq;

  constructor(risq: DwgRisq, s: number) {
    super({ row: 0, col: 0, image_path: 'icons/eye64', description: 'Toggle view mode' }, s);
    this.risq = risq;
  }

  protected released(): void {
    if (this.isHovering()) {
      this.risq.cycleViewMode();
    }
  }
}
