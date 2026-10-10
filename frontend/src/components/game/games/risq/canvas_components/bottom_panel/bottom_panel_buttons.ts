import type { DwgRisq } from '../../risq';
import { RisqMinimapCornerButton } from '../minimap/corner_button';
import { RisqActionButton } from '../left_panel/actions/action_button';
import { RISQ_MESSAGE_WARNING_COLOR } from '../message_queue';
import { getSettings } from '../../../../../../scripts/settings_store';
import { RisqHotkeyAction } from '../../application/input/hotkeys';
import { techImage } from '../../rendering/assets/buildings';

export class RisqMercenaryPanelButton extends RisqMinimapCornerButton {
  constructor(private risq: DwgRisq) {
    super(risq, { row: 0, col: 0, image_path: techImage(4), description: 'Mercenaries' }, Math.PI);
  }
  protected released(): void {
    if (this.isHovering()) {
      this.risq.toggleMercenaryPanel();
    }
  }
}

export class RisqDefaultBehaviorButton extends RisqMinimapCornerButton {
  constructor(private risq: DwgRisq) {
    super(
      risq,
      { row: 0, col: 0, image_path: 'icons/unit64', description: 'New military unit behavior' },
      -Math.PI / 2
    );
  }
  protected released(): void {
    if (this.isHovering()) {
      this.risq.toggleDefaultBehaviorPanel();
    }
  }
}

export class RisqSummaryReportButton extends RisqActionButton {
  private risq: DwgRisq;

  constructor(risq: DwgRisq) {
    super({ row: 0, col: 0, image_path: 'icons/report64', description: 'Review last turn report' }, 30);
    this.risq = risq;
  }

  override dataRefreshed(): void {
    this.dimmed = !this.risq.session.getLastTurnReport();
    this.setHotkeyCombo(getSettings().risq_hotkeys.actions[RisqHotkeyAction.SUMMARY_REPORT]);
  }

  protected released(): void {
    if (!this.isHovering()) {
      return;
    }
    if (this.dimmed) {
      this.risq.showMessage('No previous turn report to review', RISQ_MESSAGE_WARNING_COLOR);
    } else {
      this.risq.reviewLastTurnReport();
    }
  }
}

export class RisqViewModeButton extends RisqMinimapCornerButton {
  private risq: DwgRisq;

  constructor(risq: DwgRisq) {
    super(risq, { row: 0, col: 0, image_path: 'icons/eye64', description: 'Toggle view mode' }, 0);
    this.risq = risq;
  }

  override dataRefreshed(): void {
    this.setHotkeyCombo(getSettings().risq_hotkeys.actions[RisqHotkeyAction.CYCLE_VIEW_MODE]);
  }

  protected released(): void {
    if (this.isHovering()) {
      this.risq.viewport.cycleViewMode();
    }
  }
}

export class RisqTechTreeButton extends RisqMinimapCornerButton {
  private risq: DwgRisq;

  constructor(risq: DwgRisq) {
    super(risq, { row: 0, col: 0, image_path: 'icons/research64', description: 'Open tech tree' }, Math.PI / 2);
    this.risq = risq;
  }

  override dataRefreshed(): void {
    this.setHotkeyCombo(getSettings().risq_hotkeys.actions[RisqHotkeyAction.TECH_TREE]);
  }

  protected released(): void {
    if (this.isHovering()) {
      this.risq.openTechTree();
    }
  }
}
