import type { DwgRisq } from '../../../../risq';
import { RisqActionButton } from '../action_button';
import type { RisqActionButtonConfig } from '../action_button';
import { getSettings } from '../../../../../../../../scripts/settings_store';
import { RisqHotkeyAction } from '../../../../application/input/hotkeys';

export declare interface StopButtonConfig extends RisqActionButtonConfig {
  unit_internal_ids?: number[];
  building_id?: number;
}

export class RisqStopButton extends RisqActionButton {
  private risq: DwgRisq;
  private unit_internal_ids?: number[];
  private building_id?: number;

  constructor(config: StopButtonConfig, risq: DwgRisq, s: number) {
    super(config, s);
    this.risq = risq;
    this.unit_internal_ids = config.unit_internal_ids;
    this.building_id = config.building_id;
  }

  override dataRefreshed(): void {
    this.setHotkeyCombo(getSettings().risq_hotkeys.actions[RisqHotkeyAction.STOP]);
  }

  override matchesAction(action: RisqHotkeyAction): boolean {
    return action === RisqHotkeyAction.STOP;
  }

  override execute(): void {
    if (this.building_id !== undefined) {
      for (const id of this.risq.selection.selectedBuildingIds()) {
        this.risq.commands.stopBuilding(id);
      }
    } else if (this.unit_internal_ids) {
      this.risq.commands.stopUnit(this.unit_internal_ids);
    }
  }
}
