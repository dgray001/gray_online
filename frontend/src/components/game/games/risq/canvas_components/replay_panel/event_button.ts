import { DwgRectButton } from '../../../../util/canvas_components/button/rect_button';

export class RisqEventButton extends DwgRectButton {
  constructor(
    image: CanvasImageSource | undefined,
    text: string,
    private on_release: () => void,
    width: number = 32
  ) {
    super({
      button_config: {},
      p: { x: 0, y: 0 },
      w: width,
      h: 32,
      image_on_top: true,
      r: 4,
      draw_config: {
        fill_style: 'rgb(222, 184, 135)',
        stroke_style: 'rgb(70, 30, 5)',
        stroke_width: 2,
        hover_fill_style: 'rgb(245, 215, 170)',
        fixed_position: true,
      },
      text: {
        text,
        padding: 4,
        config: { align: 'center', baseline: 'middle', font: '18px sans-serif', fill_style: 'black' },
      },
    });
    if (image) {
      this.setImage(image);
    }
  }

  protected hovered(): void {}
  override setImage(image: CanvasImageSource): void {
    super.setImage(image);
  }
  protected unhovered(): void {}
  protected clicked(): void {}
  protected released(): void {
    if (this.isHovering()) {
      this.on_release();
    }
  }
}
