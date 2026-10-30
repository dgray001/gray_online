import type { RisqZone } from '../../model/types';
import { RisqOrderType } from '../../model/types';
import type { RisqDrawHost } from '../board/board_renderer';

export const BUILDING_GHOST_ALPHA = 0.5;

/** Building to preview in a hovered zone: its foundation's building, else the armed build if the zone is empty */
export function ghostBuildingId(game: RisqDrawHost, zone: RisqZone): number | undefined {
  if (!zone.hovered || zone.resource) {
    return undefined;
  }
  const foundation =
    game.planning.getLocalFoundation(zone.coordinate_key)?.building_id ??
    game.getPlayer()?.planned_foundations?.get(zone.coordinate_key)?.building_id ??
    (zone.building?.under_construction ? zone.building.building_id : undefined);
  if (foundation !== undefined) {
    return foundation;
  }
  const armed =
    game.armed.getArmedOrder() === RisqOrderType.OrderType_UnitBuild ? game.armed.getArmedBuilding() : undefined;
  return zone.building ? undefined : armed?.id;
}
