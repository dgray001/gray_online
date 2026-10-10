import type { DwgRisq } from '../../../../risq';
import { RisqActionButton } from '../action_button';
import type { RisqActionButtonConfig } from '../action_button';
import { getSettings } from '../../../../../../../../scripts/settings_store';
import { RisqHotkeyAction } from '../../../../application/input/hotkeys';

export declare interface BuildingUngarrisonButtonConfig extends RisqActionButtonConfig {
  building_id: number;
}

export class RisqBuildingUngarrisonButton extends RisqActionButton {
  private risq: DwgRisq;

  constructor(config: BuildingUngarrisonButtonConfig, risq: DwgRisq, s: number) {
    super(config, s);
    this.risq = risq;
  }

  override dataRefreshed(): void {
    this.setHotkeyCombo(getSettings().risq_hotkeys.actions[RisqHotkeyAction.UNGARRISON]);
  }

  override matchesAction(action: RisqHotkeyAction): boolean {
    return action === RisqHotkeyAction.UNGARRISON;
  }

  override execute(): void {
    for (const id of this.risq.selection.selectedBuildingIds()) {
      this.risq.commands.ungarrisonBuilding(id);
    }
  }
}
