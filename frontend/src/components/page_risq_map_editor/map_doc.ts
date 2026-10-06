export const DEFAULT_TERRAIN_ID = 1;

export interface MapDocBuilding {
  id: number;
  player: number;
}

export interface MapDocUnit {
  id: number;
  player: number;
  count: number;
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
  direction?: number;
}

export interface MapDocBackgroundImage {
  name: string;
  top_left: [number, number];
  top_right: [number, number];
}

export interface MapDocBank {
  food: number;
  wood: number;
  stone: number;
  gold: number;
}

export interface MapDoc {
  board_size: number;
  players: number;
  starting_bank?: MapDocBank;
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
