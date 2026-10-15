import { drawArrow, drawCircle } from '../../../util/canvas_util';
import type { Point2D } from '../../../util/objects2d';
import { AXIAL_DIRECTION_VECTORS, addPoint2D, equalsPoint2D } from '../../../util/objects2d';
import type { RisqSession } from '../application/session';
import type { RisqSpace, RisqSpaceLink } from '../model/types';
import type { RisqViewport } from './board/viewport';
import { hexEdgeForDirectionIndex } from './space_borders';

const STUB_LENGTH_HEX_R = 0.9;
const CURVE_REACH = 0.35;
const CURVE_SAMPLES = 16;
const FOG_VISIBILITY = 1;
const LINK_COLOR = 'rgba(130, 225, 255, 0.95)';

type LinkViewport = Pick<RisqViewport, 'hexR' | 'hexA' | 'coordinateToCanvas'>;

/** The drawn path of one link crossing: `exit` leaves the first space and `enter` arrives at the second (empty when exit already arrives) */
export declare interface LinkHop {
  exit: Point2D[];
  enter: Point2D[];
}

interface LinkEnd {
  edge: Point2D;
  outward: Point2D;
}

function midpoint(a: Point2D, b: Point2D): Point2D {
  return { x: (a.x + b.x) / 2, y: (a.y + b.y) / 2 };
}

function distance(a: Point2D, b: Point2D): number {
  return Math.hypot(a.x - b.x, a.y - b.y);
}

function sameSpace(a: Point2D, b: Point2D): boolean {
  return equalsPoint2D(a, b);
}

/** Resolves links to canvas geometry, including crossing the board edge when that is the shorter way round */
export class RisqLinks {
  constructor(
    private session: RisqSession,
    private viewport: LinkViewport
  ) {}

  hasLinks(): boolean {
    return (this.session.getGame()?.space_links.length ?? 0) > 0;
  }

  links(): RisqSpaceLink[] {
    return this.session.getGame()?.space_links ?? [];
  }

  private end(space: Point2D, direction: number): LinkEnd {
    const center = this.viewport.coordinateToCanvas(space);
    const [start, finish] = hexEdgeForDirectionIndex(center, this.viewport.hexR(), direction);
    const edge = midpoint(start, finish);
    const length = distance(edge, center);
    return { edge, outward: { x: (edge.x - center.x) / length, y: (edge.y - center.y) / length } };
  }

  private horizontalExtent(): { min: number; max: number } {
    const xs = this.session
      .getGame()!
      .spaces.flat()
      .filter((space): space is RisqSpace => !!space)
      .map((space) => this.viewport.coordinateToCanvas(space.coordinate).x);
    return { min: Math.min(...xs) - this.viewport.hexA(), max: Math.max(...xs) + this.viewport.hexA() };
  }

  /** Whether leaving the board on one side and re-entering on the other beats the direct line */
  private wrapsAround(a: Point2D, b: Point2D): boolean {
    const { min, max } = this.horizontalExtent();
    const around_dx = a.x > b.x ? max - a.x + (b.x - min) : a.x - min + (max - b.x);
    return Math.hypot(around_dx, a.y - b.y) < distance(a, b);
  }

  private stub(end: LinkEnd): Point2D[] {
    const length = STUB_LENGTH_HEX_R * this.viewport.hexR();
    return [end.edge, { x: end.edge.x + end.outward.x * length, y: end.edge.y + end.outward.y * length }];
  }

  /** A cubic that leaves the first edge and arrives at the second along their outward normals */
  private curve(a: LinkEnd, b: LinkEnd): Point2D[] {
    const reach = CURVE_REACH * distance(a.edge, b.edge);
    const c1 = { x: a.edge.x + a.outward.x * reach, y: a.edge.y + a.outward.y * reach };
    const c2 = { x: b.edge.x + b.outward.x * reach, y: b.edge.y + b.outward.y * reach };
    return Array.from({ length: CURVE_SAMPLES + 1 }, (_, i) => {
      const t = i / CURVE_SAMPLES;
      const u = 1 - t;
      const weights = [u * u * u, 3 * u * u * t, 3 * u * t * t, t * t * t];
      return {
        x: weights[0] * a.edge.x + weights[1] * c1.x + weights[2] * c2.x + weights[3] * b.edge.x,
        y: weights[0] * a.edge.y + weights[1] * c1.y + weights[2] * c2.y + weights[3] * b.edge.y,
      };
    });
  }

  private hopOf(link: RisqSpaceLink, forward: boolean): LinkHop {
    const first = forward ? this.end(link.from, link.direction) : this.end(link.to, link.to_direction);
    const second = forward ? this.end(link.to, link.to_direction) : this.end(link.from, link.direction);
    if (this.wrapsAround(first.edge, second.edge)) {
      return { exit: this.stub(first), enter: this.stub(second).reverse() };
    }
    return { exit: this.curve(first, second), enter: [] };
  }

  /** The hop for crossing from space a to space b over a link, when one joins them */
  hopBetween(a: Point2D, b: Point2D): LinkHop | undefined {
    for (const link of this.links()) {
      if (sameSpace(link.from, a) && sameSpace(link.to, b)) {
        return this.hopOf(link, true);
      }
      if (sameSpace(link.from, b) && sameSpace(link.to, a)) {
        return this.hopOf(link, false);
      }
    }
    return undefined;
  }

