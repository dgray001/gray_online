import type { BoardTransformData, DwgCanvasBoard, ModifierKeys } from '../../../../util/canvas_board/canvas_board';
import type { CanvasComponent } from '../../../../util/canvas_components/canvas_component';
import type { Point2D } from '../../../../util/objects2d';
import { normalizeRect } from '../../../../util/objects2d';
import type { RisqLeftPanel } from '../../canvas_components/left_panel/left_panel';
import { LeftPanelDataType } from '../../canvas_components/left_panel/left_panel_data';
import type { RisqSpace, RisqZone } from '../../model/types';
import { RisqOrderType } from '../../model/types';
import type { RisqViewport } from '../../rendering/board/viewport';
import { isEmptyPlot } from '../../rendering/zones/draw';
import type { OrderSubject, RisqOrderDispatch } from '../orders/dispatch';
import type { RisqOrderPlanning } from '../orders/planning';
import type { RisqBoardClicks } from '../selection/board_clicks';
import type { RisqSession } from '../session';
import type { RisqArmedState } from './armed_state';
import type { RisqCursorController } from './cursor_controller';
import type { RisqHover } from './hover';

const DRAG_SELECT_THRESHOLD = 3;

/** Routes board mouse events between the overlay components, hover, order dispatch, and click/drag selection */
export class RisqPointer {
  private armed_click = false;
  private dispatching_mousemove = false;
  private dragging_selection = false;
  private drag_additive = false;
  private drag_start: Point2D = { x: 0, y: 0 };
  private drag_current: Point2D = { x: 0, y: 0 };

  constructor(
    private board: () => DwgCanvasBoard | undefined,
    private components: CanvasComponent[],
    private panels_hovered: () => boolean,
    private session: RisqSession,
    private viewport: RisqViewport,
    private hover: RisqHover,
    private armed: RisqArmedState,
    private cursor: RisqCursorController,
    private left_panel: RisqLeftPanel,
    private planning: RisqOrderPlanning,
    private dispatch: RisqOrderDispatch,
    private board_clicks: RisqBoardClicks,
    private hud_scale: () => number
  ) {}

  /** Screen-space rectangle of an in-progress drag selection */
  dragRect(): { min: Point2D; max: Point2D } | undefined {
    return this.dragging_selection ? normalizeRect(this.drag_start, this.drag_current) : undefined;
  }

  private dragged(): boolean {
    return (
      Math.hypot(this.drag_current.x - this.drag_start.x, this.drag_current.y - this.drag_start.y) >
      DRAG_SELECT_THRESHOLD
    );
  }

  /** Re-resolves hover at the last mouse position, e.g. after panels or data change underneath it */
  recalculate(): void {
    if (this.dispatching_mousemove) {
      return;
    }
    const board = this.board();
    if (!board || !board.isInitialized()) {
      return;
    }
    this.mousemove(
      this.hover.mouseCanvas(),
      this.hover.mouseScreen(),
      this.viewport.lastTransform(),
      board.getModifiers()
    );
  }

  // scroll() signature already used by HTMLElement
  scroll(dy: number, mode: number): boolean {
    for (const component of this.components) {
      if (component.isHovering()) {
        if (component.scroll?.(dy, mode)) {
          this.recalculate();
        }
        return true;
      }
    }
    return false;
  }

  mousemove(m: Point2D, screen: Point2D, transform: BoardTransformData, modifiers: ModifierKeys): void {
    const board = this.board();
    if (!this.session.getGame() || !board) {
      return;
    }
    this.viewport.updateDrawDetail(transform.scale, board.getMaxScale());
    this.hover.setMouse(m, screen);
    if (this.dragging_selection) {
      this.drag_current = screen;
      if (this.dragged()) {
        this.hover.clearClicks();
      }
      return;
    }
    const hud_screen = { x: screen.x / this.hud_scale(), y: screen.y / this.hud_scale() };
    let hovered_other_component = false;
    this.dispatching_mousemove = true;
    try {
      for (const component of this.components) {
        hovered_other_component = component.mousemove(m, hud_screen, transform) || hovered_other_component;
      }
    } finally {
      this.dispatching_mousemove = false;
    }
    board.setPanSuppressed(false, this.panels_hovered());
    if (this.hover.update(hovered_other_component)) {
      this.cursor.update(modifiers.ctrl);
    } else {
      this.cursor.setDefault();
    }
  }

  draggingCallback(): void {
    this.hover.dragging();
  }

