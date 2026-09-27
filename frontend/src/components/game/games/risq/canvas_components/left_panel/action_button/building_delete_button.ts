import type { DwgRisq } from '../../../risq';
import { RisqActionButton, type RisqActionButtonConfig } from './action_button';
import { getSettings } from '../../../../../../../scripts/settings_store';
import { RisqHotkeyAction } from '../../../risq_hotkeys';

export declare interface BuildingDeleteButtonConfig extends RisqActionButtonConfig {
  building_id: number;
}

export class RisqBuildingDeleteButton extends RisqActionButton {
  private risq: DwgRisq;
  private building_id: number;

  constructor(config: BuildingDeleteButtonConfig, risq: DwgRisq, s: number) {
    super(config, s);
    this.risq = risq;
    this.building_id = config.building_id;
  }

  override dataRefreshed(): void {
    this.setHotkeyCombo(getSettings().risq_hotkeys.actions[RisqHotkeyAction.BUILDING_DELETE]);
  }

  protected released(): void {
    if (this.isHovering()) {
      this.risq.confirmDeleteBuilding(this.building_id);
    }
  }
}
