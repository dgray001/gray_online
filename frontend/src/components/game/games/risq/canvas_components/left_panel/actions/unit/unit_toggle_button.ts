import type { DwgRisq } from '../../../../risq';
import { RisqActionButton } from '../action_button';
import type { RisqActionButtonConfig } from '../action_button';
import { getSettings } from '../../../../../../../../scripts/settings_store';
import { RisqHotkeyAction } from '../../../../application/input/hotkeys';

export type UnitToggleField = 'interrupt_current' | 'attack_back';

const TOGGLE_HOTKEY_ACTIONS: Record<UnitToggleField, RisqHotkeyAction> = {
  interrupt_current: RisqHotkeyAction.TOGGLE_INTERRUPT_CURRENT,
  attack_back: RisqHotkeyAction.TOGGLE_ATTACK_BACK,
};

export declare interface UnitToggleButtonConfig extends RisqActionButtonConfig {
  unit_internal_ids: number[];
  field: UnitToggleField;
}

export class RisqUnitToggleButton extends RisqActionButton {
  private risq: DwgRisq;
  private unit_internal_ids: number[];
  private field: UnitToggleField;
  private active = false;

  constructor(config: UnitToggleButtonConfig, risq: DwgRisq, s: number) {
    super(config, s);
    this.risq = risq;
    this.unit_internal_ids = config.unit_internal_ids;
    this.field = config.field;
  }

  override isClicking(): boolean {
    return super.isClicking() || this.active;
  }

  override dataRefreshed(): void {
    const player = this.risq.getPlayer();
    const values = this.unit_internal_ids.map((id) => player?.units.get(id)?.[this.field]);
    this.active = values.length > 0 && values.every((value) => value === true);
    this.setHotkeyCombo(getSettings().risq_hotkeys.actions[TOGGLE_HOTKEY_ACTIONS[this.field]]);
  }

  protected released(): void {
    if (this.isHovering()) {
      this.risq.commands.setUnitToggle(this.unit_internal_ids, this.field, !this.active);
    }
  }
}
