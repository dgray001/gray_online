import { DwgRectButton } from '../game/util/canvas_components/button/rect_button';
import type { Point2D } from '../game/util/objects2d';
import { CONTROL_DRAW, CONTROL_H, TEXT_CONFIG } from './editor_style';

export class EditorButton extends DwgRectButton {
  constructor(
    text: string,
    p: Point2D,
    w: number,
    private on_release: () => void,
    h = CONTROL_H
  ) {
    super({
      button_config: {},
      p,
      w,
      h,
      draw_config: CONTROL_DRAW,
      text: { text, config: { ...TEXT_CONFIG, align: 'center', baseline: 'middle' } },
      r: 3,
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
}
