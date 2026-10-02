import { DwgSquareButton } from '../../../../util/canvas_components/button/square_button';

export class RisqOrderCancelButton extends DwgSquareButton {
  private on_cancel: () => void;

  constructor(s: number, on_cancel: () => void, is_restore = false) {
    super({
      button_config: { allow_nonleft_clicks: false },
      p: { x: 0, y: 0 },
      s,
      draw_config: {
        fill_style: 'transparent',
        stroke_style: 'transparent',
        stroke_width: 0,
        hover_fill_style: is_restore ? 'rgba(46, 125, 58, 0.35)' : 'rgba(122, 46, 27, 0.35)',
        click_fill_style: is_restore ? 'rgba(46, 125, 58, 0.55)' : 'rgba(122, 46, 27, 0.55)',
      },
      image_path: is_restore ? 'icons/refresh_gray32' : 'icons/close_gray32',
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
