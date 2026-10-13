import type { DwgRisq } from '../../../../risq';
import { RisqActionButton } from '../action_button';
import type { RisqBuilding, RisqCost, RisqProducible } from '../../../../model/types';
import { buildingCanProduce } from '../../../../application/orders/eligibility';
import { canAffordCost } from '../../../../model/types';
import { unitImage } from '../../../../rendering/assets/unit';
import { RISQ_MESSAGE_WARNING_COLOR } from '../../../message_queue';
import type { RisqTooltipData } from '../../../risq_tooltip';
import { getSettings } from '../../../../../../../../scripts/settings_store';
import { hotkeyDisplayString } from '../../../../application/input/hotkeys';

export declare interface CreateButtonConfig {
  building_id: number;
  producible: RisqProducible;
}

export class RisqCreateButton extends RisqActionButton {
  private risq: DwgRisq;
  private producible: RisqProducible;
  private ctrl_held = false;
  private batch_size = 1;

  constructor(config: CreateButtonConfig, risq: DwgRisq, s: number) {
    super(
      {
        row: config.producible.row,
        col: config.producible.col,
        image_path: unitImage(config.producible.id, true),
        description: `Create ${config.producible.display_name}`,
      },
      s
    );
    this.risq = risq;
    this.producible = config.producible;
  }

  private producerIds(): number[] {
    const player = this.risq.getPlayer();
    return this.risq.selection
      .selectedBuildings()
      .filter((building: RisqBuilding): boolean => !!player && buildingCanProduce(player, building, this.producible))
      .map((building: RisqBuilding): number => building.internal_id);
  }

  private productionCost(): RisqCost {
    const count = this.producerIds().length;
    const cost = this.producible.cost;
    return { food: count * cost.food, wood: count * cost.wood, stone: count * cost.stone, gold: count * cost.gold };
  }

  override dataRefreshed(): void {
    const player = this.risq.getPlayer();
    if (!!player && this.risq.session.givingOrders() && !player.orders_submitted) {
      this.enable();
      this.dimmed = !canAffordCost(player, this.producible.cost);
    } else {
      this.disable();
    }
  }

  override mouseup(e: MouseEvent): void {
    this.ctrl_held = e.ctrlKey;
    this.batch_size = e.shiftKey ? 5 : 1;
    super.mouseup(e);
    this.batch_size = 1;
  }

  override matchesProducible(kind: string, id: number): boolean {
    return kind === 'create_unit' && this.producible.id === id;
  }

  override execute(ctrl_held: boolean = this.ctrl_held): void {
    if (this.dimmed) {
      this.risq.showMessage('Not enough resources', RISQ_MESSAGE_WARNING_COLOR);
      return;
    }
    const producer_ids = this.producerIds();
    for (let i = 0; i < this.batch_size; i++) {
      for (const id of producer_ids) {
        if (!this.risq.commands.createUnit(id, this.producible.id, ctrl_held)) {
          return;
        }
      }
    }
  }

  protected override getTooltipData(): RisqTooltipData {
    return {
      title: this.description,
      description: `${this.producible.description} Shift-click to queue five per selected building.`,
      cost: this.productionCost(),
      stamina_cost: this.producible.stamina_cost,
      hotkey: hotkeyDisplayString(getSettings().risq_hotkeys.create_unit[this.producible.id]),
    };
  }
}
