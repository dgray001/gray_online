import type { BoardTransformData } from '../game/util/canvas_board/canvas_board';
import type { CanvasComponent } from '../game/util/canvas_components/canvas_component';
import { configDraw } from '../game/util/canvas_components/canvas_component';
import { drawRect, drawText } from '../game/util/canvas_util';
import type { Point2D } from '../game/util/objects2d';
import { EditorButton } from './editor_button';
import { HEADER_H, INK, PANEL_BACKGROUND } from './editor_style';
import type { CanvasSize } from './editor_right_panel';

const PANEL_W = 260;
const PADDING = 8;
const CLOSE_S = 22;
const TITLE_H = 28;
const LINE_H = 18;

export class EditorLeftPanel implements CanvasComponent {
  private shown = false;
  private hovering = false;
  private title = '';
  private lines: string[] = [];
  private close_button: EditorButton;

  constructor(
    private canvas_size: () => CanvasSize,
    on_close: () => void
  ) {
    this.close_button = new EditorButton(
      'x',
      { x: PANEL_W - CLOSE_S - PADDING, y: HEADER_H + PADDING },
      CLOSE_S,
      on_close,
      CLOSE_S
    );
  }

  show(title: string, lines: string[]): void {
    this.shown = true;
    this.title = title;
    this.lines = lines;
  }

  hide(): void {
    this.shown = false;
    this.hovering = false;
    this.close_button.setHovering(false);
  }

  isShown(): boolean {
    return this.shown;
  }

  isHovering(): boolean {
    return this.hovering;
  }

  setHovering(hovering: boolean): void {
    this.hovering = this.hovering && hovering;
    this.close_button.setHovering(hovering);
  }

  isClicking(): boolean {
    return this.close_button.isClicking();
  }

  setClicking(clicking: boolean): void {
    this.close_button.setClicking(clicking);
  }

  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void {
    if (!this.shown) {
      return;
    }
    const height = this.canvas_size().h - HEADER_H;
    configDraw(
      ctx,
      transform,
      { fill_style: PANEL_BACKGROUND, stroke_width: 0, fixed_position: true },
      false,
      false,
      () => {
        drawRect(ctx, { x: 0, y: HEADER_H }, PANEL_W, height);
        drawText(ctx, this.title, {
          p: { x: PADDING, y: HEADER_H + PADDING },
          w: PANEL_W - 3 * PADDING - CLOSE_S,
          fill_style: INK,
          font: 'bold 16px "Fira Sans", sans-serif',
        });
        ctx.save();
        ctx.beginPath();
        ctx.rect(0, HEADER_H + TITLE_H, PANEL_W, height - TITLE_H);
        ctx.clip();
        this.lines.forEach((line, i) => {
          drawText(ctx, line, {
            p: { x: PADDING, y: HEADER_H + TITLE_H + PADDING + i * LINE_H },
            w: PANEL_W - 2 * PADDING,
            fill_style: INK,
            font: '13px "Fira Sans", sans-serif',
          });
        });
        ctx.restore();
      }
    );
    this.close_button.draw(ctx, transform, dt);
  }

  mousemove(canvas: Point2D, screen: Point2D, transform: BoardTransformData): boolean {
    if (!this.shown) {
      return false;
    }
    this.close_button.mousemove(canvas, screen, transform);
    this.hovering = screen.x <= PANEL_W && screen.y >= HEADER_H;
    return this.hovering;
  }

  mousedown(e: MouseEvent): boolean {
    if (!this.shown) {
      return false;
    }
    const closing = this.close_button.mousedown(e);
    return closing || this.hovering;
  }

  mouseup(e: MouseEvent): void {
    if (this.shown) {
      this.close_button.mouseup(e);
    }
  }

  xi(): number {
    return 0;
  }
  yi(): number {
    return HEADER_H;
  }
  xf(): number {
    return PANEL_W;
  }
  yf(): number {
    return this.canvas_size().h;
  }
  w(): number {
    return PANEL_W;
  }
  h(): number {
    return this.canvas_size().h - HEADER_H;
  }
}
