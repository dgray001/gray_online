import type { ColorRGB } from '../../../../../../scripts/color_rgb';
import { DwgListbox } from '../../../../util/canvas_components/scrollbar/listbox';
import type { Point2D } from '../../../../util/objects2d';
import type { DwgRisq } from '../../risq';
import type { RisqOrder } from '../../risq_data';
import { RisqOrderComponent } from './order';
import { RisqOrdersScrollbar } from './orders_scrollbar';

export class RisqOrdersList extends DwgListbox<RisqOrderComponent, RisqOrdersScrollbar> {
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
      padding: 2,
      gap: 2,
    });
    this.game = risq;
  }

  setOrders(orders: RisqOrder[]) {
    this.setList(
      orders.map(
        (o) =>
          new RisqOrderComponent({
            w: 0,
            order: o,
            game: this.game,
          })
      )
    );
  }

  addOrder(_order: RisqOrder) {
    // TODO: implement
  }

  removeOrder(_order: RisqOrder) {
    // TODO: implement
  }

  override setAllSizes(size: number, p: Point2D, w: number, h: number): void {
    super.setAllSizes(size, p, w, h);
    for (const el of this.config.list) {
      el.setW(w - this.config.scrollbar.getScrollbarSize() - 2 * this.getPadding());
    }
  }
}
