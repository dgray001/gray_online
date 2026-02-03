import type { ColorRGB } from '../../../../../../scripts/color_rgb';
import type { RectScrollbarConfig } from '../../../../util/canvas_components/scrollbar/rect_scrollbar';
import { DwgRectScrollbar } from '../../../../util/canvas_components/scrollbar/rect_scrollbar';
import type { DwgRisq } from '../../risq';

export class RisqOrdersScrollbar extends DwgRectScrollbar {
  private game: DwgRisq;

  constructor(risq: DwgRisq, w: number, background: ColorRGB) {
    super({
      scrollbar_config: {
        value: { value: 0, value_min: 0, value_max: 0 },
        step_size: 1,
        scroll_pixel_constant: 10,
      },
      p: { x: 0, y: 0 },
      w,
      h: 0,
      scrollbar_size: 0,
      bar_dif_size: 1,
      vertical: true,
      draw_config: {
        fill_style: background.copy().getString(),
        stroke_width: 0,
        hover_fill_style: background.copy().dBrightness(0.15).getString(),
        click_fill_style: background.copy().dBrightness(0.3).getString(),
        fixed_position: true,
      },
      space_draw_config: {
        fill_style: 'rgba(10, 10, 10, 0.7)',
        stroke_width: 0,
        hover_fill_style: 'rgb(20, 20, 20, 0.8)',
        click_fill_style: 'rgb(30, 30, 30, 0.9)',
        fixed_position: true,
        stroke_matches_fill_style: true,
      },
      background_color: 'rgb(0, 0, 0)',
      arrow_scroll_amount: 4,
      space_scroll_amount: 40,
    } satisfies RectScrollbarConfig);
    this.game = risq;
  }

  scrollCallback(_value: number): void {}
}
