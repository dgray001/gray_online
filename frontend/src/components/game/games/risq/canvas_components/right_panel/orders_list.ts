import type { ColorRGB } from '../../../../../../scripts/color_rgb';
import { DwgListbox } from '../../../../util/canvas_components/scrollbar/listbox';
import type { Point2D } from '../../../../util/objects2d';
import type { DwgRisq } from '../../risq';
import { RisqOrder } from './order';
import { RisqOrdersScrollbar } from './orders_scrollbar';

export class RisqOrdersList extends DwgListbox<RisqOrder, RisqOrdersScrollbar> {
  private game: DwgRisq;

  constructor(risq: DwgRisq, w: number, background: ColorRGB) {
    super({
      list: [
        new RisqOrder({ w }),
        new RisqOrder({ w }),
        new RisqOrder({ w }),
        new RisqOrder({ w }),
        new RisqOrder({ w }),
        new RisqOrder({ w }),
        new RisqOrder({ w }),
      ],
      scrollbar: new RisqOrdersScrollbar(risq, w, background.copy().dBrightness(-0.1)),
      draw_config: {
        fill_style: background.copy().dBrightness(-0.2).getString(),
        stroke_style: 'rgb(10, 10, 10)',
        stroke_width: 0.5,
        fixed_position: true,
      },
      padding: 2,
      gap: 2,
    });
    this.game = risq;
  }

  override setAllSizes(size: number, p: Point2D, w: number, h: number): void {
    super.setAllSizes(size, p, w, h);
    for (const el of this.config.list) {
      el.setW(w - this.config.scrollbar.getScrollbarSize() - 2 * this.getPadding());
    }
  }
}
