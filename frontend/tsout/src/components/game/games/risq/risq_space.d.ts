import { ColorRGB } from "../../../../scripts/color_rgb";
import { DwgRisq } from "./risq";
import { RisqSpace } from "./risq_data";
export declare enum DrawRisqSpaceDetail {
    OWNERSHIP = 0,
    SPACE_DETAILS = 1,
    ZONE_DETAILS = 2
}
export declare interface DrawRisqSpaceConfig {
    hex_r: number;
    inset_w: number;
    inset_h: number;
    inset_row: number;
    draw_detail: DrawRisqSpaceDetail;
}
export declare function drawRisqSpace(ctx: CanvasRenderingContext2D, game: DwgRisq, space: RisqSpace, config: DrawRisqSpaceConfig): void;
export declare function getSpaceFill(space: RisqSpace, check_hover?: boolean): ColorRGB;
