import type { DwgRisq } from '../../risq';
import type { BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import { configDraw } from '../../../../util/canvas_components/canvas_component';
import { drawText } from '../../../../util/canvas_util';
import { RisqRange, RisqResourceType } from '../../model/types';
import { resourceTypeImage } from '../../rendering/assets/resources';
import type { RisqProducible } from '../../model/types';
import { unitImage } from '../../rendering/assets/unit';
import { RisqActionButton } from '../left_panel/actions/action_button';
import type { RisqTooltipData } from '../risq_tooltip';
import { getSettings } from '../../../../../../scripts/settings_store';
import { hotkeyDisplayString } from '../../application/input/hotkeys';

export class RisqMercenaryButton extends RisqActionButton {
  private risq: DwgRisq;
  private mercenary: RisqProducible;

  constructor(risq: DwgRisq, mercenary: RisqProducible, s: number) {
    const description = `Hire ${mercenary.display_name}`;
    super({ row: 0, col: 0, image_path: unitImage(mercenary.id, true), description }, s);
    this.risq = risq;
    this.mercenary = mercenary;
  }

  override isClicking(): boolean {
    return super.isClicking() || this.risq.armed.getArmedMercenaryId() === this.mercenary.id;
  }

  override dataRefreshed(): void {
    const player = this.risq.getPlayer();
    if (!!player && this.risq.session.givingOrders() && !player.orders_submitted) {
      this.enable();
      this.dimmed = !!this.risq.planning.mercenaryUnavailableReason(this.mercenary);
    } else {
      this.disable();
    }
  }

  protected released(): void {
    if (this.isHovering()) {
      this.risq.commands.toggleMercenary(this.mercenary);
    }
  }

  drawCost(ctx: CanvasRenderingContext2D, transform: BoardTransformData): void {
    const config = { fill_style: 'transparent', stroke_width: 0, fixed_position: true };
    configDraw(ctx, transform, config, false, false, () => {
      ctx.drawImage(this.risq.getIcon(resourceTypeImage(RisqResourceType.GOLD)), this.xi(), this.yf() + 4, 12, 12);
      drawText(ctx, this.mercenary.cost.gold.toFixed(1), {
        p: { x: this.xi() + 15, y: this.yf() + 4 },
        w: this.w() - 15,
        fill_style: 'black',
        align: 'left',
        baseline: 'top',
        font: '12px serif',
      });
    });
  }

  private tooltipStats(): [string, number][] {
    const stats = this.mercenary.stats;
    if (!stats) {
      return [];
    }
    const values: [string, number][] = [
      ['risq/icons/attack_blunt', stats.attack_blunt],
      ['risq/icons/attack_piercing', stats.attack_piercing],
      ['risq/icons/defense_blunt', stats.defense_blunt],
      ['risq/icons/defense_piercing', stats.defense_piercing],
      ['risq/icons/penetration_blunt', stats.penetration_blunt],
      ['risq/icons/penetration_piercing', stats.penetration_piercing],
    ];
    if (stats.attack_range >= RisqRange.SPACE) {
      values.push(['risq/icons/attack_range', stats.attack_range - RisqRange.SPACE]);
    }
    return values.filter(([icon, value]) => value !== 0 || icon === 'risq/icons/attack_range');
  }

  protected override getTooltipData(): RisqTooltipData {
    return {
      title: this.description,
      description: `${this.mercenary.description}${this.mercenary.stats ? ` Health: ${this.mercenary.stats.health}.` : ''}`,
      cost: this.mercenary.cost,
      stats: this.tooltipStats(),
      hotkey: hotkeyDisplayString(getSettings().risq_hotkeys.hire_mercenary[this.mercenary.id]),
    };
  }
}
