import type { DwgRisq } from '../../../../risq';
import { RisqOrderType } from '../../../../model/types';
import { RisqActionButton } from '../action_button';
import { getSettings } from '../../../../../../../../scripts/settings_store';
import { RisqHotkeyAction } from '../../../../application/input/hotkeys';

export declare interface GarrisonButtonConfig {
  row: number;
  col: number;
  unit_internal_ids: number[];
}

export class RisqGarrisonButton extends RisqActionButton {
  private risq: DwgRisq;
  private unit_internal_ids: number[];
  private ungarrison: boolean;

  constructor(config: GarrisonButtonConfig, risq: DwgRisq, s: number) {
    const ungarrison = risq.left_panel.isOnlyGarrisoned();
    super(
      {
        row: config.row,
        col: config.col,
        image_path: ungarrison ? 'icons/ungarrison128' : 'icons/garrison128',
        description: ungarrison ? 'Ungarrison' : 'Garrison',
      },
      s
    );
    this.risq = risq;
    this.unit_internal_ids = config.unit_internal_ids;
    this.ungarrison = ungarrison;
  }

  override isClicking(): boolean {
    return super.isClicking() || this.isArmed();
  }

  private isArmed(): boolean {
    return !this.ungarrison && this.risq.armed.getArmedOrder() === RisqOrderType.OrderType_UnitGarrison;
  }

  override dataRefreshed(): void {
    const hotkey_action = this.ungarrison ? RisqHotkeyAction.UNGARRISON : RisqHotkeyAction.GARRISON;
    this.setHotkeyCombo(getSettings().risq_hotkeys.actions[hotkey_action]);
  }

  override matchesAction(action: RisqHotkeyAction): boolean {
    return this.ungarrison ? action === RisqHotkeyAction.UNGARRISON : action === RisqHotkeyAction.GARRISON;
  }

  override execute(): void {
    if (this.ungarrison) {
      this.risq.commands.ungarrisonUnits(this.unit_internal_ids);
      return;
    }
    if (this.isArmed()) {
      this.risq.armed.disarmOrder();
    } else {
      this.risq.armed.armOrder(RisqOrderType.OrderType_UnitGarrison, () => {});
    }
  }
}
