import type { Point2D } from '../../../../util/objects2d';
import type { SourceRect } from './textured_quad';
import { drawTexturedQuad, lightFor, pathPolygon } from './textured_quad';
import type { WallDrawItem } from './wall_draw';
import type { BreachSample, BreachSpec, RubbleStone } from './wall_damage';
import { drawRubbleStone, seededRandom } from './wall_damage';

export declare interface Breach {
  spec: BreachSpec;
  floor: BreachSample[];
  stones: RubbleStone[];
  seed: string;
}

const CAP_SHADE = 0.95;
const FLOOR_SHADE = 1.08;
const FLOOR_ROCK_SIZE = 0.3;

export declare interface BreachDrawing {
  ctx: CanvasRenderingContext2D;
  stone: HTMLImageElement;
  atlas: CanvasImageSource;
  outer: [Point2D, Point2D];
  inner: [Point2D, Point2D];
  wall_h: number;
  top_z: number;
  edge_unit: Point2D;
  floor: BreachSample[];
  stones: RubbleStone[];
  floor_rocks: number;
  caps: boolean;
  seed: string;
}

function along(line: [Point2D, Point2D], u: number): Point2D {
  return { x: line[0].x + (line[1].x - line[0].x) * u, y: line[0].y + (line[1].y - line[0].y) * u };
}

function lift(p: Point2D, z: number): Point2D {
  return { x: p.x, y: p.y - z };
}

function stoneSource(d: BreachDrawing, across_length: number, down_length: number, random: () => number): SourceRect {
  const density = d.stone.naturalHeight / d.wall_h;
  const w = Math.min(d.stone.naturalWidth, Math.max(8, across_length * density));
  const h = Math.min(d.stone.naturalHeight, Math.max(8, down_length * density));
  return { x: random() * (d.stone.naturalWidth - w), y: random() * (d.stone.naturalHeight - h), w, h };
}

function drawFloor(d: BreachDrawing): void {
  const random = seededRandom(`${d.seed}|floor-texture`);
  for (let i = 0; i < d.floor.length - 1; i++) {
    const [a, b] = [d.floor[i], d.floor[i + 1]];
    const p0 = lift(along(d.outer, a.u), a.z);
    const p1 = lift(along(d.outer, b.u), b.z);
    const q0 = lift(along(d.inner, a.u), a.z);
    const q1 = lift(along(d.inner, b.u), b.z);
    const run = { x: p1.x - p0.x, y: p1.y - p0.y };
    const depth = { x: q0.x - p0.x, y: q0.y - p0.y };
    const source = stoneSource(d, Math.hypot(run.x, run.y), Math.hypot(depth.x, depth.y), random);
    drawTexturedQuad(d.ctx, d.stone, source, p0, run, depth, [p0, p1, q1, q0], FLOOR_SHADE);
  }
}

function drawFloorRocks(d: BreachDrawing): void {
  const random = seededRandom(`${d.seed}|floor-rocks`);
  for (let i = 0; i < d.floor_rocks; i++) {
    const sample = d.floor[1 + Math.floor(random() * (d.floor.length - 2))];
    const ground = { x: 0, y: 0 };
    const p = along(d.outer, sample.u);
    const q = along(d.inner, sample.u);
    const across = 0.25 + 0.5 * random();
    ground.x = p.x + (q.x - p.x) * across;
    ground.y = p.y + (q.y - p.y) * across;
    const stone = {
      u: sample.u,
      across,
      cell: Math.floor(random() * 8),
      size: d.wall_h * FLOOR_ROCK_SIZE,
      mirrored: random() < 0.5,
    };
    drawRubbleStone(d.ctx, d.atlas, lift(ground, sample.z), stone);
  }
}

function drawCap(d: BreachDrawing, sample: BreachSample, normal: Point2D, random: () => number): void {
  const p = along(d.outer, sample.u);
  const q = along(d.inner, sample.u);
  const height = d.top_z - sample.z;
  const jagged = Array.from({ length: 6 }, (_, i): Point2D => {
    const at = along([q, p], i / 5);
    return lift(at, d.top_z - height * (0.05 + 0.4 * random()));
  });
  const outline = [lift(p, sample.z), lift(q, sample.z), ...jagged];
  const source = stoneSource(d, Math.hypot(q.x - p.x, q.y - p.y), height, random);
  const shade = CAP_SHADE * Math.max(0.9, lightFor(normal));
  drawTexturedQuad(
    d.ctx,
    d.stone,
    source,
    lift(p, d.top_z),
    { x: q.x - p.x, y: q.y - p.y },
    { x: 0, y: height },
    outline,
    shade
  );
  d.ctx.save();
  pathPolygon(d.ctx, outline);
  d.ctx.clip();
  const soot = d.ctx.createLinearGradient(0, lift(p, d.top_z).y, 0, lift(p, sample.z).y);
  soot.addColorStop(0, 'rgba(24, 18, 12, 0)');
  soot.addColorStop(1, 'rgba(24, 18, 12, 0.4)');
  d.ctx.fillStyle = soot;
  d.ctx.fillRect(Math.min(p.x, q.x) - 2, 0, Math.abs(q.x - p.x) + 4, 1e5);
  d.ctx.restore();
}

