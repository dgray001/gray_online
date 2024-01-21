import {DwgSquareButton} from '../../../util/canvas_components/button/square_button';
import {DwgRisq} from '../risq';

export class RisqRightPanelButton extends DwgSquareButton {
  private risq: DwgRisq;

  constructor(risq: DwgRisq) {
    super({
      button_config: {
        only_right_click: true,
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
      move_animation_speed: 200,
      rotate_animation_speed: 400,
      image_path: 'icons/triangle_gray36',
      rotation: -0.5 * Math.PI,
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
