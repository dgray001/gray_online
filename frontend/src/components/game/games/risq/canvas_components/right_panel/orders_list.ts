import type { ColorRGB } from '../../../../../../scripts/color_rgb';
import { DwgListbox } from '../../../../util/canvas_components/scrollbar/listbox';
import type { Point2D } from '../../../../util/objects2d';
import type { DwgRisq } from '../../risq';
import type { RisqOrdersModel, RisqOrderRowEntry } from '../../risq_orders';
import { collapseBuildingCreateOrders } from '../../risq_orders';
import { RisqOrderRow } from '../order_row/order_row';
import { RisqOrdersScrollbar } from './orders_scrollbar';

export class RisqOrdersList extends DwgListbox<RisqOrderRow, RisqOrdersScrollbar> {
  private game: DwgRisq;
  private orders: RisqOrdersModel;

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
      title: {
        text: 'Orders',
        size: 30,
        font_color: 'rgb(0, 0, 0)',
      },
    });
    this.game = risq;
    this.orders = risq.getOrdersModel();
  }

  private newOrderRow(entry: RisqOrderRowEntry): RisqOrderRow {
    return new RisqOrderRow({
      w: this.config.scrollbar.w() - this.config.scrollbar.getScrollbarSize() - 2 * this.getPadding(),
      order: entry.order,
      collapsed_orders: entry.collapsed_orders,
      game: this.game,
      show_subject: true,
      onCancel: (order) => this.orders.cancel(order),
      onCancelAll: (orders) => orders.forEach((order) => this.orders.cancel(order)),
      onSelect: (order) => this.game.selectOrderSubjects(order),
    });
  }

  refresh() {
    this.setList(collapseBuildingCreateOrders(this.orders.all()).map((entry) => this.newOrderRow(entry)));
  }

  override setAllSizes(size: number, p: Point2D, w: number, h: number): void {
    super.setAllSizes(size, p, w, h);
    for (const el of this.config.list) {
      el.setW(w - this.config.scrollbar.getScrollbarSize() - 2 * this.getPadding());
    }
  }
}
