export declare interface Point2D {
    x: number;
    y: number;
}
export declare function equalsPoint2D(p1: Point2D | undefined, p2: Point2D | undefined): boolean;
export declare function addPoint2D(p1: Point2D, p2: Point2D): Point2D;
export declare function subtractPoint2D(p1: Point2D, p2: Point2D): Point2D;
export declare function multiplyPoint2D(s: number, p: Point2D): Point2D;
export declare function rotatePoint(p: Point2D, a: number): Point2D;
export declare function roundAxialCoordinate(p: Point2D): Point2D;
export declare const axialDirectionVectors: Point2D[];
export declare function pointInHexagon(p: Point2D, r: number): boolean;
export declare function pointInHexagonalBoard(p: Point2D, board_size: number): boolean;
export declare function coordinateInHexagonalBoard(c: Point2D, board_size: number): boolean;
export declare function hexagonalBoardNeighbors(p: Point2D, board_size: number): Point2D[];
export declare function hexagonalBoardRows(p: Point2D, board_size: number): Point2D[];
