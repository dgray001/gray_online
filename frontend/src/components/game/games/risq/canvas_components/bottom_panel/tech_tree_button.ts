import type { DwgRisq } from '../../risq';
import { RisqActionButton } from '../left_panel/action_button/action_button';

export class RisqTechTreeButton extends RisqActionButton {
  private risq: DwgRisq;

  constructor(risq: DwgRisq, s: number) {
    super({ row: 0, col: 0, image_path: 'icons/research64', description: 'Open tech tree' }, s);
    this.risq = risq;
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
