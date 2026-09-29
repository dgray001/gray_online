import type { DwgRisq } from '../../../../risq';
import { RisqActionButton } from '../action_button';
import type { RisqActionButtonConfig } from '../action_button';
import { getSettings } from '../../../../../../../../scripts/settings_store';
import { RisqHotkeyAction } from '../../../../application/input/hotkeys';

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

  protected released(): void {
    if (this.isHovering()) {
      this.risq.commands.toggleBuildingAttack();
    }
  }
}
