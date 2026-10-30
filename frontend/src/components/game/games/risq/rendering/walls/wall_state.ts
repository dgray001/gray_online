import type { RisqBuilding, RisqZone } from '../../model/types';
import type { RisqDrawHost } from '../board/board_renderer';
import { findOuterZoneIndex } from '../zones/geometry';
import { BUILDING_GHOST_ALPHA, ghostBuildingId } from '../zones/ghost';

export const WALL_BUILDING_ID = 24;

export function isWallId(building_id: number | undefined): boolean {
  return building_id === WALL_BUILDING_ID;
}

export function isCompletedWall(building: RisqBuilding | undefined): boolean {
  return isWallId(building?.building_id) && !building?.under_construction;
}

export function isWalledZone(zone: RisqZone): boolean {
  return isCompletedWall(zone.building);
}

export function isWallRubble(zone: RisqZone): boolean {
  return !zone.building && isWallId(zone.destroyed_building);
}

export function wallAlpha(game: RisqDrawHost, zone: RisqZone): number | undefined {
  if (isCompletedWall(zone.building) || isWallRubble(zone)) {
    return 1;
  }
  return isWallId(ghostBuildingId(game, zone)) ? BUILDING_GHOST_ALPHA : undefined;
}

export function drawsWallModel(game: RisqDrawHost, zone: RisqZone): boolean {
  return findOuterZoneIndex(zone.coordinate) !== -1 && wallAlpha(game, zone) !== undefined;
}
