import { DialogSize } from '../../../../../../../dialog_box/dialog_box';
import type { BoardTransformData } from '../../../../../../util/canvas_board/canvas_board';
import { configDraw } from '../../../../../../util/canvas_components/canvas_component';
import { drawCircle } from '../../../../../../util/canvas_util';
import type { Point2D } from '../../../../../../util/objects2d';
import type { DwgRisq } from '../../../../risq';
import { RisqActionButton } from '../action_button';
import { MILITARY_HELP_TEXT } from './military_help_text';

export class RisqMilitaryHelpButton extends RisqActionButton {
  constructor(
    private risq: DwgRisq,
    col: number
  ) {
    super({ row: 1, col, image_path: 'icons/question64', description: 'Military behavior help' }, 0);
  }
  override mouseOver(_canvas: Point2D, screen: Point2D): boolean {
    return this.w() > 0 && Math.hypot(screen.x - this.xc(), screen.y - this.yc()) <= 0.375 * this.w();
  }
  protected override _draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, _dt: number): void {
    configDraw(
      ctx,
      transform,
      {
        fixed_position: true,
        fill_style: 'rgba(235, 210, 165, 0.2)',
        stroke_style: 'rgb(60, 40, 20)',
        stroke_width: 1,
        hover_fill_style: 'rgba(235, 210, 165, 0.5)',
        click_fill_style: 'rgba(235, 210, 165, 0.7)',
      },
      this.isHovering(),
      this.isClicking(),
      (): void => {
        drawCircle(ctx, { x: this.xc(), y: this.yc() }, 0.375 * this.w());
        const size = 0.5 * this.w();
        ctx.drawImage(this.risq.getIcon('icons/question64'), this.xc() - size / 2, this.yc() - size / 2, size, size);
      }
    );
  }
  override execute(): void {
    const dialog = document.createElement('dwg-message-dialog');
    dialog.classList.add('risq-military-help');
    dialog.setData({ message: MILITARY_HELP_TEXT, button_text: 'Close', size: DialogSize.LARGE });
    this.risq.appendChild(dialog);
  }
}
