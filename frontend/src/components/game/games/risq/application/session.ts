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
  RisqZone,
} from '../model/types';
import type { GameRisqFromServer, SubmissionNoticeData } from '../transport/snapshot_types';
import { serverToGameRisq, serverToRisqOrder } from '../transport/snapshot_conversion';
import type { RisqTurnReport } from '../transport/turn_report';
import { replaySnapshot } from './replay/replay_snapshot';

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
  private corpse_locations = new Map<number, { space_coordinate: Point2D; zone: RisqZone }>();
  private game?: GameRisq;
  private region_by_space = new Map<number, RisqRegion>();
  private player_id = -1;
  private last_turn_report?: RisqTurnReport;
  private resource_configs = new Map<number, RisqResourceConfig>();
  private toggling = false;
  private replaying = false;
  private replay_tick = 0;
  private replay_stop = 0;
  private server_game?: GameRisqFromServer;
  private live_game?: GameRisq;

  isReplaying(): boolean {
    return this.replaying;
  }

  setReplaying(replaying: boolean): void {
    this.replaying = replaying;
  }

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

  setPlayerId(player_id: number): void {
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

  replaceSnapshot(new_game: GameRisqFromServer): void {
    this.server_game = new_game;
    this.live_game = serverToGameRisq(new_game);
    this.adoptGame(this.live_game);
    if (this.replaying) {
      this.applyReplayTick(this.replay_tick);
    }
  }

  /** Applies a submission notice to the live game and its raw snapshot; false when it is from another turn or the game ended */
  applySubmission(notice: SubmissionNoticeData): boolean {
    const live = this.live_game;
    const raw = this.server_game;
    if (!live || !raw || notice.turn_number !== live.turn_number || live.game_base.game_ended) {
      return false;
    }
    const orders = notice.active_orders;
    for (const game of new Set([live, this.game])) {
      const player = game?.players[notice.player_id];
      if (game && player) {
        game.giving_orders = notice.giving_orders;
        player.orders_submitted = notice.orders_submitted;
        player.active_orders = orders ? orders.map(serverToRisqOrder).filter((o) => !!o) : player.active_orders;
      }
    }
    raw.giving_orders = notice.giving_orders;
    const raw_player = raw.players[notice.player_id];
    if (raw_player) {
      raw_player.orders_submitted = notice.orders_submitted;
      raw_player.active_orders = orders ?? raw_player.active_orders;
    }
    return true;
  }

  /** The tick whose intents are shown at a half stop, or undefined at a whole stop */
  replayIntentTick(): number | undefined {
    return this.replaying && !Number.isInteger(this.replay_stop) ? Math.ceil(this.replay_stop) : undefined;
  }

  showReplayTick(stop: number): void {
    this.replay_stop = stop;
    const tick = Math.floor(stop);
    if (tick !== this.replay_tick || this.game === this.live_game) {
      this.applyReplayTick(tick);
    }
  }

  endReplay(): void {
    this.adoptGame(this.live_game);
  }

  private applyReplayTick(tick: number): void {
    const journal = this.live_game?.last_turn_journal;
    if (this.server_game && journal) {
      this.replay_tick = tick;
      this.adoptGame(serverToGameRisq(replaySnapshot(this.server_game, journal, tick)));
    }
  }

  private adoptGame(game: GameRisq | undefined): void {
    this.game = game;
    this.region_by_space = buildRegionLookup(this.game?.regions ?? []);
    this.corpse_locations.clear();
    for (const space of this.game?.spaces.flat() ?? []) {
      if (!space?.zones) {
        continue;
      }
      for (const zone of space.zones.flat()) {
        for (const corpse of zone.corpses) {
          this.corpse_locations.set(corpse.internal_id, { space_coordinate: space.coordinate, zone });
        }
      }
    }
  }

  getLastTurnReport(): RisqTurnReport | undefined {
    return this.last_turn_report;
  }

  setLastTurnReport(report: RisqTurnReport | undefined): void {
    this.last_turn_report = report;
  }

  givingOrders(): boolean {
    return (
      !!this.game?.giving_orders && !this.game.game_base.game_ended && !this.getPlayer()?.eliminated && !this.replaying
    );
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
    if (!this.givingOrders()) {
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

  corpseLocation(internal_id: number): { space_coordinate: Point2D; zone: RisqZone } | undefined {
    return this.corpse_locations.get(internal_id);
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

  ownedSpacesCount(player_id: number): number {
    if (!this.game) {
      return 0;
    }
    return this.game.spaces.flat().filter((s) => s?.ownership === player_id).length;
  }
}

/** Whether economic units can exist in the game; false hides every economy-specific display */
export function hasEconomy(game: GameRisq | undefined): boolean {
  return game?.has_economy !== false;
}

/** Whether buildings can exist in the game; false hides every building-specific display */
export function hasBuildings(game: GameRisq | undefined): boolean {
  return game?.has_buildings !== false;
}

/** A game with neither economy nor buildings, where only the military and ownership views make sense */
export function isMilitaryOnly(game: GameRisq | undefined): boolean {
  return !hasEconomy(game) && !hasBuildings(game);
}
