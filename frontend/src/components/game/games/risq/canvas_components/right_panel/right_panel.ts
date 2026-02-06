import type { ColorRGB } from '../../../../../../scripts/color_rgb';
import type { BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import type { CanvasComponent, Rotation } from '../../../../util/canvas_components/canvas_component';
import { configDraw } from '../../../../util/canvas_components/canvas_component';
import { drawLine, drawRect, drawText } from '../../../../util/canvas_util';
import type { Point2D } from '../../../../util/objects2d';
import type { DwgRisq } from '../../risq';
import type { GameRisqScoreEntry, RisqFrontendOrder, RisqResourceType } from '../../risq_data';
import { resourceTypeImage } from '../../risq_resources';
import { RisqOrdersList } from './orders_list';
import { RisqRightPanelButton } from './right_panel_button';
import { RisqSubmitOrdersButton } from './submit_orders_button';

/** Config for the right panel */
export declare interface RightPanelConfig {
  w: number;
  is_open: boolean;
  background: ColorRGB;
}

export class RisqRightPanel implements CanvasComponent {
  // For use in the draw function
  private static PADDING = 5;
  private static SUBMIT_SIZE = 30;

  private open_button: RisqRightPanelButton;
  private orders_list: RisqOrdersList;
  private submit_button: RisqSubmitOrdersButton;
  private need_to_set_position = true;

  private risq: DwgRisq;
  private config: RightPanelConfig;
  private opening = false;
  private hovering = false;

  constructor(risq: DwgRisq, config: RightPanelConfig) {
    this.risq = risq;
    this.config = config;
    this.open_button = new RisqRightPanelButton(risq);
    this.orders_list = new RisqOrdersList(
      risq,
      config.w - 2 * RisqRightPanel.PADDING,
      config.background.copy().addColor(255, 0, 0, 0.2)
    );
    this.submit_button = new RisqSubmitOrdersButton(risq, RisqRightPanel.SUBMIT_SIZE);
    this.toggle(config.is_open, true);
  }

  dataRefreshed() {
    this.orders_list.setOrders(this.risq.getPlayer()?.active_orders ?? []);
    if (this.risq.givingOrders() && !this.risq.ordersSubmitted()) {
      this.orders_list.enable();
      this.submit_button.setText(this.risq.ordersSubmittedTimes() > 0 ? 'Resubmit Orders': 'Submit Orders');
    } else {
      this.orders_list.disable();
      this.submit_button.setText('Unsubmit Orders');
    }
    this.submit_button.enable();
  }

  isHovering(): boolean {
    return this.hovering;
  }

  setHovering(hovering: boolean) {
    this.hovering = hovering;
  }

  isClicking(): boolean {
    return false;
  }

  setClicking(_clicking: boolean) {}

  isOpen(): boolean {
    return this.config.is_open;
  }

  toggle(open?: boolean, initial?: boolean) {
    this.config.is_open = open ?? !this.config.is_open;
    const position: Point2D = {
      x: this.risq.canvasSize().width - this.open_button.w(),
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
        this.need_to_set_position = true;
      },
      initial
    );
  }

  addOrder(order: RisqFrontendOrder) {
    this.orders_list.addOrder(order);
  }

  getOrders(): RisqFrontendOrder[] {
    return this.orders_list.getOrders();
  }

  setOrders(orders: RisqFrontendOrder[]) {
    this.orders_list.setOrders(orders);
  }

  submittingOrders() {
    this.orders_list.disable();
    this.submit_button.disable();
    this.submit_button.setText('Submitting Orders ...');
  }

  unsubmittingOrders() {
    this.orders_list.disable();
    this.submit_button.disable();
    this.submit_button.setText('Unsubmitting Orders ...');
  }

  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number) {
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
          if (this.opening) {
            return;
          }
          let yi = this.yi() + RisqRightPanel.PADDING;
          const player = this.risq.getPlayer();
          const game = this.risq.getGame();
          drawText(ctx, `Turn ${game?.turn_number ?? '??'}`, {
            p: { x: this.xc(), y: yi },
            w: this.paddedW(),
            fill_style: 'black',
            align: 'center',
            font: 'bold 36px serif',
          });
          yi += 40;
          this.drawSeparator(ctx, yi);
          yi += RisqRightPanel.PADDING;
          if (!!player) {
            ctx.font = '24px serif';
            this.drawPopulation(ctx, yi, player.units.size, player.population_limit);
            yi += 30;
            if (!!player) {
              for (const [r, a] of [...player.resources.entries()].sort((a, b) => a[0] - b[0])) {
                this.drawResource(ctx, yi, r, a);
                yi += 30;
              }
            }
            yi += RisqRightPanel.PADDING;
            this.drawSeparator(ctx, yi);
            yi += RisqRightPanel.PADDING;
          }
          for (const score of game?.scores ?? []) {
            this.drawScore(ctx, yi, score);
            yi += 30;
          }
          yi += RisqRightPanel.PADDING;
          this.drawSeparator(ctx, yi);
          yi += RisqRightPanel.PADDING;
          // TODO: logic in case yi has gone off the rectangle
          if (this.need_to_set_position) {
            const orders_list_height = this.yf() - yi - 2 * RisqRightPanel.PADDING - RisqRightPanel.SUBMIT_SIZE;
            this.orders_list.setAllSizes(
              Math.min(0.1 * this.paddedW(), 20),
              { x: this.xi() + RisqRightPanel.PADDING, y: yi },
              this.w() - 2 * RisqRightPanel.PADDING,
              orders_list_height,
            );
            yi += orders_list_height + RisqRightPanel.PADDING;
            this.submit_button.setPosition({ x: this.xi() + RisqRightPanel.PADDING, y: yi });
            this.submit_button.setW(this.w() - 2 * RisqRightPanel.PADDING)
            this.need_to_set_position = false;
          }
        }
      );
    }
    if (this.config.is_open && !this.opening) {
      this.orders_list.draw(ctx, transform, dt);
      this.submit_button.draw(ctx, transform, dt);
    }
    this.open_button.draw(ctx, transform, dt);
  }

  private drawSeparator(ctx: CanvasRenderingContext2D, yi: number) {
    ctx.strokeStyle = 'rgba(60, 60, 60, 0.6)';
    ctx.lineWidth = 2;
    drawLine(ctx, { x: this.xi() + 0.15 * this.w(), y: yi }, { x: this.xf() - 0.15 * this.w(), y: yi });
  }

  private drawPopulation(ctx: CanvasRenderingContext2D, yi: number, pop: number, limit: number) {
    ctx.beginPath();
    ctx.drawImage(this.risq.getIcon('icons/person64'), this.xi() + 0.1 * this.w(), yi, 30, 30);
    drawText(ctx, `${pop}/${limit}`, {
      p: { x: this.xi() + 0.1 * this.w() + 36, y: yi + 15 },
      w: 0.9 * this.w() - 36,
      fill_style: 'black',
      baseline: 'middle',
    });
  }

  private drawResource(ctx: CanvasRenderingContext2D, yi: number, r: RisqResourceType, a: number) {
    ctx.beginPath();
    ctx.drawImage(this.risq.getIcon(resourceTypeImage(r)), this.xi() + 0.1 * this.w(), yi, 30, 30);
    // TODO: add number of workers on each resource
    drawText(ctx, a.toString(), {
      p: { x: this.xi() + 0.1 * this.w() + 36, y: yi + 15 },
      w: 0.9 * this.w() - 36,
      fill_style: 'black',
      baseline: 'middle',
    });
  }

  private drawScore(ctx: CanvasRenderingContext2D, yi: number, score: GameRisqScoreEntry) {
    ctx.beginPath();
    drawText(ctx, `${score.nickname}: ${score.score}`, {
      p: { x: this.xi() + 0.9 * this.w(), y: yi + 15 },
      w: 0.8 * this.w(),
      fill_style: score.color.getString(),
      baseline: 'middle',
      align: 'right',
    });
  }

  scroll(dy: number, mode: number): boolean {
    return this.orders_list.scroll(dy, mode);
  }

  mousemove(m: Point2D, transform: BoardTransformData): boolean {
    if (this.open_button.mousemove(m, transform)) {
      return true;
    }
    this.orders_list.mousemove(m, transform);
    this.submit_button.mousemove(m, transform);
    m = {
      x: m.x * transform.scale - transform.view.x,
      y: m.y * transform.scale - transform.view.y,
    };
    if (m.x > this.xi() && m.y > this.yi() && m.x < this.xf() && m.y < this.yf()) {
      this.hovering = true;
    } else {
      this.hovering = false;
    }
    return this.isHovering();
  }

  mousedown(e: MouseEvent): boolean {
    if (this.open_button.mousedown(e)) {
      return true;
    }
    this.orders_list.mousedown(e);
    this.submit_button.mousedown(e);
    return this.isHovering();
  }

  mouseup(e: MouseEvent) {
    this.open_button.mouseup(e);
    this.orders_list.mouseup(e);
    this.submit_button.mouseup(e);
  }

  xi(): number {
    return this.open_button.xf();
  }
  yi(): number {
    return 0;
  }
  xf(): number {
    return this.risq.canvasSize().width;
  }
  yf(): number {
    return this.risq.canvasSize().height;
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
  paddedW(): number {
    return this.w() - 2 * RisqRightPanel.PADDING;
  }
  h(): number {
    return this.yf() - this.yi();
  }
}
