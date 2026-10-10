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
  defaults?: boolean;
}

export class RisqUnitToggleButton extends RisqActionButton {
  private risq: DwgRisq;
  private unit_internal_ids: number[];
  private field: UnitToggleField;
  private active = false;
  private defaults: boolean;

  constructor(config: UnitToggleButtonConfig, risq: DwgRisq, s: number) {
    super(config, s);
    this.risq = risq;
    this.unit_internal_ids = config.unit_internal_ids;
    this.field = config.field;
    this.defaults = config.defaults ?? false;
  }

  override isClicking(): boolean {
    return super.isClicking() || this.active;
  }

  override dataRefreshed(): void {
    const player = this.risq.getPlayer();
    const values = this.defaults
      ? [player?.[this.field === 'attack_back' ? 'default_unit_attack_back' : 'default_unit_interrupt_current']]
      : this.unit_internal_ids.map((id) => player?.units.get(id)?.[this.field]);
    this.active = values.length > 0 && values.every((value) => value === true);
    this.setHotkeyCombo(
      this.defaults ? undefined : getSettings().risq_hotkeys.actions[TOGGLE_HOTKEY_ACTIONS[this.field]]
    );
  }

  override matchesAction(action: RisqHotkeyAction): boolean {
    return action === TOGGLE_HOTKEY_ACTIONS[this.field];
  }

  override execute(): void {
    if (this.defaults) {
      this.risq.commands.setDefaultUnitBehavior({ [this.field]: !this.active });
      return;
    }
    this.risq.commands.setUnitToggle(this.unit_internal_ids, this.field, !this.active);
  }
}
