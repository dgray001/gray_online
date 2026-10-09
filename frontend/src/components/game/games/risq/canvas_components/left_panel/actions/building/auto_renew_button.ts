import type { BoardTransformData } from '../../../../../../util/canvas_board/canvas_board';
import { configDraw } from '../../../../../../util/canvas_components/canvas_component';
import { drawText } from '../../../../../../util/canvas_util';
import type { DwgRisq } from '../../../../risq';
import { canAffordCost } from '../../../../model/types';
import type { RisqTooltipData } from '../../../risq_tooltip';
import { RisqActionButton } from '../action_button';

export class RisqAutoRenewButton extends RisqActionButton {
  private change = 1;

  constructor(
    private risq: DwgRisq,
    private internal_id: number
  ) {
    super({ row: 1, col: 0, image_path: 'icons/wheat128', description: 'Auto-renew' }, 0);
  }

  override dataRefreshed(): void {
    const player = this.risq.getPlayer();
    const cost = player?.buildings.get(this.internal_id)?.renew_cost;
    this.dimmed = !!player && !!cost && !canAffordCost(player, cost);
    if (this.risq.session.canGiveOrders() && !this.risq.commands.auto_renew_pending) {
      this.enable();
    } else {
      this.disable();
    }
  }

  override mouseup(e: MouseEvent): void {
    this.change = e.button === 2 ? -1 : 1;
    super.mouseup(e);
  }

  override execute(): void {
    this.risq.commands.changeAutoRenew(this.internal_id, this.change);
    this.dataRefreshed();
  }

  override draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void {
    super.draw(ctx, transform, dt);
    const player = this.risq.getPlayer();
    const building = player?.buildings.get(this.internal_id);
    const count = building ? (player?.auto_renewals.get(building.building_id) ?? 0) : 0;
    configDraw(ctx, transform, { fixed_position: true, fill_style: 'black', stroke_width: 0 }, false, false, () => {
      const size = Math.max(10, this.h() * 0.25);
      ctx.fillRect(this.xf() - size * 2, this.yf() - size - 4, size * 2, size + 4);
      drawText(ctx, `${count}`, {
        p: { x: this.xf() - 3, y: this.yf() - 2 },
        w: size * 2 - 6,
        fill_style: 'white',
        align: 'right',
        baseline: 'bottom',
        font: `bold ${size}px serif`,
      });
    });
  }

  protected override getTooltipData(): RisqTooltipData {
    const building = this.risq.getPlayer()?.buildings.get(this.internal_id);
    return {
      title: this.description,
      description: `Left-click to queue; right-click to cancel. Shared by all your ${building?.display_name} buildings.`,
      cost: building?.renew_cost,
    };
  }
}
