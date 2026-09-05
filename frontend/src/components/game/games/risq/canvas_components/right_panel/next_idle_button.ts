import { DwgRectButton } from '../../../../util/canvas_components/button/rect_button';
import type { DwgRisq } from '../../risq';

export class RisqNextIdleButton extends DwgRectButton {
  private risq: DwgRisq;

  constructor(risq: DwgRisq, h: number) {
    super({
      button_config: {
        allow_nonleft_clicks: false,
      },
      p: { x: 0, y: 0 },
      w: 0,
      h,
      draw_config: {
        fill_style: 'rgb(220, 190, 50)',
        stroke_style: 'rgb(150, 130, 20)',
        stroke_width: 0,
        hover_fill_style: 'rgb(240, 210, 80)',
        click_fill_style: 'rgb(255, 230, 110)',
        fixed_position: true,
      },
      r: 0.1 * h,
      text: {
        text: 'Next Idle',
        config: {
          align: 'center',
          baseline: 'ideographic',
          font: `${0.75 * h}px serif`,
          fill_style: 'rgb(0, 0, 0)',
        },
        padding: 0.1 * h,
      },
    });
    this.risq = risq;
  }

  protected hovered(): void {}
  protected unhovered(): void {}
  protected clicked(): void {}
  protected released(): void {
    if (this.isHovering()) {
      this.risq.selectNextIdleUnit();
    }
  }
}
