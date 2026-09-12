import type { DwgRisq } from '../../../risq';
import { RisqActionButton, type RisqActionButtonConfig } from './action_button';

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

  protected released(): void {
    if (this.isHovering()) {
      this.risq.confirmDeleteUnit(this.unit_internal_ids);
    }
  }
}
