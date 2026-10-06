import type { ColorRGB } from '../../../../../../scripts/color_rgb';
import type { BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import type { DropdownBounds } from '../../../../util/canvas_components/dropdown/multi_select_dropdown';
import { DwgListbox } from '../../../../util/canvas_components/scrollbar/listbox';
import type { Point2D } from '../../../../util/objects2d';
import type { DwgRisq } from '../../risq';
import type { OrderFilterContext } from '../../application/orders/order_filter';
import { OrderSubjectClass } from '../../application/orders/order_filter';
import type { RisqOrdersModel, RisqOrderRowEntry } from '../../application/orders/orders_model';
import { collapseBuildingCreateOrders, isBuildingOrder, isUnitOrder } from '../../application/orders/orders_model';
import { RisqUnitType } from '../../model/types';
import { ORDER_FILTER_BAR_H, RisqOrderFilterBar } from '../order_filter_bar/order_filter_bar';
import { RisqOrderRow } from '../order_row/order_row';
import { RisqOrdersScrollbar } from './orders_scrollbar';

export class RisqOrdersList extends DwgListbox<RisqOrderRow, RisqOrdersScrollbar> {
  private game: DwgRisq;
  private orders: RisqOrdersModel;
  // when set, only orders targeting one of these subjects are shown (used by the left panel); undefined shows all
  private subject_internal_ids?: number[];
  // unit and building internal ids are separate counters and can collide, so the subject filter must also match order category
  private subject_kind?: 'unit' | 'building';
  private cancel_disabled = false;
  private filter_bar: RisqOrderFilterBar;

  constructor(risq: DwgRisq, w: number, background: ColorRGB, with_title = true, show_subject_filter: boolean = true) {
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
    this.orders = risq.orders_model;
    this.filter_bar = new RisqOrderFilterBar(
      risq,
      () => this.popoverBounds(),
      () => this.refresh(),
      show_subject_filter
    );
  }

  private popoverBounds(): DropdownBounds {
    return { x: this.xi(), y: this.yi(), w: this.w(), h: this.h() };
  }

  setSubject(subject_internal_ids: number[] | undefined, subject_kind: 'unit' | 'building' | undefined) {
    this.subject_internal_ids = subject_internal_ids;
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
      cancelling: this.orders.isCancelling(entry.order, this.subject_internal_ids),
      explicit_cancel: this.orders.isExplicitlyCancelling(entry.order, this.subject_internal_ids),
      game: this.game,
      show_subject: this.subject_internal_ids === undefined || this.subject_internal_ids.length !== 1,
      onCancel: (order) => this.orders.cancel(order),
      onCancelAll: (orders) => orders.forEach((order) => this.orders.cancel(order)),
      onSelect: (order) => this.game.selection.selectOrderSubjects(order),
    });
    if (this.cancel_disabled) {
      row.disableCancel();
    }
    return row;
  }

  refresh() {
    const subject_internal_ids = this.subject_internal_ids;
    const kind_matches = this.subject_kind === 'building' ? isBuildingOrder : isUnitOrder;
    const orders =
      subject_internal_ids === undefined
        ? this.orders.all()
        : this.orders
            .all()
            .filter((o) => kind_matches(o.order_type) && o.subjects.some((s) => subject_internal_ids.includes(s)));
    const context = this.filterContext();
    const shown = orders.filter((o) => this.filter_bar.filter.matches(o, context));
    this.setList(collapseBuildingCreateOrders(shown).map((entry) => this.newOrderRow(entry)));
  }

  private filterContext(): OrderFilterContext {
    const units = this.game.getPlayer()?.units;
    return {
      isCancelled: (order) => this.orders.isCancelling(order, this.subject_internal_ids),
      unitSubjectClass: (unit_internal_id) => {
        const unit = units?.get(unit_internal_id);
        if (!unit) {
          return undefined;
        }
        return unit.unit_type === RisqUnitType.ECONOMIC
          ? OrderSubjectClass.ECONOMIC_UNITS
          : OrderSubjectClass.MILITARY_UNITS;
      },
    };
  }

  override draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void {
    super.draw(ctx, transform, dt);
    this.filter_bar.draw(ctx, transform, dt);
  }

  override isHovering(): boolean {
    return this.filter_bar.isHovering() || super.isHovering();
  }

  override scroll(dy: number, mode: number, dx?: number): boolean {
    return this.filter_bar.scroll(dy) || super.scroll(dy, mode, dx);
  }

  override mousemove(canvas: Point2D, screen: Point2D, transform: BoardTransformData): boolean {
    if (!this.filter_bar.mousemove(canvas, screen, transform)) {
      return super.mousemove(canvas, screen, transform);
    }
    this.config.list.forEach((row) => row.setHovering(false));
    return true;
  }

  override mousedown(e: MouseEvent): boolean {
    return this.filter_bar.mousedown(e) || super.mousedown(e);
  }

  override mouseup(e: MouseEvent): void {
    this.filter_bar.mouseup(e);
    super.mouseup(e);
  }

  override setClicking(clicking: boolean): void {
    super.setClicking(clicking);
    if (!clicking) {
      this.filter_bar.setClicking(false);
      this.filter_bar.setHovering(false);
    }
  }

  override yi(): number {
    return super.yi() - ORDER_FILTER_BAR_H;
  }

  override h(): number {
    return super.h() + ORDER_FILTER_BAR_H;
  }

  override setAllSizes(size: number, p: Point2D, w: number, h: number): void {
    super.setAllSizes(size, { x: p.x, y: p.y + ORDER_FILTER_BAR_H }, w, h - ORDER_FILTER_BAR_H);
    this.filter_bar.setPosition({ x: p.x + this.getPadding(), y: this.config.scrollbar.yi() - ORDER_FILTER_BAR_H });
    this.filter_bar.setW(w - 2 * this.getPadding());
    for (const el of this.config.list) {
      el.setW(w - this.config.scrollbar.getScrollbarSize() - 2 * this.getPadding());
    }
  }
}
