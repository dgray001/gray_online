import type { ColorRGB } from '../../../../../../scripts/color_rgb';
import type { BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import type { CanvasComponent, Rotation } from '../../../../util/canvas_components/canvas_component';
import { configDraw } from '../../../../util/canvas_components/canvas_component';
import { drawRect, drawText } from '../../../../util/canvas_util';
import type { Point2D } from '../../../../util/objects2d';
import type { DwgRisq } from '../../risq';
import type { RisqResourceType } from '../../model/types';
import { RisqOrdersList } from './orders_list';
import { RisqNextIdleButton } from './next_idle_button';
import { RisqOrdersExpandButton, RisqReplayButton, RisqRightPanelButton } from './right_panel_button';
import { RisqSubmitIconButton, RisqSubmitOrdersButton } from './submit_button';
import { RisqSummaryReportButton } from '../bottom_panel/bottom_panel_buttons';
import { RisqButtonTooltip } from '../risq_tooltip';
import type { RightPanelGeometry } from './resource_tooltip';
import {
  SCORE_ROW_H,
  drawPopulationRow,
  drawResourceRow,
  drawScoreRow,
  drawSeparator,
  populationTooltipData,
  resourceRowAt,
  resourceTooltipData,
} from './resource_tooltip';

export declare interface RightPanelConfig {
  w: number;
  is_open: boolean;
  background: ColorRGB;
}

export class RisqRightPanel implements CanvasComponent {
  private static PADDING = 5;
  private static SUBMIT_SIZE = 30;
  private static WORKERS_W = 70;

  private open_button: RisqRightPanelButton;
  private orders_list: RisqOrdersList;
  private orders_expand_button: RisqOrdersExpandButton;
  private submit_button: RisqSubmitOrdersButton;
  private submit_icon_button: RisqSubmitIconButton;
  private next_idle_button: RisqNextIdleButton;
  private report_button: RisqSummaryReportButton;
  private replay_button: RisqReplayButton;
  private submit_row_y = 0;

  private risq: DwgRisq;
  private config: RightPanelConfig;
  private opening = false;
  private orders_expanded = false;
  private has_idle = false;
  private hovering = false;
  private resource_row_ys = new Map<RisqResourceType, number>();
  private resource_tooltips = new Map<RisqResourceType, RisqButtonTooltip>();
  private hovered_resource?: RisqResourceType;
  private population_row_y?: number;
  private hovered_population = false;
  private population_tooltip = new RisqButtonTooltip(() => populationTooltipData(this.risq));

  constructor(risq: DwgRisq, config: RightPanelConfig) {
    this.risq = risq;
    this.config = config;
    this.open_button = new RisqRightPanelButton(risq);
    this.orders_list = new RisqOrdersList(
      risq,
      config.w - 2 * RisqRightPanel.PADDING,
      config.background.copy().addColor(255, 0, 0, 0.2)
    );
    this.orders_expand_button = new RisqOrdersExpandButton(risq);
    this.submit_button = new RisqSubmitOrdersButton(risq, RisqRightPanel.SUBMIT_SIZE);
    this.submit_icon_button = new RisqSubmitIconButton(risq, RisqRightPanel.SUBMIT_SIZE);
    this.next_idle_button = new RisqNextIdleButton(risq, RisqRightPanel.SUBMIT_SIZE);
    this.report_button = new RisqSummaryReportButton(risq);
    this.replay_button = new RisqReplayButton(risq);
    this.toggle(config.is_open, true);
  }

  dataRefreshed(): void {
    const player = this.risq.getPlayer();
    this.has_idle =
      !!player &&
      this.risq.session.givingOrders() &&
      !player.orders_submitted &&
      this.risq.planning.idleOrderableCount() > 0;
    this.report_button.dataRefreshed();
    this.replay_button.dataRefreshed();
    this.orders_list.refresh();
    this.resolveSize();
    if (!player || !this.risq.session.givingOrders()) {
      this.orders_list.disable();
      this.submit_button.disable();
      this.submit_icon_button.disable();
      this.next_idle_button.disable();
      return;
    }
    if (this.risq.session.givingOrders() && !player.orders_submitted) {
      this.orders_list.enable();
      this.submit_button.setText(this.risq.submission.submittedTimes() > 0 ? 'Resubmit Orders' : 'Submit Orders');
    } else {
      this.orders_list.disable();
      this.submit_button.setText('Unsubmit Orders');
    }
    this.next_idle_button.setText(`Next Idle (${this.risq.planning.idleOrderableCount()})`);
    if (this.hasIdleOrderables()) {
      this.submit_icon_button.enable();
      this.next_idle_button.enable();
      this.submit_button.disable();
    } else {
      this.submit_button.enable();
      this.submit_icon_button.disable();
      this.next_idle_button.disable();
    }
  }

  private hasIdleOrderables(): boolean {
    return this.has_idle;
  }

  resolveSize(): void {
    const player = this.risq.getPlayer();
    const game = this.risq.getGame();
    const padding = RisqRightPanel.PADDING;
    const yi = this.orders_expanded
      ? this.yi() + padding
      : this.yi() +
        40 +
        4 * padding +
        (player ? 30 * (1 + player.resources.size) + 2 * padding : 0) +
        SCORE_ROW_H * (game?.scores.length ?? 0);
    const submit_height = game?.game_base.game_ended ? 0 : RisqRightPanel.SUBMIT_SIZE + padding;
    const orders_height = this.yf() - yi - padding - submit_height;
    this.orders_list.setAllSizes(
      Math.min(0.1 * this.paddedW(), 20),
      { x: this.xi() + padding, y: yi },
      this.w() - 2 * padding,
      orders_height
    );
    this.orders_expand_button.setPosition({
      x: this.orders_list.xf() - this.orders_expand_button.w() - 4,
      y: yi + 4,
    });
    this.submit_row_y = yi + orders_height + padding;
    this.report_button.setPosition({ x: this.xf() - padding - 30, y: this.yi() + padding });
    this.replay_button.setPosition({ x: this.report_button.xi() - padding - 30, y: this.yi() + padding });
    this.layoutButtons();
  }

  private layoutButtons(): void {
    const y = this.submit_row_y;
    const left = this.xi() + RisqRightPanel.PADDING;
    if (this.hasIdleOrderables()) {
      const icon_w = RisqRightPanel.SUBMIT_SIZE;
      const idle_w = this.w() - 2 * RisqRightPanel.PADDING - icon_w - RisqRightPanel.PADDING;
      this.next_idle_button.setPosition({ x: left, y });
      this.next_idle_button.setW(idle_w);
      this.submit_icon_button.setPosition({ x: left + idle_w + RisqRightPanel.PADDING, y });
    } else {
      this.submit_button.setPosition({ x: left, y });
      this.submit_button.setW(this.w() - 2 * RisqRightPanel.PADDING);
    }
  }

  isHovering(): boolean {
    return this.hovering;
  }

  setHovering(hovering: boolean): void {
    this.hovering = hovering;
    if (!hovering) {
      this.hovered_resource = undefined;
      this.hovered_population = false;
    }
  }

  isClicking(): boolean {
    return false;
  }

  setClicking(clicking: boolean): void {
    if (!clicking) {
      for (const component of [
        this.open_button,
        this.orders_list,
        this.orders_expand_button,
        this.submit_button,
        this.submit_icon_button,
        this.next_idle_button,
        this.report_button,
        this.replay_button,
      ]) {
        component.setClicking(false);
        component.setHovering(false);
      }
    }
  }

  toggleOrdersExpanded(expanded?: boolean): void {
    this.orders_expanded = expanded ?? !this.orders_expanded;
    this.orders_expand_button.setExpanded(this.orders_expanded);
    this.hovered_resource = undefined;
    this.hovered_population = false;
    this.resolveSize();
  }

  isOpen(): boolean {
    return this.config.is_open;
  }

  toggle(open?: boolean, initial?: boolean): void {
    this.report_button.setClicking(false);
    this.report_button.setHovering(false);
    this.replay_button.setClicking(false);
    this.replay_button.setHovering(false);
    this.orders_expand_button.setClicking(false);
    this.orders_expand_button.setHovering(false);
    this.config.is_open = open ?? !this.config.is_open;
    const position: Point2D = {
      x: this.risq.hudCanvasSize().width - this.open_button.w(),
      y: 0.5 * this.open_button.h(),
    };
    if (this.config.is_open) {
      position.x -= this.config.w;
    }
    this.opening = true;
    this.open_button.setPosition(
      position,
      () => {
        this.opening = false;
        const rotation: Rotation = {
          direction: this.config.is_open,
          angle: this.config.is_open ? 0.5 * Math.PI : -0.5 * Math.PI,
        };
        this.open_button.setRotation(rotation, undefined, initial);
        this.resolveSize();
      },
      initial
    );
  }

  submittingOrders(): void {
    this.orders_list.disable();
    this.submit_button.disable();
    this.submit_button.setText('Submitting Orders ...');
  }

  unsubmittingOrders(): void {
    this.orders_list.disable();
    this.submit_button.disable();
    this.submit_button.setText('Unsubmitting Orders ...');
  }

  prepareFrame(dt: number): boolean {
    const opening = this.opening;
    const moved = this.open_button.advanceAnimation(dt);
    return moved || opening !== this.opening;
  }

  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void {
    if (this.opening || this.config.is_open) {
      configDraw(
        ctx,
        transform,
        {
          fill_style: this.config.background.getString(),
          stroke_style: 'transparent',
          stroke_width: 0,
          fixed_position: true,
        },
        false,
        false,
        () => {
          drawRect(ctx, { x: this.xi(), y: this.yi() }, this.w(), this.h());
          if (this.opening || this.orders_expanded) {
            return;
          }
          let yi = this.yi() + RisqRightPanel.PADDING;
          const player = this.risq.getPlayer();
          const game = this.risq.getGame();
          drawText(ctx, game?.game_base.game_ended ? 'Game Over' : `Turn ${game?.turn_number ?? '??'}`, {
            p: { x: this.xi() + 0.15 * this.w(), y: yi },
            w: this.replay_button.xi() - RisqRightPanel.PADDING - (this.xi() + 0.15 * this.w()),
            fill_style: 'black',
            align: 'left',
            font: 'bold 36px serif',
          });
          yi += 40;
          drawSeparator(ctx, this.geom(), yi);
          yi += RisqRightPanel.PADDING;
          ctx.save();
          ctx.beginPath();
          ctx.rect(this.xi(), yi, this.w(), this.yf() - yi);
          ctx.clip();
          if (!!player) {
            ctx.font = '24px serif';
            this.population_row_y = yi;
            drawPopulationRow(
              ctx,
              this.risq,
              this.geom(),
              yi,
              player.units.size,
              this.risq.planning.pendingMercenaryCount(),
              player.unlimited_population ? undefined : player.population_limit
            );
            yi += 30;
            this.resource_row_ys.clear();
            for (const [rt, pr] of [...player.resources.entries()].sort((a, b) => a[0] - b[0])) {
              this.resource_row_ys.set(rt, yi);
              drawResourceRow(
                ctx,
                this.risq,
                this.geom(),
                yi,
                rt,
                pr,
                RisqRightPanel.WORKERS_W,
                RisqRightPanel.PADDING
              );
              yi += 30;
            }
            yi += RisqRightPanel.PADDING;
            drawSeparator(ctx, this.geom(), yi);
            yi += RisqRightPanel.PADDING;
          }
          for (const score of game?.scores ?? []) {
            drawScoreRow(ctx, this.geom(), yi, score);
            yi += SCORE_ROW_H;
          }
          ctx.restore();
          yi += RisqRightPanel.PADDING;
          drawSeparator(ctx, this.geom(), yi);
          yi += RisqRightPanel.PADDING;
        }
      );
    }
    if (this.config.is_open && !this.opening) {
      if (!this.orders_expanded) {
        this.report_button.draw(ctx, transform, dt);
        this.report_button.drawTooltip(ctx, transform, this.risq, dt);
        this.replay_button.draw(ctx, transform, dt);
        this.replay_button.drawTooltip(ctx, transform, this.risq, dt);
        this.population_tooltip.draw(ctx, transform, this.risq, dt, this.hovered_population, false);
        for (const rt of this.resource_row_ys.keys()) {
          this.resourceTooltip(rt).draw(ctx, transform, this.risq, dt, this.hovered_resource === rt, false);
        }
      }
      this.orders_list.draw(ctx, transform, dt);
      this.orders_expand_button.draw(ctx, transform, dt);
      this.orders_expand_button.drawTooltip(ctx, transform, this.risq, dt);
      if (this.hasIdleOrderables()) {
        this.next_idle_button.draw(ctx, transform, dt);
        this.next_idle_button.drawTooltip(ctx, transform, this.risq, dt);
        this.submit_icon_button.draw(ctx, transform, dt);
        this.submit_icon_button.drawTooltip(ctx, transform, this.risq, dt);
      } else if (!this.risq.getGame()?.game_base.game_ended) {
        this.submit_button.draw(ctx, transform, dt);
        this.submit_button.drawTooltip(ctx, transform, this.risq, dt);
      }
    }
    this.open_button.draw(ctx, transform, dt);
  }

  private resourceTooltip(rt: RisqResourceType): RisqButtonTooltip {
    let tooltip = this.resource_tooltips.get(rt);
    if (!tooltip) {
      tooltip = new RisqButtonTooltip(() => resourceTooltipData(this.risq, rt));
      this.resource_tooltips.set(rt, tooltip);
    }
    return tooltip;
  }

  scroll(dy: number, mode: number): boolean {
    return this.orders_list.scroll(dy, mode);
  }

  mousemove(canvas: Point2D, screen: Point2D, transform: BoardTransformData): boolean {
    const toggle_hovering = this.open_button.mousemove(canvas, screen, transform);
    const inside = screen.x > this.xi() && screen.x < this.xf() && screen.y > this.yi() && screen.y < this.yf();
    for (const [component, visible] of [
      [this.report_button, !this.orders_expanded],
      [this.replay_button, !this.orders_expanded],
      [this.orders_expand_button, true],
      [this.orders_list, true],
      [this.next_idle_button, this.has_idle],
      [this.submit_icon_button, this.has_idle],
      [this.submit_button, !this.has_idle && !this.risq.getGame()?.game_base.game_ended],
    ] as const) {
      component.mousemove(canvas, screen, transform);
      if (!inside || !this.config.is_open || this.opening || !visible) {
        component.setHovering(false);
      }
    }
    const rows_hoverable = inside && this.config.is_open && !this.opening && !this.orders_expanded;
    this.hovered_resource = rows_hoverable ? resourceRowAt(this.resource_row_ys, screen.y) : undefined;
    const population_y = this.population_row_y;
    this.hovered_population =
      rows_hoverable && population_y !== undefined && screen.y >= population_y && screen.y < population_y + 30;
    this.hovering = toggle_hovering || ((this.config.is_open || this.opening) && inside);
    return this.isHovering();
  }

  mousedown(e: MouseEvent): boolean {
    if (this.config.is_open && !this.opening && !this.orders_expanded && this.report_button.mousedown(e)) {
      return true;
    }
    if (this.config.is_open && !this.opening && !this.orders_expanded && this.replay_button.mousedown(e)) {
      return true;
    }
    if (this.config.is_open && !this.opening && this.orders_expand_button.mousedown(e)) {
      return true;
    }
    if (this.open_button.mousedown(e)) {
      return true;
    }
    if (!this.config.is_open || this.opening || !this.isHovering()) {
      return this.isHovering();
    }
    this.orders_list.mousedown(e);
    if (this.hasIdleOrderables()) {
      this.next_idle_button.mousedown(e);
      this.submit_icon_button.mousedown(e);
    } else if (!this.risq.getGame()?.game_base.game_ended) {
      this.submit_button.mousedown(e);
    }
    return this.isHovering();
  }

  mouseup(e: MouseEvent): void {
    if (this.config.is_open && !this.opening) {
      if (!this.orders_expanded) {
        this.report_button.mouseup(e);
        this.replay_button.mouseup(e);
      }
      this.orders_expand_button.mouseup(e);
    }
    this.open_button.mouseup(e);
    this.orders_list.mouseup(e);
    if (this.hasIdleOrderables()) {
      this.next_idle_button.mouseup(e);
      this.submit_icon_button.mouseup(e);
    } else if (!this.risq.getGame()?.game_base.game_ended) {
      this.submit_button.mouseup(e);
    }
  }

  xi(): number {
    return this.open_button.xf();
  }
  yi(): number {
    return 0;
  }
  xf(): number {
    return this.risq.hudCanvasSize().width;
  }
  yf(): number {
    return this.risq.hudCanvasSize().height;
  }
  xc(): number {
    return this.xi() + 0.5 * this.w();
  }
  yc(): number {
    return this.yi() + 0.5 * this.h();
  }
  w(): number {
    return this.xf() - this.xi();
  }
  fullWidth(): number {
    return this.config.w;
  }
  paddedW(): number {
    return this.w() - 2 * RisqRightPanel.PADDING;
  }
  h(): number {
    return this.yf() - this.yi();
  }
  private geom(): RightPanelGeometry {
    return { xi: this.xi(), xf: this.xf(), w: this.w() };
  }
}
