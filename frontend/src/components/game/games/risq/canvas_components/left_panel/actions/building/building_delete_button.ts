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
  private ctrl_held = false;

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

  override mouseup(e: MouseEvent): void {
    this.ctrl_held = e.ctrlKey;
    super.mouseup(e);
  }

  override execute(ctrl_held: boolean = this.ctrl_held): void {
    this.risq.commands.confirmDeleteBuilding(this.risq.selection.selectedBuildingIds(), ctrl_held);
  }
}

export declare interface BuildingUngarrisonButtonConfig extends RisqActionButtonConfig {
  building_id: number;
}

export class RisqBuildingUngarrisonButton extends RisqActionButton {
  private risq: DwgRisq;

  constructor(config: BuildingUngarrisonButtonConfig, risq: DwgRisq, s: number) {
    super(config, s);
    this.risq = risq;
  }

  override dataRefreshed(): void {
    this.setHotkeyCombo(getSettings().risq_hotkeys.actions[RisqHotkeyAction.UNGARRISON]);
  }

  override matchesAction(action: RisqHotkeyAction): boolean {
    return action === RisqHotkeyAction.UNGARRISON;
  }

  override execute(): void {
    for (const id of this.risq.selection.selectedBuildingIds()) {
      this.risq.commands.ungarrisonBuilding(id);
    }
  }
}
