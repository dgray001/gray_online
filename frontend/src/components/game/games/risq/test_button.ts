import {ClickSource} from "../../util/canvas_components/button/button";
import {DwgRectButton} from "../../util/canvas_components/button/rect_button"

export class TestButton extends DwgRectButton {
  constructor() {
    super({
      button_config: {},
      p: {x: 300, y: 100},
      w: 150,
      h: 90,
      draw_config: {
        fill_style: 'red',
        stroke_style: 'blue',
        stroke_width: 2,
        hover_fill_style: 'purple',
        click_fill_style: 'blue',
        hover_stroke_style: 'transparent',
        hover_stroke_width: 4,
        click_stroke_style: 'green',
        fixed_position: true,
      },
    });
  }

  protected hovered(): void {
    console.log('hovered');
  }
  protected unhovered(): void {
    console.log('unhovered');
  }
  protected clicked(source: ClickSource): void {
    console.log('clicked', ClickSource[source]);
  }
  protected released(source: ClickSource): void {
    console.log('released', ClickSource[source]);
  }
}