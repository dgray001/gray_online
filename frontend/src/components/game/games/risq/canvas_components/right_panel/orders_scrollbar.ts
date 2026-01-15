import type { RectScrollbarConfig } from '../../../../util/canvas_components/scrollbar/rect_scrollbar';
import { DwgRectScrollbar } from '../../../../util/canvas_components/scrollbar/rect_scrollbar';
import type { DwgRisq } from '../../risq';

export class RisqOrdersScrollbar extends DwgRectScrollbar {
  constructor(risq: DwgRisq, w: number) {
    super({
      scrollbar_config: {
        value: { value: 0, value_min: 0, value_max: 25 },
        step_size: 1,
      },
      p: { x: 0, y: 0 },
      w,
      h: 0,
      vertical: true,
      draw_config: {
        fill_style: 'rgba(180, 180, 180)',
        stroke_style: 'transparent',
        stroke_width: 0,
        hover_fill_style: 'rgb(200, 200, 200)',
        click_fill_style: 'rgb(220, 220, 220)',
        fixed_position: true,
      },
      space_draw_config: {
        fill_style: 'rgba(10, 10, 10, 0.7)',
        stroke_style: 'transparent',
        stroke_width: 0,
        hover_fill_style: 'rgb(20, 20, 20, 0.8)',
        click_fill_style: 'rgb(30, 30, 30, 0.9)',
        fixed_position: true,
      },
      background_color: 'rgb(0, 0, 0)',
    } satisfies RectScrollbarConfig);
  }

  scrollCallback(value: number): void {
    console.log('new value', value);
  }
}
