import type { DwgRisq } from '../../../risq';
import type { RisqUnitStance } from '../../../risq_data';
import { RisqActionButton, type RisqActionButtonConfig } from './action_button';

export declare interface StanceButtonConfig extends RisqActionButtonConfig {
  unit_internal_ids: number[];
  stance: RisqUnitStance;
}

export class RisqStanceButton extends RisqActionButton {
  private risq: DwgRisq;
  private unit_internal_ids: number[];
  private stance: RisqUnitStance;
  private active = false;

  constructor(config: StanceButtonConfig, risq: DwgRisq, s: number) {
    super(config, s);
    this.risq = risq;
    this.unit_internal_ids = config.unit_internal_ids;
    this.stance = config.stance;
  }

  override isClicking(): boolean {
    return super.isClicking() || this.active;
  }

  override dataRefreshed(): void {
    const player = this.risq.getPlayer();
    const stances = this.unit_internal_ids.map((id) => player?.units.get(id)?.stance);
    this.active = stances.length > 0 && stances.every((stance) => stance === this.stance);
  }

  protected released(): void {
    if (this.isHovering()) {
      this.risq.setUnitStance(this.unit_internal_ids, this.stance);
    }
  }
}
