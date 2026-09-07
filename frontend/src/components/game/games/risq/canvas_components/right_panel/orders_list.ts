import type { ColorRGB } from '../../../../../../scripts/color_rgb';
import { DwgListbox } from '../../../../util/canvas_components/scrollbar/listbox';
import type { Point2D } from '../../../../util/objects2d';
import type { DwgRisq } from '../../risq';
import type { RisqOrdersModel, RisqOrderRowEntry } from '../../risq_orders';
import { collapseBuildingCreateOrders, isBuildingOrder, isUnitOrder } from '../../risq_orders';
import { RisqOrderRow } from '../order_row/order_row';
import { RisqOrdersScrollbar } from './orders_scrollbar';

export class RisqOrdersList extends DwgListbox<RisqOrderRow, RisqOrdersScrollbar> {
  private game: DwgRisq;
  private orders: RisqOrdersModel;
  // when set, only orders targeting this subject are shown (used by the left panel); undefined shows all
  private subject_internal_id?: number;
  // unit and building internal ids are separate counters and can collide, so the subject filter must also match order category
  private subject_kind?: 'unit' | 'building';
  private cancel_disabled = false;

  constructor(risq: DwgRisq, w: number, background: ColorRGB, with_title = true) {
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
      title: with_title
        ? {
            text: 'Orders',
            size: 30,
            font_color: 'rgb(0, 0, 0)',
          }
        : undefined,
    });
    this.game = risq;
    this.orders = risq.getOrdersModel();
  }

  setSubject(subject_internal_id: number | undefined, subject_kind: 'unit' | 'building' | undefined) {
    this.subject_internal_id = subject_internal_id;
    this.subject_kind = subject_kind;
  }

  // Overridden so disabling only blocks each row's cancel button; rows stay clickable/scrollable for viewing
  override disable(): void {
    this.cancel_disabled = true;
    for (const row of this.config.list) {
      row.disableCancel();
    }
  }

  override enable(): void {
    this.cancel_disabled = false;
    for (const row of this.config.list) {
      row.enableCancel();
    }
  }

  private newOrderRow(entry: RisqOrderRowEntry): RisqOrderRow {
    const row = new RisqOrderRow({
      w: this.config.scrollbar.w() - this.config.scrollbar.getScrollbarSize() - 2 * this.getPadding(),
      order: entry.order,
      collapsed_orders: entry.collapsed_orders,
      cancelling: entry.order.internal_id !== undefined && this.orders.isCancelling(entry.order.internal_id),
      game: this.game,
      show_subject: this.subject_internal_id === undefined,
      onCancel: (order) => this.orders.cancel(order),
      onCancelAll: (orders) => orders.forEach((order) => this.orders.cancel(order)),
      onSelect: (order) => this.game.selectOrderSubjects(order),
    });
    if (this.cancel_disabled) {
      row.disableCancel();
    }
    return row;
  }

  refresh() {
    const subject_internal_id = this.subject_internal_id;
    const kind_matches = this.subject_kind === 'building' ? isBuildingOrder : isUnitOrder;
    const orders =
      subject_internal_id === undefined
        ? this.orders.all()
        : this.orders.all().filter((o) => kind_matches(o.order_type) && o.subjects.includes(subject_internal_id));
    this.setList(collapseBuildingCreateOrders(orders).map((entry) => this.newOrderRow(entry)));
  }

  override setAllSizes(size: number, p: Point2D, w: number, h: number): void {
    super.setAllSizes(size, p, w, h);
    for (const el of this.config.list) {
      el.setW(w - this.config.scrollbar.getScrollbarSize() - 2 * this.getPadding());
    }
  }
}
