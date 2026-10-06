import type { BoardTransformData } from '../../game/util/canvas_board/canvas_board';
import { configDraw } from '../../game/util/canvas_components/canvas_component';
import { drawText } from '../../game/util/canvas_util';
import { EditorButton } from '../editor_button';
import { TEXT_CONFIG } from '../editor_style';
import type { EditorConfigPanel, PanelBounds } from '../editor_tool';

export class ConnectionsPanel extends EditorButton implements EditorConfigPanel {
  constructor(
    private mode: () => string,
    private status: () => string,
    on_mode: () => void
  ) {
    super('', { x: 0, y: 0 }, 0, on_mode);
  }

  setBounds(bounds: PanelBounds): void {
    this.setPosition({ x: bounds.x + 6, y: bounds.y + 76 });
    this.setW(Math.max(0, bounds.w - 12));
  }

  override draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void {
    this.setText(this.mode());
    super.draw(ctx, transform, dt);
    configDraw(
      ctx,
      transform,
      { fill_style: 'transparent', stroke_width: 0, fixed_position: true },
      false,
      false,
      () => {
        [
          'Connections',
          'Click two spaces to connect.',
          'Repeat the pair to remove.',
          'Same space cancels.',
          this.status(),
        ].forEach((text, i) => {
          drawText(ctx, text, {
            ...TEXT_CONFIG,
            p: { x: this.xi(), y: this.yi() - 64 + i * 16 + (i === 4 ? 44 : 0) },
            w: this.w(),
          });
        });
      }
    );
  }
}
