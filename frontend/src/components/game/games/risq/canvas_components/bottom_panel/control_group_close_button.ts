import { DwgRectButton } from '../../../../util/canvas_components/button/rect_button';
import { ClickSource } from '../../../../util/canvas_components/button/button_config';
import type { Point2D } from '../../../../util/objects2d';

export class RisqControlGroupCloseButton extends DwgRectButton {
  constructor(
    p: Point2D,
    size: number,
    private on_remove: () => void
  ) {
    super({
      button_config: {},
      p,
      w: size,
      h: size,
      draw_config: {
        fill_style: 'transparent',
        stroke_width: 0,
        fixed_position: true,
        hover_fill_style: 'rgba(220, 220, 220, 0.2)',
        click_fill_style: 'rgba(250, 250, 250, 0.4)',
      },
      text: {
        text: '×',
        config: {
          fill_style: 'black',
          align: 'center',
          baseline: 'middle',
          font: 'bold 14px serif',
        },
      },
    });
  }
  protected hovered(): void {}
  protected unhovered(): void {}
  protected clicked(): void {}
  protected released(source: ClickSource): void {
    if (source === ClickSource.LEFT_MOUSE && this.isHovering()) {
      this.on_remove();
    }
  }
}
