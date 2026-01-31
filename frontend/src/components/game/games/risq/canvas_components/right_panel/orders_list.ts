import type { ColorRGB } from '../../../../../../scripts/color_rgb';
import type { CanvasComponent } from '../../../../util/canvas_components/canvas_component';
import { DwgListbox } from '../../../../util/canvas_components/scrollbar/listbox';
import type { DwgRisq } from '../../risq';
import { RisqOrdersScrollbar } from './orders_scrollbar';

export class RisqOrdersList extends DwgListbox<CanvasComponent, RisqOrdersScrollbar> {
  private game: DwgRisq;

  constructor(risq: DwgRisq, w: number, background: ColorRGB) {
    super({
      list: [],
      scrollbar: new RisqOrdersScrollbar(risq, w, background.copy().dBrightness(-0.1)),
      draw_config: {
        fill_style: background.copy().dBrightness(-0.2).getString(),
        stroke_style: 'rgb(10, 10, 10)',
        stroke_width: 0.5,
        fixed_position: true,
      },
    });
    console.log('!! 1', background);
    this.game = risq;
  }
}
