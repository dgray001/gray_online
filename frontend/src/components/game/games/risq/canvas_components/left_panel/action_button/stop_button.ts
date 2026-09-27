import type { DwgRisq } from '../../../risq';
import { RisqActionButton, type RisqActionButtonConfig } from './action_button';
import { getSettings } from '../../../../../../../scripts/settings_store';
import { RisqHotkeyAction } from '../../../risq_hotkeys';

export declare interface StopButtonConfig extends RisqActionButtonConfig {
  unit_internal_ids: number[];
}

export class RisqStopButton extends RisqActionButton {
  private risq: DwgRisq;
  private unit_internal_ids: number[];

  constructor(config: StopButtonConfig, risq: DwgRisq, s: number) {
    super(config, s);
    this.risq = risq;
    this.unit_internal_ids = config.unit_internal_ids;
  }

  override dataRefreshed(): void {
    this.setHotkeyCombo(getSettings().risq_hotkeys.actions[RisqHotkeyAction.STOP]);
  }

  protected released(): void {
    if (this.isHovering()) {
      this.risq.stopUnit(this.unit_internal_ids);
    }
  }
}
