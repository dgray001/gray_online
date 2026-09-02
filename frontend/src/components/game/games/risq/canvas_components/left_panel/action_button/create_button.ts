import type { DwgRisq } from '../../../risq';
import type { RisqActionButtonConfig } from './action_button';
import { RisqActionButton } from './action_button';

export declare interface CreateButtonConfig extends RisqActionButtonConfig {
  building_id: number;
  unit_id: number;
}

export class RisqCreateButton extends RisqActionButton {
  private risq: DwgRisq;
  private building_id: number;
  private unit_id: number;

  constructor(config: CreateButtonConfig, risq: DwgRisq, s: number) {
    super(config, s);
    this.risq = risq;
    this.building_id = config.building_id;
    this.unit_id = config.unit_id;
  }

  protected released(): void {
    if (this.isHovering()) {
      this.risq.createUnit(this.building_id, this.unit_id);
    }
  }
}
