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
      this.dimmed = !canAffordCost(player, this.productionCost());
    } else {
      this.disable();
    }
  }

  override mouseup(e: MouseEvent): void {
    this.ctrl_held = e.ctrlKey;
    super.mouseup(e);
  }

  override matchesProducible(kind: string, id: number): boolean {
    return kind === 'create_unit' && this.producible.id === id;
  }

  override execute(ctrl_held = this.ctrl_held): void {
    if (this.dimmed) {
      this.risq.showMessage('Not enough resources', RISQ_MESSAGE_WARNING_COLOR);
      return;
    }
    for (const id of this.producerIds()) {
      this.risq.commands.createUnit(id, this.producible.id, ctrl_held);
    }
  }

  protected override getTooltipData(): RisqTooltipData {
    return {
      title: this.description,
      description: this.producible.description,
      cost: this.productionCost(),
      stamina_cost: this.producible.stamina_cost,
      hotkey: hotkeyDisplayString(getSettings().risq_hotkeys.create_unit[this.producible.id]),
    };
  }
}
