import { DwgSquareButton } from '../../../../util/canvas_components/button/square_button';

export class RisqOrderCancelButton extends DwgSquareButton {
  private on_cancel: () => void;

  constructor(s: number, on_cancel: () => void) {
    super({
      button_config: { allow_nonleft_clicks: false },
      p: { x: 0, y: 0 },
      s,
      draw_config: {
        fill_style: 'transparent',
        stroke_style: 'transparent',
        stroke_width: 0,
        hover_fill_style: 'rgba(122, 46, 27, 0.35)',
        click_fill_style: 'rgba(122, 46, 27, 0.55)',
      },
      image_path: 'icons/close_gray32',
    });
    this.on_cancel = on_cancel;
  }

  protected hovered(): void {}
  protected unhovered(): void {}
  protected clicked(): void {}
  protected released(): void {
    if (this.isHovering()) {
      this.on_cancel();
    }
  }
}
