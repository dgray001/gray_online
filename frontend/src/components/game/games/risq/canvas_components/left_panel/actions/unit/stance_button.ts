import type { DwgRisq } from '../../../../risq';
import { RisqUnitStance } from '../../../../model/types';
import { RisqActionButton } from '../action_button';
import type { RisqActionButtonConfig } from '../action_button';
import { getSettings } from '../../../../../../../../scripts/settings_store';
import { RisqHotkeyAction } from '../../../../application/input/hotkeys';

const STANCE_HOTKEY_ACTIONS: Record<RisqUnitStance, RisqHotkeyAction | undefined> = {
  [RisqUnitStance.NONE]: undefined,
  [RisqUnitStance.PASSIVE]: RisqHotkeyAction.STANCE_PASSIVE,
  [RisqUnitStance.AGGRESSIVE]: RisqHotkeyAction.STANCE_AGGRESSIVE,
  [RisqUnitStance.DEFENSIVE]: RisqHotkeyAction.STANCE_DEFENSIVE,
  [RisqUnitStance.STAND_GROUND]: RisqHotkeyAction.STANCE_STAND_GROUND,
};

export declare interface StanceButtonConfig extends RisqActionButtonConfig {
  unit_internal_ids: number[];
  stance: RisqUnitStance;
}

export class RisqStanceButton extends RisqActionButton {
  private risq: DwgRisq;
  private unit_internal_ids: number[];
  private stance: RisqUnitStance;
  private active = false;

  constructor(config: StanceButtonConfig, risq: DwgRisq, s: number) {
    super(config, s);
    this.risq = risq;
    this.unit_internal_ids = config.unit_internal_ids;
    this.stance = config.stance;
  }

  override isClicking(): boolean {
    return super.isClicking() || this.active;
  }

  override dataRefreshed(): void {
    const player = this.risq.getPlayer();
    const stances = this.unit_internal_ids.map((id) => player?.units.get(id)?.stance);
    this.active = stances.length > 0 && stances.every((stance) => stance === this.stance);
    const hotkey_action = STANCE_HOTKEY_ACTIONS[this.stance];
    this.setHotkeyCombo(hotkey_action !== undefined ? getSettings().risq_hotkeys.actions[hotkey_action] : undefined);
  }

  override matchesAction(action: RisqHotkeyAction): boolean {
    return action === STANCE_HOTKEY_ACTIONS[this.stance];
  }

  override execute(): void {
    this.risq.commands.setUnitStance(this.unit_internal_ids, this.stance);
  }
}
