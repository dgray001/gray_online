import type { BoardTransformData } from '../../../../../util/canvas_board/canvas_board';
import { configDraw } from '../../../../../util/canvas_components/canvas_component';
import { DwgSquareButton } from '../../../../../util/canvas_components/button/square_button';
import { drawRect, drawText } from '../../../../../util/canvas_util';

export declare interface RisqActionButtonConfig {
  row: number;
  col: number;
  image_path: string;
  description: string;
}

export abstract class RisqActionButton extends DwgSquareButton {
  readonly row: number;
  readonly col: number;
  readonly description: string;

  constructor(config: RisqActionButtonConfig, s: number) {
    super({
      button_config: {},
      p: { x: 0, y: 0 },
      s,
      draw_config: {
        fill_style: 'transparent',
        stroke_style: 'rgb(60, 40, 20)',
        stroke_width: 1,
        hover_fill_style: 'rgba(210, 180, 130, 0.5)',
        click_fill_style: 'rgba(235, 210, 165, 0.7)',
        draw_clicked_when_unhovered: true,
        fixed_position: true,
      },
      image_path: config.image_path,
    });
    this.row = config.row;
    this.col = config.col;
    this.description = config.description;
  }

  protected hovered(): void {}
  protected unhovered(): void {}
  protected clicked(): void {}

  dataRefreshed(): void {}

  override draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void {
    super.draw(ctx, transform, dt);
  }

  drawTooltip(ctx: CanvasRenderingContext2D, transform: BoardTransformData) {
    if (!this.isHovering()) {
      return;
    }
    configDraw(
      ctx,
      transform,
      { fill_style: 'transparent', stroke_width: 0, fixed_position: true },
      false,
      false,
      () => {
        const h = 20;
        const font = '12px serif';
        ctx.font = font;
        const w = ctx.measureText(this.description).width + 8;
        const p = { x: this.xi(), y: this.yi() - 2 - h };
        ctx.fillStyle = 'rgba(0, 0, 0, 0.85)';
        ctx.strokeStyle = 'rgba(250, 250, 250, 0.9)';
        ctx.lineWidth = 1;
        drawRect(ctx, p, w, h, 3);
        drawText(ctx, this.description, {
          p: { x: p.x + 4, y: p.y + 0.5 * h },
          w: w - 8,
          fill_style: 'white',
          align: 'left',
          baseline: 'middle',
          font,
        });
      }
    );
  }
}
