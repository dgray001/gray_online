import { ColorRGB } from "../../../../scripts/color_rgb";
import { Point2D } from "../../util/objects2d";
import { DwgRisq } from "./risq";
import { RisqSpace, RisqUnit, RisqZone, UnitByTypeData } from "./risq_data";
export declare const INNER_ZONE_MULTIPLIER = 0.4;
export declare function organizeZoneUnits(units: Map<number, RisqUnit>): Map<number, Map<number, UnitByTypeData>>;
export declare function getZoneFill(zone: RisqZone, check_hover?: boolean, alpha_multiplier?: number): ColorRGB;
export declare function drawRisqZone(ctx: CanvasRenderingContext2D, game: DwgRisq, zone: RisqZone, black_text: boolean, r: number, rotation: number, p1: Point2D, p2: Point2D, p3: Point2D): void;
export declare function resolveHoveredZones(p: Point2D, space: RisqSpace, r: number, override_center?: Point2D, ignore_parts?: boolean): RisqZone | undefined;
export declare function unhoverRisqZone(zone: RisqZone): void;