  private linkedNeighbors(space: Point2D): Point2D[] {
    return this.links().flatMap((link) =>
      sameSpace(link.from, space) ? [link.to] : sameSpace(link.to, space) ? [link.from] : []
    );
  }

  private neighbors(space: Point2D): Point2D[] {
    const adjacent = AXIAL_DIRECTION_VECTORS.map((v) => addPoint2D(space, v)).filter((c) => {
      const neighbor = this.session.spaceAt(c);
      return !!neighbor && !neighbor.terrain_stats?.impassable;
    });
    return [...adjacent, ...this.linkedNeighbors(space)];
  }

  /** Fewest-space route over borders and links; ties keep borders, so a route only crosses a link when it saves steps */
  route(from: Point2D, to: Point2D): Point2D[] {
    const parents = new Map<string, Point2D | undefined>([[`${from.x},${from.y}`, undefined]]);
    const queue = [from];
    for (let head = 0; head < queue.length && !parents.has(`${to.x},${to.y}`); head++) {
      for (const next of this.neighbors(queue[head])) {
        if (!parents.has(`${next.x},${next.y}`)) {
          parents.set(`${next.x},${next.y}`, queue[head]);
          queue.push(next);
        }
      }
    }
    const route: Point2D[] = [];
    for (let at: Point2D | undefined = parents.has(`${to.x},${to.y}`) ? to : undefined; at; ) {
      route.unshift(at);
      at = parents.get(`${at.x},${at.y}`);
    }
    return route;
  }

  private nearestSpace(point: Point2D): Point2D | undefined {
    let best: { space: Point2D; d: number } | undefined;
    for (const space of this.session.getGame()?.spaces.flat() ?? []) {
      const d = space ? distance(this.viewport.coordinateToCanvas(space.coordinate), point) : Infinity;
      if (space && (!best || d < best.d)) {
        best = { space: space.coordinate, d };
      }
    }
    return best?.space;
  }

  /** Polylines drawing a straight order from a to b, rerouted over links when that is the shortest way across the map */
  legs(from: Point2D, to: Point2D): Point2D[][] {
    const from_space = this.hasLinks() ? this.nearestSpace(from) : undefined;
    const to_space = from_space ? this.nearestSpace(to) : undefined;
    if (!from_space || !to_space || sameSpace(from_space, to_space)) {
      return [[from, to]];
    }
    const route = this.route(from_space, to_space);
    const hops = route.slice(1).map((space, i) => ({ a: route[i], b: space, hop: this.hopBetween(route[i], space) }));
    const crossings = hops.filter(
      ({ a, b, hop }) => hop && !AXIAL_DIRECTION_VECTORS.some((v) => sameSpace(addPoint2D(a, v), b))
    );
    if (!crossings.length) {
      return [[from, to]];
    }
    const legs: Point2D[][] = [];
    let start = from;
    for (const { hop } of crossings) {
      legs.push([start, ...hop!.exit]);
      start = hop!.enter.length ? hop!.enter[hop!.enter.length - 1] : hop!.exit[hop!.exit.length - 1];
      if (hop!.enter.length) {
        legs.push(hop!.enter);
      }
    }
    legs.push([start, to]);
    return legs;
  }

  /** The connector lines between linked spaces, with a stub at an end whose partner is unexplored */
  draw(ctx: CanvasRenderingContext2D, scale: number): void {
    ctx.save();
    ctx.strokeStyle = LINK_COLOR;
    ctx.fillStyle = LINK_COLOR;
    ctx.lineWidth = 2.5 / scale;
    for (const link of this.links()) {
      ctx.setLineDash([7 / scale, 5 / scale]);
      this.linkPaths(link).forEach((path) => this.strokePath(ctx, path));
      ctx.setLineDash([]);
      for (const end of [this.end(link.from, link.direction), this.end(link.to, link.to_direction)]) {
        drawCircle(ctx, end.edge, 4 / scale);
      }
    }
    ctx.restore();
  }

  private linkPaths(link: RisqSpaceLink): Point2D[][] {
    const explored = (space: Point2D): boolean => (this.session.spaceAt(space)?.visibility ?? 0) >= FOG_VISIBILITY;
    if (explored(link.from) && explored(link.to)) {
      const hop = this.hopOf(link, true);
      return [hop.exit, hop.enter];
    }
    return [
      explored(link.from)
        ? this.stub(this.end(link.from, link.direction))
        : this.stub(this.end(link.to, link.to_direction)),
    ];
  }

  private strokePath(ctx: CanvasRenderingContext2D, points: Point2D[]): void {
    if (points.length < 2) {
      return;
    }
    ctx.beginPath();
    ctx.moveTo(points[0].x, points[0].y);
    for (const point of points.slice(1)) {
      ctx.lineTo(point.x, point.y);
    }
    ctx.stroke();
  }

  /** Strokes polylines as one order path, ending the last with an arrow head of the given size when it has one */
  drawLegs(ctx: CanvasRenderingContext2D, legs: Point2D[][], arrow_size: number | undefined): void {
    for (const [i, leg] of legs.entries()) {
      const last = i === legs.length - 1;
      const body = last && arrow_size !== undefined ? leg.slice(0, -1) : leg;
      this.strokePath(ctx, body);
      if (last && arrow_size !== undefined && leg.length >= 2) {
        drawArrow(ctx, leg[leg.length - 2], leg[leg.length - 1], arrow_size);
      }
    }
  }
}