export declare interface WallPiece {
  span: [number, number];
  breach?: number;
}

export function breachPieces(
  ranges: [number, number][],
  outer: [Point2D, Point2D],
  inner: [Point2D, Point2D]
): WallPiece[] {
  const ground_y = ({ span }: WallPiece): number => {
    const u = (span[0] + span[1]) / 2;
    return (along(outer, u).y + along(inner, u).y) / 2;
  };
  const pieces: WallPiece[] = [];
  let cursor = 0;
  ranges.forEach((range, breach) => {
    pieces.push({ span: [cursor, range[0]] }, { span: range, breach });
    cursor = range[1];
  });
  pieces.push({ span: [cursor, 1] });
  return pieces.sort((p, q) => ground_y(p) - ground_y(q));
}

export function drawBreachSurfaces(d: BreachDrawing): void {
  drawFloor(d);
  drawFloorRocks(d);
  const backward = { x: -d.edge_unit.x, y: -d.edge_unit.y };
  const random = seededRandom(`${d.seed}|cap-texture`);
  for (const [sample, normal] of [
    [d.floor[0], d.edge_unit],
    [d.floor[d.floor.length - 1], backward],
  ] as const) {
    if (d.caps && normal.y > 0.02) {
      drawCap(d, sample, normal, random);
    }
  }
}

function rubbleSide(d: BreachDrawing, front: boolean): { at: Point2D; stone: RubbleStone }[] {
  return d.stones
    .map((stone) => {
      const p = along(d.outer, stone.u);
      const q = along(d.inner, stone.u);
      const at = { x: p.x + (q.x - p.x) * stone.across, y: p.y + (q.y - p.y) * stone.across };
      return { at, stone, front: at.y >= (p.y + q.y) / 2 };
    })
    .filter((placed) => placed.front === front);
}

export function drawRubblePile(d: BreachDrawing, front: boolean): void {
  for (const { at, stone } of rubbleSide(d, front)) {
    drawRubbleStone(d.ctx, d.atlas, at, stone);
  }
}

export function rubbleItems(
  drawings: BreachDrawing[],
  anchor: Point2D,
  wall_y: number,
  key: string,
  alpha: number
): WallDrawItem[] {
  return drawings.flatMap((drawing, i) =>
    [false, true].flatMap(
      (front) => rubbleItem(drawing, anchor, front, wall_y, { key: `${key}|${i}|${front ? 'f' : 'b'}`, alpha }) ?? []
    )
  );
}

function rubbleItem(
  d: BreachDrawing,
  anchor: Point2D,
  front: boolean,
  wall_y: number,
  fields: { key: string; alpha: number }
): WallDrawItem | undefined {
  const placed = rubbleSide(d, front);
  if (placed.length === 0) {
    return undefined;
  }
  const boxes = placed.map(({ at, stone }) => ({
    x: at.x - stone.size / 2,
    y: at.y - 0.8 * stone.size,
    size: stone.size,
  }));
  const left = Math.min(...boxes.map((b) => b.x)) - 2;
  const top = Math.min(...boxes.map((b) => b.y)) - 2;
  const right = Math.max(...boxes.map((b) => b.x + b.size)) + 2;
  const bottom = Math.max(...boxes.map((b) => b.y + b.size)) + 2;
  const mean = placed.reduce((sum, { at }) => sum + at.y, 0) / placed.length;
  return {
    y: anchor.y + (front ? Math.max(mean, wall_y + 0.5) : Math.min(mean, wall_y - 0.5)),
    anchor,
    alpha: fields.alpha,
    bounds: { x: left, y: top, w: right - left, h: bottom - top },
    key: fields.key,
    draw: (ctx): void => drawRubblePile({ ...d, ctx }, front),
  };
}
