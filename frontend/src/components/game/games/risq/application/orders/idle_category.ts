import type { RisqBuilding, RisqUnit } from '../../model/types';
import { RisqProducibleKind, RisqUnitType } from '../../model/types';

export type IdleCategory = 'economic' | 'military' | 'unit_production' | 'other_buildings';

export function idleCategory(entity: RisqUnit | RisqBuilding): IdleCategory {
  if ('unit_id' in entity) {
    return entity.unit_type === RisqUnitType.ECONOMIC ? 'economic' : 'military';
  }
  return entity.produces.some((item: { kind: RisqProducibleKind }): boolean => item.kind === RisqProducibleKind.UNIT)
    ? 'unit_production'
    : 'other_buildings';
}
