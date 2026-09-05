import type { ColorRGB } from '../../../../../../scripts/color_rgb';
import { DwgListbox } from '../../../../util/canvas_components/scrollbar/listbox';
import type { Point2D } from '../../../../util/objects2d';
import type { DwgRisq } from '../../risq';
import { RisqOrderType, type RisqFrontendOrder } from '../../risq_data';
import { collapseBuildingCreateOrders, isBuildingOrder, isUnitOrder, type RisqOrderRowEntry } from '../../risq_orders';
import { RisqOrderRow } from '../order_row/order_row';
import { RisqOrdersScrollbar } from './orders_scrollbar';

export class RisqOrdersList extends DwgListbox<RisqOrderRow, RisqOrdersScrollbar> {
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
      title: {
        text: 'Orders',
        size: 30,
        font_color: 'rgb(0, 0, 0)',
      },
    });
    this.game = risq;
  }

  private newOrderRow(entry: RisqOrderRowEntry): RisqOrderRow {
    return new RisqOrderRow({
      w: this.config.scrollbar.w() - this.config.scrollbar.getScrollbarSize() - 2 * this.getPadding(),
      order: entry.order,
      collapsed_orders: entry.collapsed_orders,
      game: this.game,
      show_subject: true,
      onCancel: (order) => this.cancelOrder(order),
      onCancelAll: (orders) => orders.forEach((order) => this.cancelOrder(order)),
    });
  }

  private allOrders(): RisqFrontendOrder[] {
    return this.getList().flatMap((row) => row.getOrders());
  }

  private setGroupedList(orders: RisqFrontendOrder[]) {
    this.setList(collapseBuildingCreateOrders(orders).map((entry) => this.newOrderRow(entry)));
  }

  /** Replaces the already-active (server-truth) orders; queued-but-unsubmitted orders are untouched */
  setOrders(orders: RisqFrontendOrder[]) {
    const pending = this.allOrders().filter((o) => o.internal_id === undefined);
    this.setGroupedList([...orders, ...pending]);
  }

  /** Only the queued-but-unsubmitted orders (no `internal_id` yet) are ever sent on submit */
  getOrders(): RisqFrontendOrder[] {
    return this.allOrders().filter((o) => o.internal_id === undefined);
  }

  /** Called once a submission has actually been sent, since those orders are now tracked server-side */
  clearPendingOrders() {
    this.setGroupedList(this.allOrders().filter((o) => o.internal_id !== undefined));
  }

  addOrder(order: RisqFrontendOrder) {
    let orders = this.allOrders();
    if (order.clear_previous_orders) {
      orders = orders.filter((o) => {
        if (
          o.internal_id !== undefined ||
          !(
            (isUnitOrder(order.order_type) && isUnitOrder(o.order_type)) ||
            (isBuildingOrder(order.order_type) && isBuildingOrder(o.order_type))
          )
        ) {
          return true;
        }
        for (const new_id of order.subjects) {
          o.subjects = o.subjects.filter((id) => id !== new_id);
        }
        return o.subjects.length > 0;
      });
    }
    orders.push(order);
    this.setGroupedList(orders);
  }

  /** Cancels a single order: dropped locally if never submitted, otherwise queues a CancelOrder for next submit */
  cancelOrder(order: RisqFrontendOrder) {
    if (order.internal_id === undefined) {
      this.setGroupedList(this.allOrders().filter((o) => o !== order));
      return;
    }
    this.addOrder({
      player_id: order.player_id,
      order_type: RisqOrderType.OrderType_CancelOrder,
      subjects: [],
      target_id: order.internal_id,
      clear_previous_orders: false,
    });
  }

  /** Cancels every order (submitted or pending) that has the given subject */
  cancelOrdersForSubject(subject_internal_id: number) {
    for (const order of this.allOrders()) {
      if (order.subjects.includes(subject_internal_id)) {
        this.cancelOrder(order);
      }
    }
  }

  override setAllSizes(size: number, p: Point2D, w: number, h: number): void {
    super.setAllSizes(size, p, w, h);
    for (const el of this.config.list) {
      el.setW(w - this.config.scrollbar.getScrollbarSize() - 2 * this.getPadding());
    }
  }
}