  cancelInput(): void {
    this.dragging_selection = false;
    this.hover.clearClicks();
    this.hover.leave();
    for (const component of this.components) {
      component.setClicking(false);
      component.setHovering(false);
    }
    this.cursor.setDefault();
  }

  mouseleave(): void {
    this.cancelInput();
  }

  // returns false if mousedown event should initiate dragging
  mousedown(e: MouseEvent): boolean {
    this.armed_click = false;
    if (this.components.map((c) => c.mousedown(e)).some(Boolean)) {
      return true;
    }
    if (this.armed.anythingArmed() && (e.button === 0 || e.button === 2)) {
      this.armed_click = true;
      if (e.button === 2) {
        this.armed.disarmAll();
      } else if (this.session.canGiveOrders()) {
        if (this.armed.getArmedOrder() === RisqOrderType.OrderType_BuyMercenary) {
          this.dispatch.placeMercenary(e.ctrlKey);
        } else {
          this.orderSelection(e.ctrlKey);
        }
      }
      return true;
    }
    if (e.button === 0) {
      this.pressBoard();
    } else if (e.button === 2 && this.left_panel.isOrderable() && this.session.canGiveOrders()) {
      this.orderSelection(e.ctrlKey);
      return true;
    }
    const armed = this.armed.anythingArmed();
    const drag_select = this.viewport.zoneView() ? !e.shiftKey : e.shiftKey;
    if (drag_select && e.button === 0 && !armed) {
      this.dragging_selection = true;
      this.drag_additive = e.ctrlKey;
      this.drag_start = { ...this.hover.mouseScreen() };
      this.drag_current = this.drag_start;
      return true;
    }
    return (e.button === 2 && armed) || (e.button !== 0 && e.button !== 2);
  }

  /** Marks the pressed space, zone, or zone part as clicked so mouseup can act on it if still hovered */
  private pressBoard(): void {
    const space = this.hover.space();
    const zone = this.hover.zone();
    if (!space || space.visibility <= 0) {
      return;
    }
    space.clicked = true;
    if (!this.viewport.zoneView() || !zone) {
      return;
    }
    space.clicked = false;
    zone.clicked = true;
    for (const [i, part] of zone.hovered_data.entries()) {
      if (part.hovered && !(i === 0 && isEmptyPlot(this.planning, zone))) {
        part.clicked = true;
        zone.clicked = false;
        break;
      }
    }
  }

  private orderSelection(ctrl_held: boolean): void {
    const data = this.left_panel.getData();
    switch (data?.data_type) {
      case LeftPanelDataType.UNIT:
        this.dispatch.unitOrder([data.data], ctrl_held);
        break;
      case LeftPanelDataType.UNITS_BY_TYPE:
      case LeftPanelDataType.ECONOMIC_UNITS:
      case LeftPanelDataType.MILITARY_UNITS: {
        const units: OrderSubject[] = data.data.units.flatMap((u) =>
          [...u.units].map((internal_id) => ({ unit_type: u.unit_type, internal_id }))
        );
        this.dispatch.unitOrder(units, ctrl_held);
        break;
      }
      case LeftPanelDataType.BUILDING:
        this.dispatch.buildingOrder([data.data], ctrl_held);
        break;
      case LeftPanelDataType.BUILDINGS:
        this.dispatch.buildingOrder(data.data.buildings, ctrl_held);
        break;
      default:
        break;
    }
  }

  mouseup(e: MouseEvent): void {
    const armed_before = this.armed.anythingArmed();
    for (const component of this.components) {
      component.mouseup(e);
    }
    if (this.armed_click) {
      this.armed_click = false;
      return;
    }
    if (this.dragging_selection) {
      this.dragging_selection = false;
      if (this.dragged()) {
        this.board_clicks.dragSelect(this.drag_start, this.drag_current, this.drag_additive);
        return;
      }
    }
    const space = this.hover.space();
    const zone = this.hover.zone();
    if (e.button === 0 && !!space && space.visibility > 0 && !armed_before) {
      this.releaseBoard(space, zone, e);
    } else if (e.button === 0 && !armed_before && !this.panels_hovered()) {
      this.left_panel.close();
    }
  }

  private releaseBoard(space: RisqSpace, zone: RisqZone | undefined, e: MouseEvent): void {
    if (!!zone) {
      if (this.viewport.zoneView()) {
        this.board_clicks.zoneClick(space, zone, e);
      }
      zone.clicked = false;
      for (const part of zone.hovered_data) {
        part.clicked = false;
      }
    } else {
      this.board_clicks.spaceClick(space);
    }
    space.clicked = false;
  }
}
