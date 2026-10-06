import type { BoardTransformData } from '../game/util/canvas_board/canvas_board';
import { DwgSquareButton } from '../game/util/canvas_components/button/square_button';
import type { CanvasComponent } from '../game/util/canvas_components/canvas_component';
import { configDraw } from '../game/util/canvas_components/canvas_component';
import { drawLine, drawRect } from '../game/util/canvas_util';
import type { Point2D } from '../game/util/objects2d';
import { EditorButton } from './editor_button';
import type { EditorConfigPanel, EditorTool } from './editor_tool';
import { HEADER_H, INK, PANEL_BACKGROUND } from './editor_style';

const PANEL_W = 240;
const PADDING = 6;
const GAP = 6;
const COLUMNS = 3;
const BUTTON_H = 28;
const TOGGLE_S = 36;

export interface CanvasSize {
  w: number;
  h: number;
}

class PanelToggleButton extends DwgSquareButton {
  constructor(private on_release: () => void) {
    super({
      button_config: { allow_nonleft_clicks: false },
      p: { x: 0, y: 0 },
      s: TOGGLE_S,
      draw_config: {
        fill_style: 'transparent',
        stroke_style: 'transparent',
        stroke_width: 0,
        hover_fill_style: 'rgb(180, 180, 180, 0.5)',
        click_fill_style: 'rgb(210, 210, 210, 0.7)',
        fixed_position: true,
      },
      image_path: 'icons/triangle_gray36',
      rotation: -0.5 * Math.PI,
    });
  }

  protected hovered(): void {}
  protected unhovered(): void {}
  protected clicked(): void {}

  protected released(): void {
    if (this.isHovering()) {
      this.on_release();
      this.setHovering(false);
    }
  }
}

export class EditorRightPanel implements CanvasComponent {
  private open = true;
  private hovering = false;
  private active = 0;
  private buttons: EditorButton[];
  private toggle_button: PanelToggleButton;

  constructor(
    private tools: EditorTool[],
    private canvas_size: () => CanvasSize,
    on_tool: (index: number) => void
  ) {
    const width = (PANEL_W - 2 * PADDING - (COLUMNS - 1) * GAP) / COLUMNS;
    this.buttons = tools.map(
      (tool, i) =>
        new EditorButton(
          `${tool.label} (${tool.hotkey.toUpperCase()})`,
          { x: 0, y: 0 },
          width,
          () => on_tool(i),
          BUTTON_H
        )
    );
    this.toggle_button = new PanelToggleButton(() => (this.open = !this.open));
  }

  setActive(index: number): void {
    this.active = index;
  }

  private config(): EditorConfigPanel | undefined {
    return this.tools[this.active].config();
  }

  private panelX(): number {
    return this.canvas_size().w - (this.open ? PANEL_W : 0);
  }

  private toolsBottom(): number {
    return HEADER_H + PADDING + Math.ceil(this.tools.length / COLUMNS) * (BUTTON_H + GAP);
  }

  private layout(): void {
    const x = this.panelX();
    const button_w = this.buttons[0]?.w() ?? 0;
    this.buttons.forEach((button, i) => {
      const column = i % COLUMNS;
      const row = Math.floor(i / COLUMNS);
      button.setPosition({
        x: x + PADDING + column * (button_w + GAP),
        y: HEADER_H + PADDING + row * (BUTTON_H + GAP),
      });
    });
    const config_top = this.toolsBottom() + PADDING;
    this.config()?.setBounds({
      x: x + PADDING,
      y: config_top,
      w: PANEL_W - 2 * PADDING,
      h: this.canvas_size().h - config_top - PADDING,
    });
    this.toggle_button.setPosition({ x: x - TOGGLE_S, y: HEADER_H + PADDING });
    this.toggle_button.setRotation({ direction: true, angle: (this.open ? 0.5 : -0.5) * Math.PI }, undefined, true);
  }

  private activeComponents(): CanvasComponent[] {
    const config = this.config();
    return this.open ? [...this.buttons, ...(config ? [config] : [])] : [];
  }

  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void {
    this.layout();
    if (this.open) {
      this.drawBackground(ctx, transform);
      this.buttons.forEach((button) => button.draw(ctx, transform, dt));
      this.drawActiveOutline(ctx, transform);
      this.config()?.draw(ctx, transform, dt);
    }
    this.toggle_button.draw(ctx, transform, dt);
  }

  private drawBackground(ctx: CanvasRenderingContext2D, transform: BoardTransformData): void {
    const x = this.panelX();
    const draw_config = { fill_style: PANEL_BACKGROUND, stroke_width: 0, fixed_position: true };
    configDraw(ctx, transform, draw_config, false, false, () => {
      drawRect(ctx, { x, y: HEADER_H }, PANEL_W, this.canvas_size().h - HEADER_H);
      ctx.strokeStyle = 'rgba(60, 60, 60, 0.6)';
      ctx.lineWidth = 2;
      const y = this.toolsBottom();
      drawLine(ctx, { x: x + PADDING, y }, { x: x + PANEL_W - PADDING, y });
    });
  }

  private drawActiveOutline(ctx: CanvasRenderingContext2D, transform: BoardTransformData): void {
    const button = this.buttons[this.active];
    if (!button) {
      return;
    }
    configDraw(
      ctx,
      transform,
      { fill_style: 'transparent', stroke_style: INK, stroke_width: 2.5, fixed_position: true },
      false,
      false,
      () => {
        drawRect(ctx, { x: button.xi(), y: button.yi() }, button.w(), button.h(), 3);
      }
    );
  }

  isHovering(): boolean {
    return this.hovering || this.toggle_button.isHovering();
  }

  setHovering(hovering: boolean): void {
    this.hovering = this.hovering && hovering;
    this.activeComponents().forEach((c) => c.setHovering(hovering));
  }

  isClicking(): boolean {
    return this.activeComponents().some((c) => c.isClicking());
  }

  setClicking(clicking: boolean): void {
    this.activeComponents().forEach((c) => c.setClicking(clicking));
  }

  mousemove(canvas: Point2D, screen: Point2D, transform: BoardTransformData): boolean {
    this.layout();
    const over_toggle = this.toggle_button.mousemove(canvas, screen, transform);
    const over_children = this.activeComponents().map((c) => c.mousemove(canvas, screen, transform));
    this.hovering = this.open && screen.x >= this.panelX() && screen.y >= HEADER_H;
    return over_toggle || this.hovering || over_children.some(Boolean);
  }

  scroll(dy: number, mode: number, dx?: number): boolean {
    const consumed = this.activeComponents().some((c) => c.scroll?.(dy, mode, dx));
    return consumed || this.hovering;
  }

  mousedown(e: MouseEvent): boolean {
    const toggled = this.toggle_button.mousedown(e);
    const consumed = this.activeComponents().map((c) => c.mousedown(e));
    return toggled || this.hovering || consumed.some(Boolean);
  }

  mouseup(e: MouseEvent): void {
    this.toggle_button.mouseup(e);
    this.activeComponents().forEach((c) => c.mouseup(e));
  }

  xi(): number {
    return this.panelX();
  }
  yi(): number {
    return HEADER_H;
  }
  xf(): number {
    return this.canvas_size().w;
  }
  yf(): number {
    return this.canvas_size().h;
  }
  w(): number {
    return this.open ? PANEL_W : 0;
  }
  h(): number {
    return this.canvas_size().h - HEADER_H;
  }
}
