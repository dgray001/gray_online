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
  zones: RisqZone[][];
  center: Point2D;
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
}

/** Data describing a risq unit */
export declare interface RisqBuilding {
}
