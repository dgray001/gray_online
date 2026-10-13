import type { DwgRisq } from '../../risq';
import { RisqSubmitButton } from './submit_button';

export class RisqSubmitIconButton extends RisqSubmitButton {
  constructor(risq: DwgRisq, h: number) {
    super(risq, {
      button_config: {
        allow_nonleft_clicks: false,
      },
      p: { x: 0, y: 0 },
      w: h,
      h,
      draw_config: {
        fill_style: 'rgb(50, 220, 50)',
        stroke_style: 'rgb(20, 150, 20)',
        stroke_width: 0,
        hover_fill_style: 'rgb(80, 240, 80)',
        click_fill_style: 'rgb(110, 255, 110)',
        fixed_position: true,
      },
      r: 0.1 * h,
      image_path: 'icons/submit128',
      image_on_top: true,
    });
  }
}
