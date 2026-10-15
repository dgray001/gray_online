export const DEFAULT_TERRAIN_ID = 1;

export interface MapDocBuilding {
  id: number;
  player: number;
}

/** A number, or an expression over the setup settings that the editor keeps as written */
export type MapDocValue = number | string;

export interface MapDocUnit {
  id: number;
  player: number;
  count: MapDocValue;
}

export interface MapDocZone {
  x: number;
  y: number;
  resource?: number;
  terrain_override?: number;
  building?: MapDocBuilding;
  units?: MapDocUnit[];
}

export interface MapDocSpace {
  x: number;
  y: number;
  terrain?: number;
  /** Shown as the space's title in place of its terrain's name; at most 40 characters */
  name?: string;
  zones?: MapDocZone[];
  player_slot?: number;
}

type MapDocStartZone = Omit<MapDocZone, 'building' | 'units'> & {
  building?: Omit<MapDocBuilding, 'player'>;
  units?: Omit<MapDocUnit, 'player'>[];
};

export interface MapDocPlayerStart {
  size: number;
  terrain?: number;
  spaces: Array<Omit<MapDocSpace, 'player_slot' | 'zones'> & { zones?: MapDocStartZone[] }>;
}

export interface MapDocRegion {
  name: string;
  gold_bonus?: number;
  spaces: [number, number][];
}

export interface MapDocConnection {
  from: [number, number];
  to: [number, number];
  /** 0 to 5: the edge of `from` this connection leaves through */
  direction: number;
  /** 0 to 5: the edge of `to` it arrives at; the edge opposite `direction` when absent */
  to_direction?: number;
}

export function arrivalDirection(link: MapDocConnection): number {
  return link.to_direction ?? (link.direction + 3) % 6;
}

export interface MapDocBackgroundImage {
  name: string;
  top_left: [number, number];
  top_right: [number, number];
}

export interface MapDocBank {
  food: MapDocValue;
  wood: MapDocValue;
  stone: MapDocValue;
  gold: MapDocValue;
}

export interface MapDoc {
  board_size: number;
  players: number;
  starting_bank?: MapDocBank;
  rotation?: number;
  config_dir?: string;
  background_image?: MapDocBackgroundImage;
  player_start?: MapDocPlayerStart;
  spaces: MapDocSpace[];
  regions?: MapDocRegion[];
  connections?: MapDocConnection[];
}

export function findSpace(doc: MapDoc, x: number, y: number): MapDocSpace | undefined {
  return doc.spaces.find((space) => space.x === x && space.y === y);
}

export function newMapDoc(board_size: number, players: number): MapDoc {
  const spaces: MapDocSpace[] = [];
  for (let x = -board_size; x <= board_size; x++) {
    for (let y = Math.max(-board_size, -x - board_size); y <= Math.min(board_size, -x + board_size); y++) {
      spaces.push({ x, y, terrain: DEFAULT_TERRAIN_ID });
    }
  }
  return { board_size, players, spaces };
}

export const MAP_NAME_PATTERN = /^[a-z0-9_]{1,40}$/;
export const MAX_BOARD_SIZE = 10;

export function newMapError(name: string, size: number, existing_names: string[]): string | undefined {
  if (!MAP_NAME_PATTERN.test(name)) {
    return 'Enter a map name: lowercase letters, digits and underscores';
  }
  if (existing_names.includes(name)) {
    return `A map named ${name} already exists`;
  }
  if (!(size >= 0 && size <= MAX_BOARD_SIZE)) {
    return `Size must be 0 to ${MAX_BOARD_SIZE}`;
  }
  return undefined;
}
