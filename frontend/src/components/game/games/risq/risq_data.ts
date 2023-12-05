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
  zones?: RisqZone[][];
  buildings?: Map<number, RisqBuilding>;
  units?: Map<number, RisqUnit>;
  // purely frontend fields
  center: Point2D;
  hovered: boolean;
  hovered_neighbor: boolean;
  hovered_row: boolean;
  clicked: boolean;
}

/** Describes rectangle hover data */
export declare interface RectHoverData {
  ps: Point2D;
  pe: Point2D;
  hovered?: boolean;
  clicked?: boolean;
}

/** Describes ellipse hover data */
export declare interface EllipHoverData {
  c: Point2D;
  r: Point2D;
  hovered?: boolean;
  clicked?: boolean;
}

/** Data describing zones inside a risq space */
export declare interface RisqZone {
  coordinate: Point2D;
  building?: RisqBuilding;
  units: Map<number, RisqUnit>;
  // purely frontend fields
  hovered: boolean;
  clicked: boolean;
  hovered_data: EllipHoverData[];
  units_by_type?: Map<number, UnitByTypeData>; // <unit_id, internal_ids>
}

/** Data describing units_by_type data */
export declare interface UnitByTypeData {
  unit_id: number;
  units: Set<number>; // internal ids
  hover_data?: RectHoverData;
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

/** Data describing a game of risq as returned by server */
export declare interface GameRisqFromServer {
  game_base: GameBase; // alredy been converted
  players: RisqPlayerFromServer[];
  board_size: number;
  spaces: RisqSpaceFromServer[][];
}

/** Data describing a risq player */
export declare interface RisqPlayerFromServer {
  player: GamePlayer;
}

/** Data describing a hexagonal space in risq */
export declare interface RisqSpaceFromServer {
  coordinate: Point2D;
  visibility: number;
  zones?: RisqZoneFromServer[][];
  buildings?: RisqBuildingFromServer[];
  units?: RisqUnitFromServer[];
}

/** Data describing zones inside a risq space */
export declare interface RisqZoneFromServer {
  coordinate: Point2D;
  building?: RisqBuildingFromServer;
  units: RisqUnitFromServer[];
}

/** Data describing a risq unit */
export declare interface RisqUnitFromServer {
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
export declare interface RisqBuildingFromServer {
  internal_id: number;
  player_id: number;
  building_id: number;
}

/** Converts a server response to a frontend risq game */
export function serverToGameRisq(server_game: GameRisqFromServer): GameRisq {
  if (!server_game) {
    return undefined;
  }
  const spaces: RisqSpace[][] = [];
  for (const server_row of server_game.spaces) {
    const row: RisqSpace[] = [];
    for (const space of server_row) {
      row.push(serverToRisqSpace(space));
    }
    spaces.push(row);
  }
  return {
    game_base: server_game.game_base,
    players: server_game.players,
    board_size: server_game.board_size,
    spaces,
  };
}

/** Converts a server response to a frontend risq space */
export function serverToRisqSpace(server_space: RisqSpaceFromServer): RisqSpace {
  if (!server_space) {
    return undefined;
  }
  const space: RisqSpace = {
    coordinate: server_space.coordinate,
    visibility: server_space.visibility,
    // purely frontend fields
    center: {x: 0, y: 0},
    hovered: false,
    hovered_neighbor: false,
    hovered_row: false,
    clicked: false,
  };
  if (!!server_space.zones) {
    const zones: RisqZone[][] = [];
    for (const server_row of server_space.zones) {
      const row: RisqZone[] = [];
      for (const zone of server_row) {
        row.push(serverToRisqZone(zone));
      }
      zones.push(row);
    }
    space.zones = zones;
  }
  if (!!server_space.buildings) {
    space.buildings = new Map(server_space.buildings.map(server_building => [server_building.internal_id, serverToRisqBuilding(server_building)]));
  }
  if (!!server_space.units) {
    space.units = new Map(server_space.units.map(server_unit => [server_unit.internal_id, serverToRisqUnit(server_unit)]));
  }
  return space;
}

/** Converts a server response to a frontend risq zone */
export function serverToRisqZone(server_zone: RisqZoneFromServer): RisqZone {
  if (!server_zone) {
    return undefined;
  }
  return {
    coordinate: server_zone.coordinate,
    building: serverToRisqBuilding(server_zone.building),
    units: new Map(server_zone.units.map(server_unit => [server_unit.internal_id, serverToRisqUnit(server_unit)])),
    // purely frontend fields
    hovered: false,
    clicked: false,
    hovered_data: [],
  };
}

/** Converts a server response to a frontend risq zone */
export function serverToRisqBuilding(server_building: RisqBuildingFromServer): RisqBuilding {
  if (!server_building) {
    return undefined;
  }
  return server_building;
}

/** Converts a server response to a frontend risq zone */
export function serverToRisqUnit(server_unit: RisqUnitFromServer): RisqUnit {
  if (!server_unit) {
    return undefined;
  }
  return server_unit;
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
