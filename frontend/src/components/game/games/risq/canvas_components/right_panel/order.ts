import type { BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import { configDraw, type CanvasComponent } from '../../../../util/canvas_components/canvas_component';
import { drawRect, drawText } from '../../../../util/canvas_util';
import type { Point2D } from '../../../../util/objects2d';
import type { DwgRisq } from '../../risq';
import type { RisqFrontendOrder } from '../../risq_data';
import { isBuildingOrder, isUnitOrder, RisqOrderType } from '../../risq_data';

/** Data describing a risq order config */
export declare interface RisqOrderComponentConfig {
  w: number;
  order: RisqFrontendOrder;
  game: DwgRisq;
}

export class RisqOrderComponent implements CanvasComponent {
  private config: RisqOrderComponentConfig;
  private hovering = false;
  private clicking = false;

  constructor(config: RisqOrderComponentConfig) {
    this.config = config;
  }

  getOrder(): RisqFrontendOrder {
    return this.config.order;
  }

  isHovering() {
    return this.hovering;
  }

  setHovering(hovering: boolean) {
    this.hovering = hovering;
  }

  isClicking() {
    return this.clicking;
  }

  setClicking(clicking: boolean) {
    this.clicking = clicking;
  }

  setW(w: number) {
    this.config.w = w;
  }

  padding(): number {
    return 4;
  }

  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, _dt: number) {
    configDraw(
      ctx,
      transform,
      {
        fill_style: 'rgb(0, 200, 0)',
        stroke_width: 0,
        hover_fill_style: 'rgb(80, 255, 80)',
        click_fill_style: 'rgb(160, 255, 160)',
      },
      this.isHovering(),
      this.isClicking(),
      () => {
        drawRect(ctx, { x: this.xi(), y: this.yi() }, this.w(), this.h(), 8);
        const order = this.config.order;
        if (!order || this.h() - 2 * this.padding() <= 0) {
          return;
        }
        const row_h = (this.h() - 2 * this.padding()) / 3;
        let yc = this.yi() + this.padding() + 0.5 * row_h;
        const subject_text =
          order.subjects.length > 1
            ? `${this.getSubjectName()}, +${order.subjects.length - 1} more`
            : this.getSubjectName();
        drawText(ctx, subject_text, {
          p: { x: this.xi() + this.padding(), y: yc },
          w: this.w() - 2 * this.padding(),
          align: 'left',
          baseline: 'middle',
          font: `${0.8 * row_h}px serif`,
          fill_style: 'rgb(0, 0, 0)',
        });
        yc += row_h;
        drawText(ctx, RisqOrderType[order.order_type], {
          p: { x: this.xi() + this.padding(), y: yc },
          w: this.w() - 2 * this.padding(),
          align: 'left',
          baseline: 'middle',
          font: `${0.8 * row_h}px serif`,
          fill_style: 'rgb(0, 0, 0)',
        });
        yc += row_h;
        drawText(ctx, order.target_id.toString(), {
          p: { x: this.xi() + this.padding(), y: yc },
          w: this.w() - 2 * this.padding(),
          align: 'left',
          baseline: 'middle',
          font: `${0.8 * row_h}px serif`,
          fill_style: 'rgb(0, 0, 0)',
        });
      }
    );
  }

  private getSubjectName(): string {
    const game = this.config.game.getGame();
    if (!this.config.order?.subjects?.length || !game) {
      return 'ERROR: no subjects or game';
    }
    const subject = this.config.order.subjects[0];
    const player = game.players[this.config.order.player_id];
    if (!subject || !player) {
      return 'ERROR: no subject or player';
    }
    if (isUnitOrder(this.config.order.order_type)) {
      return player.units.get(subject)?.display_name ?? '';
    } else if (isBuildingOrder(this.config.order.order_type)) {
      return player.buildings.get(subject)?.display_name ?? '';
    }
    return 'ERROR: not a valid order type';
  }

  mousemove(m: Point2D, transform: BoardTransformData): boolean {
    m = {
      x: m.x * transform.scale - transform.view.x,
      y: m.y * transform.scale - transform.view.y,
    };
    if (m.x < this.xi() || m.y < this.yi() || m.x > this.xf() || m.y > this.yf()) {
      this.hovering = false;
    } else {
      this.hovering = true;
    }
    return this.isHovering();
  }

  mousedown(_e: MouseEvent): boolean {
    if (this.isHovering()) {
      this.clicking = true;
    }
    return this.isClicking();
  }

  mouseup(_e: MouseEvent): void {
    this.clicking = false;
  }

  xi(): number {
    return 0;
  }

  xf(): number {
    return this.xi() + this.w();
  }

  yi(): number {
    return 0;
  }

  yf(): number {
    return this.yi() + this.h();
  }

  w(): number {
    return this.config.w;
  }

  h(): number {
    return 120;
  }
}
