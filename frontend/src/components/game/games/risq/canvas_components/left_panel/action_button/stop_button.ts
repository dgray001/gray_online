import type { DwgRisq } from '../../../risq';
import { RisqActionButton, type RisqActionButtonConfig } from './action_button';

export declare interface StopButtonConfig extends RisqActionButtonConfig {
  unit_internal_id: number;
}

export class RisqStopButton extends RisqActionButton {
  private risq: DwgRisq;
  private unit_internal_id: number;

  constructor(config: StopButtonConfig, risq: DwgRisq, s: number) {
    super(config, s);
    this.risq = risq;
    this.unit_internal_id = config.unit_internal_id;
  }

  protected released(): void {
    if (this.isHovering()) {
      this.risq.stopUnit(this.unit_internal_id);
    }
  }
}
