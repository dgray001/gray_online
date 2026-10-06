import type { DwgRisq } from '../../../../risq';
import { RisqActionButton } from '../action_button';
import type { RisqActionButtonConfig } from '../action_button';
import { getSettings } from '../../../../../../../../scripts/settings_store';
import { RisqHotkeyAction } from '../../../../application/input/hotkeys';

export declare interface DeleteButtonConfig extends RisqActionButtonConfig {
  unit_internal_ids: number[];
}

export class RisqDeleteButton extends RisqActionButton {
  private risq: DwgRisq;
  private unit_internal_ids: number[];

  constructor(config: DeleteButtonConfig, risq: DwgRisq, s: number) {
    super(config, s);
    this.risq = risq;
    this.unit_internal_ids = config.unit_internal_ids;
  }

  override dataRefreshed(): void {
    this.setHotkeyCombo(getSettings().risq_hotkeys.actions[RisqHotkeyAction.DELETE]);
  }

  override matchesAction(action: RisqHotkeyAction): boolean {
    return action === RisqHotkeyAction.DELETE;
  }

  override execute(): void {
    this.risq.commands.confirmDeleteUnit(this.unit_internal_ids);
  }
}
