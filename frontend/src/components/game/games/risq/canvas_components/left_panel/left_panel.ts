import { err } from '../../../../../../scripts/log';
import { ColorRGB } from '../../../../../../scripts/color_rgb';
import type { BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import type { CanvasComponent } from '../../../../util/canvas_components/canvas_component';
import { configDraw } from '../../../../util/canvas_components/canvas_component';
import { drawRect } from '../../../../util/canvas_util';
import type { Point2D } from '../../../../util/objects2d';
import { RisqVisibilityLevel } from '../../model/types';
import { unitsByPlayerFiltered } from '../../model/unit_groups';
import type { DwgRisq } from '../../risq';
import { ROW_H as ORDER_ROW_H } from '../order_row/order_row';
import { RisqOrdersList } from '../right_panel/orders_list';
import type { RisqActionButton } from './actions/action_button';
import { buildPanelActions } from './actions/action_factory';
import { drawBuilding, drawFoundation, drawResource } from './content/entity_views';
import type { PanelDrawContext } from './content/primitives';
import { RisqSpaceHexagon } from './content/space_hexagon';
import { RisqSpaceView, drawRegion } from './content/space_view';
import { RisqStatsView } from './content/stats_view';
import { drawUnit, drawUnitsGeneric } from './content/unit_views';
import { drawZone } from './content/zone_view';
import { RisqLeftPanelButton } from './controls/left_panel_close';
import type { RisqTargetPriorityControl } from './controls/target_priority_control';
import type { LeftPanelLayout } from './layout';
import {
  BUILDING_ACTION_GRID_ROWS,
  PANEL_PADDING,
  UNIT_ACTION_GRID_ROWS,
  computeLayout,
  emptyLayout,
  gridCell,
} from './layout';
import type { LeftPanelConfig, LeftPanelData } from './left_panel_data';
import { LeftPanelDataType } from './left_panel_data';
import { RisqLeftPanelInput } from './panel_input';
import {
  hasMilitary,
  hasVillager,
  isOnlyGarrisoned,
  isOnlyMilitary,
  isOnlyVillagers,
  isOrderable,
  isUnitSelection,
  normalizeSelection,
  orderListSubjects,
  unitResolver,
} from './selection_queries';

const UNIT_DATA_TYPES = [
  LeftPanelDataType.MULTIPLE_PLAYERS_UNITS,
  LeftPanelDataType.UNITS,
  LeftPanelDataType.UNITS_BY_TYPE,
  LeftPanelDataType.ECONOMIC_UNITS,
  LeftPanelDataType.MILITARY_UNITS,
  LeftPanelDataType.UNIT,
];

/** Selection details panel: shows the current selection's content, its action buttons, and its orders */
export class RisqLeftPanel implements CanvasComponent {
  private close_button: RisqLeftPanelButton;
  private order_rows_list: RisqOrdersList;

  private risq: DwgRisq;
  private config: LeftPanelConfig;
  private size: Point2D = { x: 0, y: 0 };
  private showing = false;
  private hovering = false;
  private visibility?: number;
  private data?: LeftPanelData;
  private buttons: RisqActionButton[] = [];
  private target_priority_control?: RisqTargetPriorityControl;
  private layout: LeftPanelLayout = emptyLayout();
  private stats = new RisqStatsView();
  private hexagon = new RisqSpaceHexagon();
  private space_view = new RisqSpaceView((economic) => this.openSpaceUnitsRow(economic));
  private input: RisqLeftPanelInput;
  // hover rects are only laid out by draw, so a hover check right after openPanel uses stale rects
  private hover_stale = false;

  constructor(risq: DwgRisq, config: LeftPanelConfig) {
    this.risq = risq;
    if (config.w < 1) {
      config.w = 120;
    }
    this.config = config;
    this.close_button = new RisqLeftPanelButton(risq);
    this.order_rows_list = new RisqOrdersList(
      risq,
      config.w,
      new ColorRGB(222, 184, 135).addColor(255, 0, 0, 0.2),
      false
    );
    this.input = new RisqLeftPanelInput(risq, this, this.hexagon, this.space_view, this.stats);
    this.resolveSize();
  }

  private shownData(): LeftPanelData | undefined {
    return this.showing ? this.data : undefined;
  }

  private refreshActionButtons() {
    this.buttons = [];
    this.target_priority_control = undefined;
    this.refreshOrderRows();
    if (!this.isOrderable()) {
      this.resolveSize();
      return;
    }
    const actions = buildPanelActions(this.risq, this.data);
    this.buttons = actions.buttons;
    this.target_priority_control = actions.target_priority_control;
    this.resolveSize();
    for (const button of this.buttons) {
      button.dataRefreshed();
    }
    this.target_priority_control?.dataRefreshed();
  }

  private showOrderRows(): boolean {
    return orderListSubjects(this.data, this.risq.getPlayer()?.player.player_id).kind !== undefined;
  }

  private refreshOrderRows() {
    // an empty list never matches a real internal_id, so an unowned/non-orderable selection just shows an empty list
    const subjects = orderListSubjects(this.data, this.risq.getPlayer()?.player.player_id);
    this.order_rows_list.setSubject(subjects.ids, subjects.kind);
    this.order_rows_list.refresh();
    const player = this.risq.getPlayer();
    if (this.risq.session.givingOrders() && !!player && !player.orders_submitted) {
      this.order_rows_list.enable();
    } else {
      this.order_rows_list.disable();
    }
  }

  private statsSectionEnd(): number {
    const building = this.data?.data_type === LeftPanelDataType.BUILDING ? this.data.data : undefined;
    const has_stats = this.data?.data_type === LeftPanelDataType.UNIT || (!!building && !building.under_construction);
    if (!has_stats && !this.isUnit()) {
      return this.yi() + 0.5 * this.size.y - PANEL_PADDING;
    }
    return this.yi() + 0.25 * this.size.y + 6 + RisqStatsView.height(this.risq, building);
  }

  resolveSize() {
    const h = Math.min(4 * this.config.w, this.risq.viewport.canvasSize().height);
    this.size = { x: h / 3, y: h };
    this.close_button.setPosition({ x: this.size.x, y: this.yi() + 0.5 * this.close_button.h() });
    const action_rows = this.isUnit() ? UNIT_ACTION_GRID_ROWS : BUILDING_ACTION_GRID_ROWS;
    const garrison_capacity =
      this.isBuilding() && this.data?.data_type === LeftPanelDataType.BUILDING ? this.data.data.garrison_capacity : 0;
    this.layout = computeLayout(this, this.statsSectionEnd(), action_rows, garrison_capacity);
    const s = this.layout.grid_s;
    for (const button of this.buttons) {
      button.setSize(s, s);
      button.setPosition(gridCell(this.layout, button.row, button.col));
    }
    this.target_priority_control?.setSize(3 * s + 2 * PANEL_PADDING, s);
    this.target_priority_control?.setPosition(gridCell(this.layout, 2, 2));
    const P = PANEL_PADDING;
    const min_orders_height = 2 * this.order_rows_list.getPadding() + 2 * ORDER_ROW_H + this.order_rows_list.getGap();
    const orders_y0 = Math.min(this.layout.grid_bottom_separator + P, this.yi() + this.size.y - P - min_orders_height);
    this.order_rows_list.setAllSizes(
      Math.min(0.1 * this.w(), 16),
      { x: this.xi() + P, y: orders_y0 },
      this.w() - 2 * P,
      this.yi() + this.size.y - orders_y0 - P
    );
  }

  isHovering(): boolean {
    return this.hovering;
  }

  setHovering(hovering: boolean): void {
    this.hovering = hovering;
  }

  isClicking(): boolean {
    return false;
  }

  setClicking(_clicking: boolean): void {}

  isShowing(): boolean {
    return this.showing;
  }

  getData(): LeftPanelData | undefined {
    return this.data;
  }

  dataRefreshed() {
    this.refreshActionButtons();
  }

  isOrderable(): boolean {
    return isOrderable(this.shownData(), this.visibility, this.risq.getPlayer()?.player.player_id ?? -1);
  }

  isUnit(): boolean {
    return isUnitSelection(this.shownData());
  }

  hasVillager(): boolean {
    return hasVillager(this.shownData());
  }

  hasMilitary(): boolean {
    return hasMilitary(this.shownData());
  }

  isOnlyVillagers(): boolean {
    return isOnlyVillagers(this.shownData());
  }

  isOnlyMilitary(): boolean {
    return isOnlyMilitary(this.shownData());
  }

  isOnlyGarrisoned(): boolean {
    return isOnlyGarrisoned(this.shownData(), unitResolver(this.risq));
  }

  isBuilding(): boolean {
    return this.shownData()?.data_type === LeftPanelDataType.BUILDING;
  }

  close() {
    this.showing = false;
    this.visibility = undefined;
    this.data = undefined;
    this.hexagon.clearHoveredZone();
    this.stats.clearHover();
    this.buttons = [];
  }

  openPanel(open_data: LeftPanelData, visibility: number) {
    if (visibility < RisqVisibilityLevel.FOG) {
      return; // not explored
    }
    if (visibility < RisqVisibilityLevel.GOOD && UNIT_DATA_TYPES.includes(open_data.data_type)) {
      return; // units not individually identifiable yet
    }
    this.risq.armed.disarmAll();
    this.input.reset();
    this.hover_stale = true;
    this.visibility = visibility;
    this.data = open_data;
    this.showing = true;
    const normalized = normalizeSelection(open_data, unitResolver(this.risq));
    if (normalized) {
      this.data = normalized;
    } else {
      this.close();
    }
    this.refreshActionButtons();
    this.risq.pointer.recalculate();
  }

  /** Opens the shown space's villagers or military units as a unit selection */
  private openSpaceUnitsRow(economic: boolean): void {
    if (this.data?.data_type !== LeftPanelDataType.SPACE) {
      return;
    }
    const space = this.data.data;
    this.openPanel(
      {
        data_type: LeftPanelDataType.UNITS,
        data: { space, units_by_player: unitsByPlayerFiltered(space.units ?? new Map(), economic) },
      },
      this.visibility ?? 0
    );
  }

  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number) {
    if (!this.isShowing()) {
      return;
    }
    const pc: PanelDrawContext = {
      ctx,
      risq: this.risq,
      frame: this,
      layout: this.layout,
      visibility: this.visibility ?? 0,
      stats: this.stats,
    };
    configDraw(
      ctx,
      transform,
      { fill_style: this.config.background, stroke_style: 'transparent', stroke_width: 0, fixed_position: true },
      false,
      false,
      () => {
        drawRect(ctx, { x: this.xi(), y: this.yi() }, this.w(), this.h());
        ctx.save();
        ctx.beginPath();
        ctx.rect(this.xi(), this.yi(), this.w(), this.h());
        ctx.clip();
        this.drawContent(pc);
        ctx.restore();
      }
    );
    const data = this.data;
    const stats_subject =
      data?.data_type === LeftPanelDataType.UNIT || data?.data_type === LeftPanelDataType.BUILDING
        ? data.data
        : undefined;
    this.stats.drawTooltips(ctx, transform, this.risq, dt, stats_subject);
    if (data?.data_type === LeftPanelDataType.SPACE || data?.data_type === LeftPanelDataType.ZONE) {
      const space = data.data_type === LeftPanelDataType.SPACE ? data.data : data.data.space;
      this.hexagon.drawTooltip(ctx, transform, this.risq, dt, space);
    }
    ctx.beginPath();
    for (const button of this.buttons) {
      button.draw(ctx, transform, dt);
      button.drawTooltip(ctx, transform, this.risq, dt);
    }
    this.target_priority_control?.draw(ctx, transform);
    if (data?.data_type === LeftPanelDataType.SPACE && (this.visibility ?? 0) > RisqVisibilityLevel.POOR) {
      this.space_view.drawRowButtons(ctx, transform, dt);
    }
    if (this.showOrderRows()) {
      this.order_rows_list.draw(ctx, transform, dt);
    }
    ctx.beginPath();
    this.close_button.draw(ctx, transform, dt);
    if (this.hover_stale) {
      this.hover_stale = false;
      this.risq.pointer.recalculate();
    }
  }

  private drawContent(pc: PanelDrawContext) {
    switch (this.data?.data_type) {
      case LeftPanelDataType.RESOURCE:
        drawResource(pc, this.data.data);
        break;
      case LeftPanelDataType.BUILDING:
        drawBuilding(pc, this.data.data);
        break;
      case LeftPanelDataType.SPACE:
        this.space_view.draw(pc, this.data.data, this.hexagon);
        break;
      case LeftPanelDataType.REGION:
        drawRegion(pc, this.data.data);
        break;
      case LeftPanelDataType.ZONE:
        drawZone(pc, this.data.data, this.hexagon);
        break;
      case LeftPanelDataType.MULTIPLE_PLAYERS_UNITS:
        drawUnitsGeneric(pc, this.data.data.units_by_player);
        break;
      case LeftPanelDataType.UNITS_BY_TYPE:
      case LeftPanelDataType.ECONOMIC_UNITS:
      case LeftPanelDataType.MILITARY_UNITS:
        if (this.data.data.units.length > 0) {
          drawUnitsGeneric(pc, [[this.data.data.units[0].player_id, this.data.data.units]]);
        }
        break;
      case LeftPanelDataType.UNITS:
        err('LeftPanelDataType.UNITS should have been converted by normalizeSelection', this.data);
        break;
      case LeftPanelDataType.UNIT:
        drawUnit(pc, this.data.data);
        break;
      case LeftPanelDataType.FOUNDATION:
        drawFoundation(pc, this.data.data);
        break;
      default:
        err('Unknown data type for left panel', this.data);
        break;
    }
  }

  scroll(dy: number, mode: number): boolean {
    if (this.showOrderRows() && this.order_rows_list.isHovering()) {
      return this.order_rows_list.scroll(dy, mode);
    }
    return false;
  }

  mousemove(canvas: Point2D, screen: Point2D, transform: BoardTransformData): boolean {
    if (this.close_button.mousemove(canvas, screen, transform)) {
      return true;
    }
    this.hovering = screen.x > this.xi() && screen.y > this.yi() && screen.x < this.xf() && screen.y < this.yf();
    for (const button of this.buttons) {
      button.mousemove(canvas, screen, transform);
    }
    this.target_priority_control?.mousemove(screen);
    if (this.showOrderRows()) {
      this.order_rows_list.mousemove(canvas, screen, transform);
    }
    this.input.mousemove(this.data, canvas, screen, transform);
    return this.isHovering();
  }

  mousedown(e: MouseEvent): boolean {
    if (this.close_button.mousedown(e)) {
      return true;
    }
    this.input.mousedown(this.data, e);
    for (const button of this.buttons) {
      button.mousedown(e);
    }
    this.target_priority_control?.mousedown(e);
    if (this.showOrderRows()) {
      this.order_rows_list.mousedown(e);
    }
    return this.isHovering();
  }

  mouseup(e: MouseEvent) {
    this.close_button.mouseup(e);
    if (this.showOrderRows()) {
      this.order_rows_list.mouseup(e);
    }
    this.input.mouseup(this.data, this.visibility ?? 0, e);
    for (const button of this.buttons) {
      button.mouseup(e);
    }
    this.target_priority_control?.mouseup(e);
  }

  xi(): number {
    return 0;
  }
  yi(): number {
    return 0.5 * (this.risq.viewport.canvasSize().height - this.size.y);
  }
  xf(): number {
    return this.showing ? this.xi() + this.w() : 0;
  }
  yf(): number {
    return this.showing ? this.yi() + this.h() : 0;
  }
  xc(): number {
    return this.xi() + 0.5 * this.w();
  }
  yc(): number {
    return this.yi() + 0.5 * this.h();
  }
  w(): number {
    return this.size.x;
  }
  h(): number {
    return this.size.y;
  }
}
