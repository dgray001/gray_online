import type { Point2D } from '../../../util/objects2d';
import { hexagonalBoardNeighbors, hexagonalBoardRows } from '../../../util/objects2d';
import { coordinateToIndex, getSpace } from '../model/coordinates';
import type {
  GameRisq,
  RisqBuilding,
  RisqPlayer,
  RisqRegion,
  RisqResourceConfig,
  RisqSpace,
  RisqUnit,
} from '../model/types';
import type { GameRisqFromServer } from '../transport/snapshot_types';
import { serverToGameRisq } from '../transport/snapshot_conversion';
import type { RisqTurnReport } from '../transport/turn_report';

/** Maps each explored space's coordinate key to the region it belongs to */
function buildRegionLookup(regions: RisqRegion[]): Map<number, RisqRegion> {
  const lookup = new Map<number, RisqRegion>();
  for (const region of regions) {
    for (const key of region.spaces) {
      lookup.set(key, region);
    }
  }
  return lookup;
}

/** Owns the current game snapshot and the local player's identity within it */
export class RisqSession {
  private game?: GameRisq;
  private region_by_space = new Map<number, RisqRegion>();
  private player_id = -1;
  private last_turn_report?: RisqTurnReport;
  private resource_configs = new Map<number, RisqResourceConfig>();
  private toggling = false;

  isToggling(): boolean {
    return this.toggling;
  }

  setToggling(toggling: boolean): void {
    this.toggling = toggling;
  }

  setResourceConfigs(configs: RisqResourceConfig[]): void {
    this.resource_configs = new Map(configs.map((config) => [config.resource_id, config]));
  }

  getResourceConfigs(): ReadonlyMap<number, RisqResourceConfig> {
    return this.resource_configs;
  }

  setPlayerId(player_id: number) {
    this.player_id = player_id;
  }

  getPlayerId(): number {
    return this.player_id;
  }

  getGame(): GameRisq | undefined {
    return this.game;
  }

  getPlayer(): RisqPlayer | undefined {
    if (!this.game) {
      return undefined;
    }
    return this.player_id > -1 ? this.game.players[this.player_id] : undefined;
  }

  replaceSnapshot(new_game: GameRisqFromServer) {
    this.game = serverToGameRisq(new_game);
    this.region_by_space = buildRegionLookup(this.game?.regions ?? []);
  }

  getLastTurnReport(): RisqTurnReport | undefined {
    return this.last_turn_report;
  }

  setLastTurnReport(report: RisqTurnReport | undefined) {
    this.last_turn_report = report;
  }

  givingOrders(): boolean {
    return this.game?.giving_orders ?? false;
  }

  canGiveOrders(): boolean {
    const player = this.getPlayer();
    const game = this.getGame();
    if (!player || !game) {
      return false;
    }
    if (this.toggling || player.orders_submitted) {
      return false;
    }
    if (!game.giving_orders) {
      return false;
    }
    return true;
  }

  getRegionForSpace(coordinate_key: number): RisqRegion | undefined {
    return this.region_by_space.get(coordinate_key);
  }

  getRegionLookup(): Map<number, RisqRegion> {
    return this.region_by_space;
  }

  findBuildingById(internal_id: number): RisqBuilding | undefined {
    for (const player of this.game?.players ?? []) {
      const building = player.buildings.get(internal_id);
      if (building) {
        return building;
      }
    }
    return undefined;
  }

  findUnitById(internal_id: number): RisqUnit | undefined {
    for (const player of this.game?.players ?? []) {
      const unit = player.units.get(internal_id);
      if (unit) {
        return unit;
      }
    }
    return undefined;
  }

  /** A garrisoned unit has no coordinates of its own, so it's located at its building */
  unitLocation(unit: RisqUnit): { space_coordinate: Point2D; zone_coordinate: Point2D } | undefined {
    return unit.garrisoned_in === undefined ? unit : this.findBuildingById(unit.garrisoned_in);
  }

  spaceAt(coordinate: Point2D): RisqSpace | undefined {
    if (!this.game) {
      return undefined;
    }
    return getSpace(this.game, coordinateToIndex(this.game.board_size, coordinate));
  }

  boardNeighbors(space: RisqSpace): RisqSpace[] {
    if (!this.game) {
      return [];
    }
    return hexagonalBoardNeighbors(space.coordinate, this.game.board_size)
      .map((c) => this.spaceAt(c))
      .filter((s): s is RisqSpace => !!s);
  }

  boardRows(space: RisqSpace): RisqSpace[] {
    if (!this.game) {
      return [];
    }
    return hexagonalBoardRows(space.coordinate, this.game.board_size)
      .map((c) => this.spaceAt(c))
      .filter((s): s is RisqSpace => !!s);
  }
}
