import {GameBase, GamePlayer} from '../../data_models';
import {Point2D} from '../../util/objects2d';

/** Data describing a game of risq */
export declare interface GameRisq {
  game_base: GameBase;
  players: RisqPlayer[];
  board_size: number;
  spaces: RisqSpace[][];
}

/** Data describing a risq player */
export declare interface RisqPlayer {
  player: GamePlayer;
}

/** Data describing a hexagonal space in risq */
export declare interface RisqSpace {
  coordinate: Point2D;
  visibility: number;
  zones: RisqZone[][];
  center: Point2D;
  // purely frontend fields
  hovered: boolean;
  hovered_neighbor: boolean;
  hovered_row: boolean;
  clicked: boolean;
}

/** Data describing zones inside a risq space */
export declare interface RisqZone {
  coordinate: Point2D;
  units: RisqUnit[];
  building?: RisqBuilding;
}

/** Data describing a risq unit */
export declare interface RisqUnit {
  internal_id: number;
  player_id: number;
  unit_id: number;
  max_health: number;
  speed: number;
  attack_type: number;
  attack_blunt: number;
  attack_piercing: number;
  attack_magic: number;
  defense_blunt: number;
  defense_piercing: number;
  defense_magic: number;
  penetration_blunt: number;
  penetration_piercing: number;
  penetration_magic: number;
}

/** Data describing a risq building */
export declare interface RisqBuilding {
  internal_id: number;
  player_id: number;
  building_id: number;
}
