import type {
  RisqBuilding,
  RisqRegion,
  RisqResource,
  RisqSpace,
  RisqUnit,
  RisqZone,
  UnitByTypeData,
} from '../../risq_data';

/** Config for the left panel */
export declare interface LeftPanelConfig {
  w: number;
  background: string;
}

/** Possible objects in a space/zone that can be hovered */
export enum HoverableObjectType {
  NONE,
  UNIT,
  BUILDING,
  RESOURCE,
}

/** A hovered object paired with its type, so consumers narrow to the concrete type without casting */
export type HoveredObject =
  | { type: HoverableObjectType.UNIT; object: RisqUnit }
  | { type: HoverableObjectType.BUILDING; object: RisqBuilding }
  | { type: HoverableObjectType.RESOURCE; object: RisqResource };

export interface UnitsDrawData {
  space: RisqSpace;
  units_by_player: Map<number, UnitByTypeData[]>;
}

export interface PlayerUnitsDrawData {
  space?: RisqSpace;
  units: UnitByTypeData[];
}

export interface MultiplePlayersUnitsDrawData {
  space: RisqSpace;
  units_by_player: [number, UnitByTypeData[]][];
}

/** All the data types that can be displayed in the left panel */
export enum LeftPanelDataType {
  RESOURCE,
  BUILDING,
  SPACE,
  ZONE,
  MULTIPLE_PLAYERS_UNITS,
  UNITS,
  UNITS_BY_TYPE, // must all be of same player
  ECONOMIC_UNITS,
  MILITARY_UNITS,
  UNIT,
  FOUNDATION,
  REGION,
}

export interface ResourceData {
  data_type: LeftPanelDataType.RESOURCE;
  data: RisqResource;
}

export interface BuildingData {
  data_type: LeftPanelDataType.BUILDING;
  data: RisqBuilding;
}

export interface SpaceData {
  data_type: LeftPanelDataType.SPACE;
  data: RisqSpace;
}

export interface ZoneData {
  data_type: LeftPanelDataType.ZONE;
  data: {
    space: RisqSpace;
    zone: RisqZone;
  };
}

export interface MultiplePlayersUnitsData {
  data_type: LeftPanelDataType.MULTIPLE_PLAYERS_UNITS;
  data: MultiplePlayersUnitsDrawData;
}

export interface UnitsData {
  data_type: LeftPanelDataType.UNITS;
  data: UnitsDrawData;
}

export interface UnitsByTypeData {
  data_type: LeftPanelDataType.UNITS_BY_TYPE;
  data: PlayerUnitsDrawData;
}

export interface EconomicUnitsData {
  data_type: LeftPanelDataType.ECONOMIC_UNITS;
  data: PlayerUnitsDrawData;
}

export interface MilitaryUnitsData {
  data_type: LeftPanelDataType.MILITARY_UNITS;
  data: PlayerUnitsDrawData;
}

export interface UnitData {
  data_type: LeftPanelDataType.UNIT;
  data: RisqUnit;
}

export interface FoundationDrawData {
  coordinate_key: number;
  building_id: number;
  player_id: number;
  is_local: boolean;
  display_name: string;
  zone: RisqZone;
}

export interface FoundationData {
  data_type: LeftPanelDataType.FOUNDATION;
  data: FoundationDrawData;
}

export interface RegionData {
  data_type: LeftPanelDataType.REGION;
  data: RisqRegion;
}

export type LeftPanelData =
  | ResourceData
  | BuildingData
  | SpaceData
  | ZoneData
  | MultiplePlayersUnitsData
  | UnitsData
  | UnitsByTypeData
  | EconomicUnitsData
  | MilitaryUnitsData
  | UnitData
  | FoundationData
  | RegionData;
