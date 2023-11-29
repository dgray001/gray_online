import {GameBase} from '../../data_models';
import {Point2D} from '../../util/objects2d';

/** Data describing a game of risq */
export declare interface GameRisq {
  game_base: GameBase;
  board_size: number;
  spaces: RisqSpace[][];
}

/** Data describing a hexagonal space in risq */
export declare interface RisqSpace {
  coordinate: Point2D;
  center: Point2D;
  hovered: boolean;
}
