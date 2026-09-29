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
  private building_id: number;

  constructor(config: BuildingUngarrisonButtonConfig, risq: DwgRisq, s: number) {
    super(config, s);
    this.risq = risq;
    this.building_id = config.building_id;
  }

  override dataRefreshed(): void {
    this.setHotkeyCombo(getSettings().risq_hotkeys.actions[RisqHotkeyAction.UNGARRISON]);
  }

  protected released(): void {
    if (this.isHovering()) {
      this.risq.commands.ungarrisonBuilding(this.building_id);
    }
  }
}
