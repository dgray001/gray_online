import type { DwgRisq } from '../../../../risq';
import { RisqActionButton } from '../action_button';
import type { RisqActionButtonConfig } from '../action_button';
import { getSettings } from '../../../../../../../../scripts/settings_store';
import { RisqHotkeyAction } from '../../../../application/input/hotkeys';

export declare interface BuildingDeleteButtonConfig extends RisqActionButtonConfig {
  building_id: number;
}

export class RisqBuildingDeleteButton extends RisqActionButton {
  private risq: DwgRisq;

  constructor(config: BuildingDeleteButtonConfig, risq: DwgRisq, s: number) {
    super(config, s);
    this.risq = risq;
  }

  override dataRefreshed(): void {
    this.setHotkeyCombo(getSettings().risq_hotkeys.actions[RisqHotkeyAction.BUILDING_DELETE]);
  }

  override matchesAction(action: RisqHotkeyAction): boolean {
    return action === RisqHotkeyAction.BUILDING_DELETE || action === RisqHotkeyAction.DELETE;
  }

  override execute(): void {
    this.risq.commands.confirmDeleteBuilding(this.risq.selection.selectedBuildingIds());
  }
}
