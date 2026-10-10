import type { RisqFrontendOrder, RisqTickAction, RisqUnit } from '../../model/types';
import { RisqOrderType } from '../../model/types';

export function lastAreaAttack(
  unit: RisqUnit,
  order?: Pick<RisqFrontendOrder, 'internal_id' | 'order_type' | 'target_id'>
): RisqTickAction | undefined {
  const actions = unit.tick_actions ?? [];
  const area =
    order ??
    actions
      .slice()
      .reverse()
      .find((action: RisqTickAction): boolean => !!action.order)?.order;
  if (
    !area ||
    area.internal_id === undefined ||
    (area.order_type !== RisqOrderType.OrderType_UnitAttackSpace &&
      area.order_type !== RisqOrderType.OrderType_UnitAttackZone)
  ) {
    return undefined;
  }
  for (let i = actions.length - 1; i >= 0; i--) {
    const action = actions[i];
    if (action.order?.internal_id !== area.internal_id) {
      continue;
    }
    if (action.execute.kind === 'order_cancelled') {
      return undefined;
    }
    const target = action.execute.target;
    if (
      action.execute.kind === 'attack' &&
      action.execute.outcome === 'executed' &&
      (target?.kind === 'unit' || target?.kind === 'building') &&
      target.internal_id !== undefined &&
      action.execute.target_location
    ) {
      return action;
    }
  }
  return undefined;
}
