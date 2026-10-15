import type { RisqPlayer } from '../../model/types';
import { RisqProducibleKind, RisqUnitType } from '../../model/types';
import { capitalize } from '../../../../../../scripts/util';
import type { DwgRisq } from '../../risq';
import type { RisqTooltipData } from '../risq_tooltip';

function unitTypes(player: RisqPlayer): Map<number, RisqUnitType> {
  const types = new Map<number, RisqUnitType>();
  for (const unit of player.units.values()) {
    types.set(unit.unit_id, unit.unit_type);
  }
  const producibles = [...player.buildings.values()].flatMap((b) => b.produces).concat(player.available_mercenaries);
  for (const producible of producibles) {
    if (producible.kind === RisqProducibleKind.UNIT && producible.unit_type !== undefined) {
      types.set(producible.id, producible.unit_type);
    }
  }
  return types;
}

export function populationTooltipData(risq: DwgRisq): RisqTooltipData {
  const player = risq.getPlayer();
  if (!player) {
    return { title: 'Population' };
  }
  const current = new Map<RisqUnitType, number>();
  for (const unit of player.units.values()) {
    current.set(unit.unit_type, (current.get(unit.unit_type) ?? 0) + 1);
  }
  const queued = new Map<RisqUnitType, number>();
  const types = unitTypes(player);
  for (const [id, count] of risq.planning.queuedUnitCounts()) {
    const type = types.get(id) ?? RisqUnitType.NONE;
    queued.set(type, (queued.get(type) ?? 0) + count);
  }
  const queued_total = [...queued.values()].reduce((sum, count) => sum + count, 0);
  const ids = [...new Set([...current.keys(), ...queued.keys()])].sort((a, b) => a - b);
  const type_lines = ids.map((id) => {
    const extra = queued.get(id);
    return `${capitalize(RisqUnitType[id].toLowerCase())}: ${current.get(id) ?? 0}${extra ? ` +${extra}` : ''}`;
  });
  const limit = player.unlimited_population ? '' : ` / ${player.population_limit}`;
  const total = `Total: ${player.units.size}${queued_total ? ` +${queued_total}` : ''}${limit}`;
  return { title: 'Population', lines: [total, ...type_lines] };
}
