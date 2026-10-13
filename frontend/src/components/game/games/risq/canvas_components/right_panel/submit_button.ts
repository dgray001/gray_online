import type { BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import type { RectButtonConfig } from '../../../../util/canvas_components/button/rect_button';
import { DwgRectButton } from '../../../../util/canvas_components/button/rect_button';
import type { DwgRisq } from '../../risq';
import { RisqButtonTooltip } from '../risq_tooltip';
import { getSettings } from '../../../../../../scripts/settings_store';
import { RisqHotkeyAction, hotkeyDisplayString } from '../../application/input/hotkeys';

export abstract class RisqSubmitButton extends DwgRectButton {
  private tooltip = new RisqButtonTooltip(() => ({
    title: 'Submit Orders',
    hotkey: hotkeyDisplayString(getSettings().risq_hotkeys.actions[RisqHotkeyAction.SUBMIT_ORDERS]),
  }));

  constructor(
    private risq: DwgRisq,
    config: RectButtonConfig
  ) {
    super(config);
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
