import type { BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import { DwgRectButton } from '../../../../util/canvas_components/button/rect_button';
import { configDraw } from '../../../../util/canvas_components/canvas_component';

export type ReplayGlyph = 'play' | 'pause' | 'close';

export const REPLAY_CONTROL_SIZE = 28;

export class RisqReplayControlButton extends DwgRectButton {
  constructor(
    private glyph: () => ReplayGlyph,
    private on_release: () => void
  ) {
    super({
      button_config: {},
      p: { x: 0, y: 0 },
      w: REPLAY_CONTROL_SIZE,
      h: REPLAY_CONTROL_SIZE,
      draw_config: {
        fill_style: 'transparent',
        stroke_style: 'rgb(60, 40, 20)',
        stroke_width: 1,
        hover_fill_style: 'rgba(210, 180, 130, 0.5)',
        click_fill_style: 'rgba(235, 210, 165, 0.7)',
        fixed_position: true,
      },
      r: 4,
    });
  }

  protected hovered(): void {}
  protected unhovered(): void {}
  protected clicked(): void {}

  protected released(): void {
    if (this.isHovering()) {
      this.on_release();
    }
  }

  protected override _draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void {
    super._draw(ctx, transform, dt);
    const draw_config = { fill_style: 'black', stroke_style: 'black', stroke_width: 3, fixed_position: true };
    configDraw(ctx, transform, draw_config, false, false, () => this.drawGlyph(ctx));
  }

  private drawGlyph(ctx: CanvasRenderingContext2D): void {
    const r = 0.25 * this.w();
    const c = { x: this.xi() + 0.5 * this.w(), y: this.yi() + 0.5 * this.h() };
    switch (this.glyph()) {
      case 'play':
        ctx.moveTo(c.x - 0.8 * r, c.y - r);
        ctx.lineTo(c.x + r, c.y);
        ctx.lineTo(c.x - 0.8 * r, c.y + r);
        ctx.closePath();
        ctx.fill();
        break;
      case 'pause':
        ctx.fillRect(c.x - r, c.y - r, 0.7 * r, 2 * r);
        ctx.fillRect(c.x + 0.3 * r, c.y - r, 0.7 * r, 2 * r);
        break;
      case 'close':
        ctx.moveTo(c.x - r, c.y - r);
        ctx.lineTo(c.x + r, c.y + r);
        ctx.moveTo(c.x + r, c.y - r);
        ctx.lineTo(c.x - r, c.y + r);
        ctx.stroke();
        break;
    }
  }
}
