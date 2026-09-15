import type { DwgRisq } from '../../../risq';
import { RisqActionButton, type RisqActionButtonConfig } from './action_button';

export type UnitToggleField = 'interrupt_current' | 'attack_back';

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
  }

  protected released(): void {
    if (this.isHovering()) {
      this.risq.setUnitToggle(this.unit_internal_ids, this.field, !this.active);
    }
  }
}
