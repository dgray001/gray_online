import type { DwgRisq } from '../../../risq';
import { RisqActionButton, type RisqActionButtonConfig } from './action_button';
import { getSettings } from '../../../../../../../scripts/settings_store';
import { RisqHotkeyAction } from '../../../risq_hotkeys';

export declare interface BuildingAttackButtonConfig extends RisqActionButtonConfig {
  building_id: number;
}

export class RisqBuildingAttackButton extends RisqActionButton {
  private risq: DwgRisq;
  private building_id: number;

  constructor(config: BuildingAttackButtonConfig, risq: DwgRisq, s: number) {
    super(config, s);
    this.risq = risq;
    this.building_id = config.building_id;
  }

  override dataRefreshed(): void {
    this.setHotkeyCombo(getSettings().risq_hotkeys.actions[RisqHotkeyAction.BUILDING_ATTACK]);
  }

  override isClicking(): boolean {
    return super.isClicking() || this.risq.isBuildingAttackArmed();
  }

  protected released(): void {
    if (this.isHovering()) {
      this.risq.attackFromBuilding(this.building_id);
    }
  }
}
