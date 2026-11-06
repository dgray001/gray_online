import { ColorRGB } from '../../../../../../../../scripts/color_rgb';
import type { BoardTransformData } from '../../../../../../util/canvas_board/canvas_board';
import type { DwgRisq } from '../../../../risq';
import { PLAYER_ICON_SIZE } from '../../../../rendering/assets/image_cache';
import { RisqActionButton } from '../action_button';
import type { RisqActionButtonConfig } from '../action_button';
import { getSettings } from '../../../../../../../../scripts/settings_store';
import { RisqHotkeyAction } from '../../../../application/input/hotkeys';
import type { Point2D } from '../../../../../../util/objects2d';
import type { RisqBuilding } from '../../../../model/types';
import { RisqGatherObjectType, RisqGatherPointLocationKind } from '../../../../model/types';
import { invertPair, invertZoneKey } from '../../../../model/coordinates';
import type { RisqTooltipData } from '../../../risq_tooltip';

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

  protected override getTooltipData(): RisqTooltipData {
    const points = new Set(
      this.risq.selection.selectedBuildings().map((building: RisqBuilding): string => this.gatherPointName(building))
    );
    return { ...super.getTooltipData(), lines: [...points].map((name: string): string => `Gather point: ${name}`) };
  }

  private gatherPointName(building: RisqBuilding): string {
    const point = this.risq.commands.gather_points.point(building);
    if (!point) {
      return 'Not set';
    }
    const session = this.risq.session;
    let space: Point2D | undefined;
    if (point.object_type === RisqGatherObjectType.UNIT) {
      const unit = session.findUnitById(point.object_id);
      space = unit ? session.unitLocation(unit)?.space_coordinate : undefined;
    } else if (point.object_type === RisqGatherObjectType.BUILDING && point.object_id !== 0) {
      space = session.findBuildingById(point.object_id)?.space_coordinate;
    } else {
      space =
        point.location_kind === RisqGatherPointLocationKind.SPACE
          ? invertPair(point.location_id)
          : invertZoneKey(point.location_id).space;
    }
    return space ? (session.getPlaceName(space) ?? 'Unnamed location') : 'Unavailable';
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
    return action === RisqHotkeyAction.BUILDING_GATHER_POINT;
  }

  override execute(): void {
    this.risq.commands.toggleBuildingGatherPoint(this.risq.selection.selectedBuildingIds());
  }
}
