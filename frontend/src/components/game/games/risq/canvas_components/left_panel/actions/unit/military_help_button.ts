import { DialogSize } from '../../../../../../../dialog_box/dialog_box';
import type { BoardTransformData } from '../../../../../../util/canvas_board/canvas_board';
import { configDraw } from '../../../../../../util/canvas_components/canvas_component';
import { drawCircle } from '../../../../../../util/canvas_util';
import type { Point2D } from '../../../../../../util/objects2d';
import type { DwgRisq } from '../../../../risq';
import { RisqActionButton } from '../action_button';
const MILITARY_HELP_TEXT = `Military unit behavior

Aggressive: Automatically attacks anything in weapon range plus anything in its own or adjacent spaces, and keeps walking wherever it keeps finding targets, even across the whole map.

Defensive: Automatically attacks anything it can hit from any zone of its current space. It never leaves that space.

Stand Ground: Automatically attacks only what it can hit from its current zone. It never leaves that zone. Explicit movement orders still work.

Passive: Never automatically seeks enemies. Attack Back can still trigger retaliation within attack range.

Interrupt Current: Allows automatic attacks to replace existing orders. Automatic searches keep explicitly assigned unit or building attack targets, but Attack Back can replace them when this toggle is enabled.

Attack Back: Responds to reachable attackers from the latest recorded attack tick. Without Interrupt Current, Passive retaliates only while idle; other stances can interrupt non-attack orders but keep existing attacks. Passive and Stand Ground retaliate only within attack range; Defensive also retaliates against attackers in the current space, and Aggressive against attackers within one space.

Target priority: Click an icon to add its category, drag ranked icons to reorder, and click a ranked icon to remove it. Ranked categories are tried first; unranked categories remain fallback targets. Nearer targets win within a category, then the lowest id. Targets that need a long detour (over 10 zone moves) are ignored. Priority applies to automatic attacks, retaliation, and attack-space or attack-zone orders.

Multiple units: A solid highlight means all selected units share that setting. Stripes mark stances present in a mixed selection or a boolean with both on and off values. Click a stance to apply it to everyone. Left-click a striped boolean to enable it for everyone; right-click to disable it. Different priority lists show a striped Mixed container. Click a category to give everyone that shared priority, then add, remove, or reorder categories normally.`;

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
