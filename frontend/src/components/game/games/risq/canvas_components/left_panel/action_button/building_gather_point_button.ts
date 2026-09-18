import { ColorRGB } from '../../../../../../../scripts/color_rgb';
import type { BoardTransformData } from '../../../../../util/canvas_board/canvas_board';
import type { DwgRisq } from '../../../risq';
import { PLAYER_ICON_SIZE } from '../../../risq_image_cache';
import { RisqActionButton, type RisqActionButtonConfig } from './action_button';

const GATHER_POINT_ICON = 'risq/icons/garrison_flag';
const BLACK = new ColorRGB(0, 0, 0);

export declare interface BuildingGatherPointButtonConfig extends RisqActionButtonConfig {
  building_id: number;
}

export class RisqBuildingGatherPointButton extends RisqActionButton {
  private risq: DwgRisq;
  private building_id: number;

  constructor(config: BuildingGatherPointButtonConfig, risq: DwgRisq, s: number) {
    super(config, s);
    this.risq = risq;
    this.building_id = config.building_id;
  }

  override isClicking(): boolean {
    return super.isClicking() || this.risq.isGatherPointArmed();
  }

  override draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void {
    const icon = this.risq
      .getImageCache()
      .getPlayerColoredIcon(GATHER_POINT_ICON, this.risq.getIcon(GATHER_POINT_ICON), PLAYER_ICON_SIZE, BLACK);
    if (icon) {
      this.setImage(icon);
    }
    super.draw(ctx, transform, dt);
  }

  protected released(): void {
    if (this.isHovering()) {
      this.risq.toggleBuildingGatherPoint(this.building_id);
    }
  }
}
