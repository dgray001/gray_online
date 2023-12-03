
/** Data describing a point in 2d space */
export declare interface Point2D {
  x: number;
  y: number;
}

/** Returns whether the points exist and equal each other */
export function equalsPoint2D(p1: Point2D|undefined, p2: Point2D|undefined): boolean {
  if (!p1 && !p2) {
    return true;
  }
  if (!p1 || !p2) {
    return false;
  }
  return p1.x === p2.x && p1.y === p2.y;
}

/** Returns a point resulting from adding the two points */
export function addPoint2D(p1: Point2D, p2: Point2D): Point2D {
  return {x: p1.x + p2.x, y: p1.y + p2.y};
}

/** Returns a point resulting from subtracting the two points */
export function subtractPoint2D(p1: Point2D, p2: Point2D): Point2D {
  return {x: p1.x - p2.x, y: p1.y - p2.y};
}

/** Returns a point resulting from multiplying the point with a scalar */
export function multiplyPoint2D(s: number, p: Point2D): Point2D {
  return {x: s * p.x, y: s * p.y};
}

/** Returns the rounded integer axial coordinate from a fractional coordinate */
export function roundAxialCoordinate(p: Point2D): Point2D {
  let q = Math.round(p.x);
  let r = Math.round(p.y);
  let s = Math.round(-p.x-p.y);

  const q_diff = Math.abs(q - p.x);
  const r_diff = Math.abs(r - p.y);
  const s_diff = Math.abs(s + p.x + p.y);

  if (q_diff > r_diff && q_diff > s_diff) {
    q = -r-s;
  } else if (r_diff > s_diff) {
    r = -q-s;
  } else {
    s = -q-r;
  }

  return {x: q, y: r};
}

/** Returns the ordered list of axial direction vectors for a hexagonal grid */
export const axialDirectionVectors: Point2D[] = [
  {x: 1, y: 0},
  {x: 1, y: -1},
  {x: 0, y: -1},
  {x: -1, y: 0},
  {x: -1, y: 1},
  {x: 0, y: 1},
];

/** Returns whether the point is in the regular hexagon around the origin */
export function pointInHexagon(p: Point2D, r: number): boolean {
  // largely stolen from http://www.playchilla.com/how-to-check-if-a-point-is-inside-a-hexagon
  const ar = 0.5 * 1.732 * r;
  const q4p = {x: Math.abs(p.x), y: Math.abs(p.y)}; // transform p to Q4
  if (q4p.x > ar || q4p.y > r) {
    return false; // bounding test (since qq is in quadrant 2 only 2 tests are needed)
  }
  return (r * ar - 0.5 * r * q4p.x - ar * q4p.y) >= 0; // finally the dot product can be reduced to this due to the hexagon symmetry
}

/** Returns whether the point is in the hexagonal board */
export function pointInHexagonalBoard(p: Point2D, board_size: number): boolean {
  const c = roundAxialCoordinate(p);
  return coordinateInHexagonalBoard(c, board_size);
}

/** Returns whether the rounded axial coordinate is in the hexagonal board */
export function coordinateInHexagonalBoard(c: Point2D, board_size: number): boolean {
  if (Math.abs(c.x) > board_size || Math.abs(c.y) > board_size || Math.abs(c.x + c.y) > board_size) {
    return false;
  }
  return true;
}

/** Returns neighbor coordinates given axial coordinates and board size */
export function hexagonalBoardNeighbors(p: Point2D, board_size: number): Point2D[] {
  const c = roundAxialCoordinate(p);
  const neighbors: Point2D[] = [];
  for (const v of axialDirectionVectors) {
    const potential_neighbor = addPoint2D(c, v);
    if (coordinateInHexagonalBoard(potential_neighbor, board_size)) {
      neighbors.push(potential_neighbor);
    }
  }
  return neighbors;
}

/** Returns neighbor coordinates given axial coordinates and board size */
export function hexagonalBoardRows(p: Point2D, board_size: number): Point2D[] {
  const c = roundAxialCoordinate(p);
  const neighbors: Point2D[] = [];
  for (const v of axialDirectionVectors) {
    let last_row = c;
    while(true) {
      last_row = addPoint2D(last_row, v);
      if (coordinateInHexagonalBoard(last_row, board_size)) {
        neighbors.push(last_row);
      } else {
        break;
      }
    }
  }
  return neighbors;
}
