import {DwgSquareButton} from '../../../util/canvas_components/button/square_button';
import {DwgRisq} from '../risq';

export class RisqLeftPanelButton extends DwgSquareButton {
  private risq: DwgRisq;

  constructor(risq: DwgRisq) {
    super({
      button_config: {
        only_left_click: true,
      },
      p: {x: 0, y: 0},
      s: 36,
      draw_config: {
        fill_style: 'transparent',
        stroke_style: 'transparent',
        stroke_width: 0,
        hover_fill_style: 'rgb(180, 180, 180, 0.5)',
        click_fill_style: 'rgb(210, 210, 210, 0.7)',
        fixed_position: true,
      },
      image_path: 'icons/close64',
    });
    this.risq = risq;
  }

  protected hovered(): void {}
  protected unhovered(): void {}
  protected clicked(): void {}
  protected released(): void {
    if (!this.isHovering()) {
      return;
    }
    this.risq.toggleRightPanel();
    this.setHovering(false);
  }
}
