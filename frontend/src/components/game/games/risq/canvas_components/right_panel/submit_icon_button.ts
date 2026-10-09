import type { BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import { DwgRectButton } from '../../../../util/canvas_components/button/rect_button';
import type { DwgRisq } from '../../risq';
import { RisqButtonTooltip } from '../risq_tooltip';
import { getSettings } from '../../../../../../scripts/settings_store';
import { RisqHotkeyAction, hotkeyDisplayString } from '../../application/input/hotkeys';

export class RisqSubmitIconButton extends DwgRectButton {
  private risq: DwgRisq;
  private tooltip = new RisqButtonTooltip(() => ({
    title: 'Submit Orders',
    hotkey: hotkeyDisplayString(getSettings().risq_hotkeys.actions[RisqHotkeyAction.SUBMIT_ORDERS]),
  }));

  constructor(risq: DwgRisq, h: number) {
    super({
      button_config: {
        allow_nonleft_clicks: false,
      },
      p: { x: 0, y: 0 },
      w: h,
      h,
      draw_config: {
        fill_style: 'rgb(50, 220, 50)',
        stroke_style: 'rgb(20, 150, 20)',
        stroke_width: 0,
        hover_fill_style: 'rgb(80, 240, 80)',
        click_fill_style: 'rgb(110, 255, 110)',
        fixed_position: true,
      },
      r: 0.1 * h,
      image_path: 'icons/submit128',
      image_on_top: true,
    });
    this.risq = risq;
  }

  protected hovered(): void {}
  protected unhovered(): void {}
  protected clicked(): void {}
  protected released(): void {
    if (this.isHovering()) {
      this.risq.submission.confirmSubmit();
    }
  }

  drawTooltip(ctx: CanvasRenderingContext2D, transform: BoardTransformData, risq: DwgRisq, dt: number): void {
    this.tooltip.draw(ctx, transform, risq, dt, this.isHovering(), this.isPressed());
  }
}
