import { ColorRGB } from '../../../../../../../../scripts/color_rgb';
import type { BoardTransformData } from '../../../../../../util/canvas_board/canvas_board';
import type { DwgRisq } from '../../../../risq';
import { PLAYER_ICON_SIZE } from '../../../../rendering/assets/image_cache';
import { RisqActionButton } from '../action_button';
import type { RisqActionButtonConfig } from '../action_button';
import { getSettings } from '../../../../../../../../scripts/settings_store';
import { RisqHotkeyAction } from '../../../../application/input/hotkeys';

const GATHER_POINT_ICON = 'risq/icons/garrison_flag';
const BLACK = new ColorRGB(0, 0, 0);

export class RisqBuildingAttackButton extends RisqActionButton {
  private risq: DwgRisq;

  constructor(config: RisqActionButtonConfig, risq: DwgRisq, s: number) {
    super(config, s);
    this.risq = risq;
  }

  override dataRefreshed(): void {
    this.setHotkeyCombo(getSettings().risq_hotkeys.actions[RisqHotkeyAction.BUILDING_ATTACK]);
  }

  override isClicking(): boolean {
    return super.isClicking() || this.risq.armed.isBuildingAttackArmed();
  }

  override matchesAction(action: RisqHotkeyAction): boolean {
    return action === RisqHotkeyAction.BUILDING_ATTACK || action === RisqHotkeyAction.ATTACK;
  }

  override execute(): void {
    this.risq.commands.toggleBuildingAttack();
  }
}

export declare interface BuildingGatherPointButtonConfig extends RisqActionButtonConfig {
  building_id: number;
}

export class RisqBuildingGatherPointButton extends RisqActionButton {
  private risq: DwgRisq;

  constructor(config: BuildingGatherPointButtonConfig, risq: DwgRisq, s: number) {
    super(config, s);
    this.risq = risq;
  }

  override isClicking(): boolean {
    return super.isClicking() || this.risq.armed.isGatherPointArmed();
  }

  override dataRefreshed(): void {
    this.setHotkeyCombo(getSettings().risq_hotkeys.actions[RisqHotkeyAction.BUILDING_GATHER_POINT]);
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

  override matchesAction(action: RisqHotkeyAction): boolean {
    return action === RisqHotkeyAction.BUILDING_GATHER_POINT || action === RisqHotkeyAction.BUILDING_CLEAR_GATHER_POINT;
  }

  override execute(): void {
    this.risq.commands.toggleBuildingGatherPoint(this.risq.selection.selectedBuildingIds());
  }
}
