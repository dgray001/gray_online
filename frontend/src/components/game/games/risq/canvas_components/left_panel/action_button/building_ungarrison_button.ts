import type { DwgRisq } from '../../../risq';
import { RisqActionButton, type RisqActionButtonConfig } from './action_button';

export declare interface BuildingUngarrisonButtonConfig extends RisqActionButtonConfig {
  building_id: number;
}

export class RisqBuildingUngarrisonButton extends RisqActionButton {
  private risq: DwgRisq;
  private building_id: number;

  constructor(config: BuildingUngarrisonButtonConfig, risq: DwgRisq, s: number) {
    super(config, s);
    this.risq = risq;
    this.building_id = config.building_id;
  }

  protected released(): void {
    if (this.isHovering()) {
      this.risq.ungarrisonBuilding(this.building_id);
    }
  }
}
