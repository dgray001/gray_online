import type { BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import type { CanvasComponent, Rotation } from '../../../../util/canvas_components/canvas_component';
import { configDraw } from '../../../../util/canvas_components/canvas_component';
import { drawLine, drawRect, drawText } from '../../../../util/canvas_util';
import type { Point2D } from '../../../../util/objects2d';
import type { DwgRisq } from '../../risq';
import type { GameRisqScoreEntry, RisqResourceType } from '../../risq_data';
import { resourceTypeImage } from '../../risq_resources';
import { RisqRightPanelButton } from './right_panel_button';

/** Config for the right panel */
export declare interface RightPanelConfig {
  w: number;
  is_open: boolean;
  background: string;
}

export class RisqRightPanel implements CanvasComponent {
  private open_button: RisqRightPanelButton;

  private risq: DwgRisq;
  private config: RightPanelConfig;
  private opening = false;
  private hovering = false;

  constructor(risq: DwgRisq, config: RightPanelConfig) {
    this.risq = risq;
    this.config = config;
    this.open_button = new RisqRightPanelButton(risq);
    this.toggle(config.is_open, true);
  }

  isHovering(): boolean {
    return this.hovering;
  }

  isClicking(): boolean {
    return false;
  }

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
      },
      initial
    );
  }

  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number) {
    if (this.opening || this.config.is_open) {
      configDraw(
        ctx,
        transform,
        {
          fill_style: this.config.background,
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
          let yi = this.yi() + 3;
          drawText(ctx, `Turn ${this.risq.getGame()?.turn_number ?? '??'}`, {
            p: { x: this.xc(), y: yi },
            w: this.w(),
            fill_style: 'black',
            align: 'center',
            font: 'bold 36px serif',
          });
          yi += 40;
          const player = this.risq.getPlayer();
          this.drawSeparator(ctx, yi);
          yi += 5;
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
            yi += 5;
            this.drawSeparator(ctx, yi);
            yi += 5;
          }
          for (const score of this.risq.getGame()?.scores ?? []) {
            this.drawScore(ctx, yi, score);
            yi += 30;
          }
          yi += 5;
          this.drawSeparator(ctx, yi);
          yi += 5;
          // TODO: logic in case yi has gone off the rectangle
        }
      );
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

  mousemove(m: Point2D, transform: BoardTransformData): boolean {
    if (this.open_button.mousemove(m, transform)) {
      return true;
    }
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
    return this.isHovering();
  }

  mouseup(e: MouseEvent) {
    this.open_button.mouseup(e);
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
  h(): number {
    return this.yf() - this.yi();
  }
}
