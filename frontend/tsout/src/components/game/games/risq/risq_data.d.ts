import { ColorRGB } from '../../../../scripts/color_rgb';
import { GameBase, GamePlayer } from '../../data_models';
import { Point2D } from '../../util/objects2d';
export declare interface GameRisq {
    game_base: GameBase;
    players: RisqPlayer[];
    scores: GameRisqScoreEntry[];
    board_size: number;
    population_limit: number;
    turn_number: number;
    spaces: RisqSpace[][];
}
export declare interface GameRisqScoreEntry {
    player_id: number;
    nickname: string;
    score: number;
    color: ColorRGB;
}
export declare interface RisqPlayer {
    player: GamePlayer;
    buildings: Map<number, RisqBuilding>;
    units: Map<number, RisqUnit>;
    resources: Map<RisqResourceType, number>;
    population_limit: number;
    score: number;
    color: ColorRGB;
}
export declare enum RisqResourceType {
    ERROR = 0,
    FOOD = 1,
    WOOD = 2,
    STONE = 3,
    GOLD = 4
}
export declare enum RisqTerrainType {
    FLATLANDS = 0,
    HILLY = 1,
    MOUNTAINOUS = 2,
    VALLEY = 3,
    SWAMP = 4,
    SHALLOWS = 5,
    WATER = 6,
    DEEP_WATER = 7
}
export declare function risqTerrainName(terrain: RisqTerrainType): string;
export declare const ZONE_VISIBILITY = 3;
export declare interface RisqSpace {
    terrain: RisqTerrainType;
    coordinate: Point2D;
    visibility: number;
    zones?: RisqZone[][];
    resources?: Map<number, RisqResource>;
    buildings?: Map<number, RisqBuilding>;
    units?: Map<number, RisqUnit>;
    center: Point2D;
    hovered: boolean;
    hovered_neighbor: boolean;
    hovered_row: boolean;
    clicked: boolean;
    num_military_units?: number;
    num_villager_units?: number;
    total_resources?: Map<RisqResourceType, number>;
}
export declare interface RectHoverData {
    ps: Point2D;
    pe: Point2D;
    hovered?: boolean;
    clicked?: boolean;
}
export declare interface EllipHoverData {
    c: Point2D;
    r: Point2D;
    hovered?: boolean;
    clicked?: boolean;
}
export declare interface RisqZone {
    coordinate: Point2D;
    resource?: RisqResource;
    building?: RisqBuilding;
    units: Map<number, RisqUnit>;
    hovered: boolean;
    clicked: boolean;
    hovered_data: EllipHoverData[];
    units_by_type: Map<number, Map<number, UnitByTypeData>>;
    economic_units_by_type: Map<number, UnitByTypeData[]>;
    military_units_by_type: Map<number, UnitByTypeData[]>;
    economic_units: number[];
    military_units: number[];
    reset_hovered_data?: boolean;
}
export declare interface UnitByTypeData {
    player_id: number;
    unit_id: number;
    units: Set<number>;
    hover_data?: RectHoverData;
}
export declare interface RisqUnit {
    internal_id: number;
    player_id: number;
    unit_id: number;
    display_name: string;
    space_coordinate: Point2D;
    zone_coordinate: Point2D;
    speed: number;
    combat_stats: RisqCombatStats;
    hover_data: RectHoverData;
}
export declare interface RisqBuilding {
    internal_id: number;
    player_id: number;
    building_id: number;
    display_name: string;
    space_coordinate: Point2D;
    zone_coordinate: Point2D;
    population_support: number;
    combat_stats: RisqCombatStats;
    hover_data: RectHoverData;
}
export declare interface RisqCombatStats {
    health: number;
    max_health: number;
    attack_type: RisqAttackType;
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
export declare enum RisqAttackType {
    NONE = 0,
    BLUNT = 1,
    PIERCING = 2,
    MAGIC = 3,
    BLUNT_PIERCING = 4,
    PIERCING_MAGIC = 5,
    MAGIC_BLUNT = 6,
    BLUNT_PIERCING_MAGIC = 7
}
export declare interface RisqResource {
    internal_id: number;
    resource_id: number;
    display_name: string;
    space_coordinate: Point2D;
    zone_coordinate: Point2D;
    resources_left: number;
    base_gather_speed: number;
    hover_data: RectHoverData;
}
export declare interface GameRisqFromServer {
    game_base: GameBase;
    players: RisqPlayerFromServer[];
    board_size: number;
    population_limit: number;
    turn_number: number;
    spaces: RisqSpaceFromServer[][];
}
export declare interface RisqPlayerResourcesFromServer {
    wood: number;
    food: number;
    stone: number;
}
export declare interface RisqPlayerFromServer {
    player: GamePlayer;
    buildings: RisqBuildingFromServer[];
    units: RisqUnitFromServer[];
    resources: RisqPlayerResourcesFromServer;
    population_limit: number;
    score: number;
    color: string;
}
export declare interface RisqSpaceFromServer {
    terrain: RisqTerrainType;
    coordinate: Point2D;
    visibility: number;
    zones?: RisqZoneFromServer[][];
    resources?: RisqResourceFromServer[];
    buildings?: RisqBuildingFromServer[];
    units?: RisqUnitFromServer[];
}
export declare interface RisqZoneFromServer {
    coordinate: Point2D;
    building?: RisqBuildingFromServer;
    resource?: RisqResourceFromServer;
    units: RisqUnitFromServer[];
}
export declare interface RisqUnitFromServer {
    internal_id: number;
    player_id: number;
    unit_id: number;
    display_name: string;
    space_coordinate: Point2D;
    zone_coordinate: Point2D;
    speed: number;
    combat_stats: RisqCombatStatsFromServer;
}
export declare interface RisqBuildingFromServer {
    internal_id: number;
    player_id: number;
    building_id: number;
    display_name: string;
    space_coordinate: Point2D;
    zone_coordinate: Point2D;
    population_support: number;
    combat_stats: RisqCombatStatsFromServer;
}
export declare interface RisqCombatStatsFromServer {
    health: number;
    max_health: number;
    attack_type: RisqAttackType;
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
export declare interface RisqResourceFromServer {
    internal_id: number;
    resource_id: number;
    display_name: string;
    space_coordinate: Point2D;
    zone_coordinate: Point2D;
    resources_left: number;
    base_gather_speed: number;
}
export declare function serverToGameRisq(server_game: GameRisqFromServer): GameRisq;
export declare function serverToRisqResources(server_resources: RisqPlayerResourcesFromServer): Map<RisqResourceType, number>;
export declare function serverToRisqPlayer(server_player: RisqPlayerFromServer): RisqPlayer;
export declare function serverToRisqSpace(server_space: RisqSpaceFromServer): RisqSpace;
export declare function serverToRisqZone(server_zone: RisqZoneFromServer): RisqZone;
export declare function serverToRisqBuilding(server_building: RisqBuildingFromServer): RisqBuilding;
export declare function serverToRisqResource(server_resource: RisqResourceFromServer): RisqResource;
export declare function serverToRisqUnit(server_unit: RisqUnitFromServer): RisqUnit;
export declare function getSpace(game: GameRisq, index: Point2D): RisqSpace | undefined;
export declare function coordinateToIndex(board_size: number, coordinate: Point2D): Point2D;
export declare function indexToCoordinate(board_size: number, index: Point2D): Point2D;
export declare interface StartTurnData {
    game: GameRisqFromServer;
}
