import type { BoardTransformData } from '../../game/util/canvas_board/canvas_board';
import type { CanvasComponent } from '../../game/util/canvas_components/canvas_component';
import { configDraw } from '../../game/util/canvas_components/canvas_component';
import { drawText } from '../../game/util/canvas_util';
import type { Point2D } from '../../game/util/objects2d';
import { EditorButton } from '../editor_button';
import { FONT, INK, TEXT_CONFIG } from '../editor_style';
import type { EditorConfigPanel, EditorToolContext, PanelBounds } from '../editor_tool';
import { HeaderInput } from '../map_editor_header';

export type BackgroundCorner = 'top_left' | 'top_right';

export interface BackgroundActions {
  set: (name: string) => void;
  clear: () => void;
  nudge: (corner: BackgroundCorner, axis: 0 | 1, delta: number) => void;
}

const PADDING = 6;
const GAP = 4;
const ROW_H = 24;
const LABEL_W = 130;
const STEP_W = 24;
const CORNER_ROWS: { corner: BackgroundCorner; axis: 0 | 1; label: string }[] = [
  { corner: 'top_left', axis: 0, label: 'Top left x' },
  { corner: 'top_left', axis: 1, label: 'Top left y' },
  { corner: 'top_right', axis: 0, label: 'Top right x' },
  { corner: 'top_right', axis: 1, label: 'Top right y' },
];

export class BackgroundPanel implements EditorConfigPanel {
  private bounds: PanelBounds = { x: 0, y: 0, w: 0, h: 0 };
  private name_input = new HeaderInput({ x: 0, y: 0 }, 0, 'image_name', {
    max_length: 40,
    allowed_chars: /[a-z0-9_]/,
  });
  private set_button: EditorButton;
  private clear_button: EditorButton;
  private steppers: EditorButton[];
  private components: CanvasComponent[];

  constructor(
    private context: EditorToolContext,
    actions: BackgroundActions
  ) {
    this.set_button = new EditorButton('Set', { x: 0, y: 0 }, 0, () => actions.set(this.name_input.value().trim()));
    this.clear_button = new EditorButton('Clear', { x: 0, y: 0 }, 0, () => actions.clear());
    this.steppers = CORNER_ROWS.flatMap(({ corner, axis }) => [
      new EditorButton('-', { x: 0, y: 0 }, STEP_W, () => actions.nudge(corner, axis, -1)),
      new EditorButton('+', { x: 0, y: 0 }, STEP_W, () => actions.nudge(corner, axis, 1)),
    ]);
    this.components = [this.name_input, this.set_button, this.clear_button, ...this.steppers];
  }

  setBounds(bounds: PanelBounds): void {
    this.bounds = bounds;
    const x = bounds.x + PADDING;
    const half = (bounds.w - 2 * PADDING - GAP) / 2;
    this.name_input.setPosition({ x, y: this.rowY(1) });
    this.name_input.setSize(bounds.w - 2 * PADDING, ROW_H);
    this.set_button.setPosition({ x, y: this.rowY(2) });
    this.set_button.setW(half);
    this.clear_button.setPosition({ x: x + half + GAP, y: this.rowY(2) });
    this.clear_button.setW(half);
    this.steppers.forEach((button, i) => {
      const x_offset = LABEL_W + (i % 2) * (STEP_W + GAP);
      button.setPosition({ x: x + x_offset, y: this.rowY(3 + Math.floor(i / 2)) });
    });
  }

  private rowY(row: number): number {
    return this.bounds.y + PADDING + row * (ROW_H + GAP);
  }

  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void {
    if (this.bounds.w <= 0 || this.bounds.h <= 0) {
      return;
    }
    const background = this.context.doc()?.background_image;
    configDraw(
      ctx,
      transform,
      { fill_style: 'transparent', stroke_width: 0, fixed_position: true },
      false,
      false,
      () => {
        const x = this.bounds.x + PADDING;
        const text = { ...TEXT_CONFIG, w: this.bounds.w - 2 * PADDING, baseline: 'middle' as const };
        drawText(ctx, background ? `Image: ${background.name}` : 'No background image', {
          ...text,
          p: { x, y: this.rowY(0) + ROW_H / 2 },
        });
        CORNER_ROWS.forEach(({ corner, axis, label }, i) => {
          const value = background ? background[corner][axis] : '-';
          drawText(ctx, `${label}: ${value}`, { ...text, w: LABEL_W, p: { x, y: this.rowY(3 + i) + ROW_H / 2 } });
        });
        drawText(ctx, 'Corners are space coordinates; height follows the image aspect.', {
          ...text,
          font: FONT,
          fill_style: INK,
          p: { x, y: this.rowY(7) + ROW_H / 2 },
        });
      }
    );
    this.components.forEach((c) => c.draw(ctx, transform, dt));
  }

  mousemove(canvas: Point2D, screen: Point2D, transform: BoardTransformData): boolean {
    const over_child = this.components.map((c) => c.mousemove(canvas, screen, transform)).some(Boolean);
    return over_child || this.contains(screen);
  }

  mousedown(e: MouseEvent): boolean {
    const consumed = this.components.map((c) => c.mousedown(e)).some(Boolean);
    return consumed || this.isHovering();
  }

  mouseup(e: MouseEvent): void {
    this.components.forEach((c) => c.mouseup(e));
  }

  scroll(): boolean {
    return this.isHovering();
  }

  isTyping(): boolean {
    return this.name_input.isFocused();
  }

  isHovering(): boolean {
    return this.components.some((c) => c.isHovering()) || this.hovered;
  }

  private hovered = false;

  private contains(p: Point2D): boolean {
    this.hovered = p.x >= this.xi() && p.x <= this.xf() && p.y >= this.yi() && p.y <= this.yf();
    return this.hovered;
  }

  setHovering(hovering: boolean): void {
    this.hovered = this.hovered && hovering;
    this.components.forEach((c) => c.setHovering(hovering));
  }

  isClicking(): boolean {
    return this.components.some((c) => c.isClicking());
  }

  setClicking(clicking: boolean): void {
    this.components.forEach((c) => c.setClicking(clicking));
  }

  xi(): number {
    return this.bounds.x;
  }
  yi(): number {
    return this.bounds.y;
  }
  xf(): number {
    return this.bounds.x + this.bounds.w;
  }
  yf(): number {
    return this.bounds.y + this.bounds.h;
  }
  w(): number {
    return this.bounds.w;
  }
  h(): number {
    return this.bounds.h;
  }
}
