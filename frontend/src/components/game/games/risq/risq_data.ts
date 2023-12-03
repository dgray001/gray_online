import {ColorRGB} from '../../../../scripts/color_rgb';
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
  // purely frontend fields
  hovered: boolean;
  clicked: boolean;
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

/** Returns the space from the input index, if the space exists */
export function getSpace(game: GameRisq, index: Point2D): RisqSpace|undefined {
  if (index.x < 0 || index.x >= game.spaces.length) {
    return undefined;
  }
  const row = game.spaces[index.x];
  if (index.y < 0 || index.y >= row.length) {
    return undefined;
  }
  return row[index.y];
}

/** Transforms the input coordinate in axial space to index space */
export function coordinateToIndex(board_size: number, coordinate: Point2D): Point2D {
  return {
    x: coordinate.y + board_size,
    y: coordinate.x - Math.max(-board_size, -(board_size + coordinate.y)),
  };
}

/** Transforms the input coordinate in index space to axial space */
export function indexToCoordinate(board_size: number, index: Point2D): Point2D {
  const cy = index.x - board_size;
  return {
    x: index.y + Math.max(-board_size, -(board_size + cy)),
    y: cy,
  };
}

/** Returns the fill color for the input space */
export function getSpaceFill(space: RisqSpace): ColorRGB {
  const color = new ColorRGB(0, 0, 0, 0);
  if (!!space) {
    color.setColor(90, 90, 90, 0.8);
    if (space.visibility > 0) {
      color.setColor(10, 120, 10, 0.8);
      if (space.hovered) {
        if (space.clicked) {
          color.addColor(210, 210, 210, 0.4);
        } else {
          color.addColor(190, 190, 190, 0.2);
        }
      }
    } else if (space.hovered) {
      color.addColor(150, 150, 150, 0.1);
    }
  }
  return color;
}
