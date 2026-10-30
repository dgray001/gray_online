import type { Point2D } from '../../../../util/objects2d';
import { pathPolygon } from './textured_quad';

export type WallDamage = 0 | 1 | 2 | 3 | 4;

export function isRuin(damage: WallDamage): boolean {
  return damage >= 3;
}

export function isWeathered(damage: WallDamage): boolean {
  return damage === 4;
}

const CLEAR_LEFT_HALF: [number, number] = [0.08, 0.36];
const CLEAR_RIGHT_HALF: [number, number] = [0.64, 0.92];
const RUBBLE_CELL = 64;
const RUBBLE_COLUMNS = 4;
const RUBBLE_STONES = 8;
const MERLON_GONE_CHANCE = [0, 0.12, 0.4, 0.8, 0.8];
const MERLON_CHIPPED_CHANCE = [0, 0.3, 0.3, 0.15, 0.15];

export function seededRandom(seed: string): () => number {
  let state = 2166136261;
  for (const char of seed) {
    state = Math.imul(state ^ char.charCodeAt(0), 16777619);
  }
  return (): number => {
    state = (state + 0x6d2b79f5) | 0;
    let t = Math.imul(state ^ (state >>> 15), 1 | state);
    t ^= t + Math.imul(t ^ (t >>> 7), 61 | t);
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

export function merlonScales(damage: WallDamage, count: number, random: () => number): number[] {
  const scales = Array.from({ length: count }, (): number => {
    const roll = random();
    if (roll < MERLON_GONE_CHANCE[damage]) {
      return 0;
    }
    return roll < MERLON_GONE_CHANCE[damage] + MERLON_CHIPPED_CHANCE[damage] ? 0.35 + 0.3 * random() : 1;
  });
  if (damage > 0 && !scales.includes(0)) {
    scales[Math.floor(random() * count)] = 0;
  }
  return scales;
}

function clipToFace(
  ctx: CanvasRenderingContext2D,
  a: Point2D,
  b: Point2D,
  height: number,
  [from, to]: [number, number]
): void {
  const corners = [facePoint(a, b, height, from, 0), facePoint(a, b, height, to, 0)];
  corners.push(facePoint(a, b, height, to, 1), facePoint(a, b, height, from, 1));
  ctx.beginPath();
  corners.forEach((p, i) => (i === 0 ? ctx.moveTo(p.x, p.y) : ctx.lineTo(p.x, p.y)));
  ctx.closePath();
  ctx.clip();
}

export function drawScorch(
  ctx: CanvasRenderingContext2D,
  [a, b]: [Point2D, Point2D],
  height: number,
  damage: WallDamage,
  random: () => number,
  span: [number, number],
  focus: number[]
): void {
  ctx.save();
  clipToFace(ctx, a, b, height, span);
  ctx.transform(b.x - a.x, b.y - a.y, 0, -height, a.x, a.y);
  for (let i = 0; i < 4 + 2 * damage; i++) {
    const streak = focus.length > 0 && damage === 2 && i % 2 === 0;
    ctx.save();
    ctx.translate(
      streak ? focus[(i >> 1) % focus.length] + (random() - 0.5) * 0.03 : random(),
      streak ? 0.8 : 0.55 + 0.5 * random()
    );
    ctx.scale(
      streak ? 0.04 + 0.04 * random() : 0.08 + 0.1 * random(),
      streak ? 1.2 + 0.5 * random() : 0.7 + 0.5 * random()
    );
    const soot = ctx.createRadialGradient(0, 0, 0, 0, 0, 1);
    soot.addColorStop(0, `rgba(18, 12, 8, ${0.6 + 0.15 * Math.min(damage, 2)})`);
    soot.addColorStop(1, 'rgba(18, 12, 8, 0)');
    ctx.fillStyle = soot;
    ctx.fillRect(-1, -1, 2, 2);
    ctx.restore();
  }
  ctx.restore();
}

export function drawCracks(
  ctx: CanvasRenderingContext2D,
  [a, b]: [Point2D, Point2D],
  height: number,
  damage: WallDamage,
  random: () => number,
  span: [number, number]
): void {
  ctx.save();
  clipToFace(ctx, a, b, height, span);
  ctx.strokeStyle = 'rgba(28, 22, 16, 0.85)';
  ctx.lineWidth = Math.max(1, 0.08 * height);
  for (let i = 0; i < [3, 3, 8, 12, 8][damage]; i++) {
    let u = random();
    let v = 1;
    ctx.beginPath();
    ctx.moveTo(...pointArgs(facePoint(a, b, height, u, v)));
    for (let step = 0; step < 4; step++) {
      u += (random() - 0.5) * 0.04;
      v -= 0.12 + 0.2 * random();
      ctx.lineTo(...pointArgs(facePoint(a, b, height, u, v)));
      if (damage >= 2 && step === 1) {
        ctx.lineTo(...pointArgs(facePoint(a, b, height, u + (random() - 0.5) * 0.06, v - 0.2)));
        ctx.moveTo(...pointArgs(facePoint(a, b, height, u, v)));
      }
    }
    ctx.stroke();
  }
  ctx.restore();
}

export declare interface BreachSpec {
  range: [number, number];
  big: boolean;
}

export function breachSpecs(damage: WallDamage, face: number, random: () => number): BreachSpec[] {
  if (damage === 0) {
    return [];
  }
  const place = ([low, high]: [number, number], width: number, big: boolean): BreachSpec => {
    const start = low + (high - low - width) * random();
    return { range: [start, start + width], big };
  };
  if (damage === 1) {
    return [place(face % 2 === 0 ? CLEAR_LEFT_HALF : CLEAR_RIGHT_HALF, 0.1 + 0.04 * random(), false)];
  }
  if (isRuin(damage)) {
    return [{ range: [0, 1], big: true }];
  }
  const big_left = random() < 0.5;
  return [
    place(CLEAR_LEFT_HALF, big_left ? 0.17 + 0.04 * random() : 0.1 + 0.04 * random(), big_left),
    place(CLEAR_RIGHT_HALF, big_left ? 0.1 + 0.04 * random() : 0.17 + 0.04 * random(), !big_left),
  ];
}

export declare interface BreachSample {
  u: number;
  z: number;
}

export function breachFloor(
  [start, end]: [number, number],
  wall_h: number,
  [low, high]: [number, number],
  random: () => number
): BreachSample[] {
  const steps = 7;
  return Array.from({ length: steps + 1 }, (_, i): BreachSample => {
    const depth = low + (high - low) * random();
    return { u: start + ((end - start) * i) / steps, z: wall_h * depth };
  });
}

export function ruinFloor(wall_h: number, random: () => number, wear = 1): BreachSample[] {
  const peaks = Array.from({ length: 3 }, () => ({
    at: 0.12 + 0.76 * random(),
    width: 0.035 + 0.07 * random(),
    height: 0.4 + 0.45 * random(),
  }));
  return Array.from({ length: 29 }, (_, i): BreachSample => {
    const u = i / 28;
    const bump = Math.max(...peaks.map((p) => p.height * Math.exp(-(((u - p.at) / p.width) ** 2))));
    const rough = (0.7 + 0.6 * random()) * bump * wear + 0.12 * (random() - 0.5);
    return { u, z: wall_h * Math.min(0.9, Math.max(0.07, 0.12 + 0.08 * random() + rough)) };
  });
}

export function ruinStones(wall_h: number, count: number, random: () => number): RubbleStone[] {
  const clusters = Array.from({ length: 5 }, () => 0.05 + 0.9 * random());
  return Array.from({ length: count }, (): RubbleStone => {
    const near = random() < 0.65;
    const center = clusters[Math.floor(random() * clusters.length)];
    const u = near ? center + (random() + random() - 1) * 0.07 : random();
    return {
      u: Math.min(1, Math.max(0, u)),
      across: random() < 0.5 ? -0.2 - 0.9 * random() : 1.15 + 1.1 * random(),
      cell: Math.floor(random() * RUBBLE_STONES),
      size: wall_h * (0.25 + 0.7 * random() ** 2),
      mirrored: random() < 0.5,
    };
  });
}

export function floorDepth(damage: WallDamage, big: boolean): [number, number] {
  if (isRuin(damage)) {
    return [0.12, 0.4];
  }
  return big ? [0.5, 0.7] : [0.62, 0.87];
}

export function convexHull(points: Point2D[]): Point2D[] {
  const sorted = [...points].sort((p, q) => p.x - q.x || p.y - q.y);
  const turn = (o: Point2D, a: Point2D, b: Point2D): number => (a.x - o.x) * (b.y - o.y) - (a.y - o.y) * (b.x - o.x);
  const build = (input: Point2D[]): Point2D[] => {
    const chain: Point2D[] = [];
    for (const p of input) {
      while (chain.length >= 2 && turn(chain[chain.length - 2], chain[chain.length - 1], p) <= 0) {
        chain.pop();
      }
      chain.push(p);
    }
    chain.pop();
    return chain;
  };
  return [...build(sorted), ...build([...sorted].reverse())];
}

export declare interface RubbleStone {
  u: number;
  across: number;
  cell: number;
  size: number;
  mirrored: boolean;
}

export function rubbleStones(
  [start, end]: [number, number],
  wall_h: number,
  per_pile: number,
  random: () => number
): RubbleStone[] {
  const piles: [number, number][] = [
    [start - 0.035, 0.05],
    [(start + end) / 2, end - start],
    [end + 0.035, 0.05],
  ];
  return piles.flatMap(([center, spread]) =>
    Array.from(
      { length: per_pile },
      (): RubbleStone => ({
        u: center + (random() - 0.5) * spread,
        across: random() < 0.5 ? -0.2 - 0.5 * random() : 1.15 + 0.6 * random(),
        cell: Math.floor(random() * RUBBLE_STONES),
        size: wall_h * (0.4 + 0.35 * random()),
        mirrored: random() < 0.5,
      })
    )
  );
}

const ruin_atlases = [
  new WeakMap<HTMLImageElement, HTMLCanvasElement>(),
  new WeakMap<HTMLImageElement, HTMLCanvasElement>(),
];
const RUIN_SHADE = ['rgba(0, 0, 0, 0.48)', 'rgba(0, 0, 0, 0.14)'];

export function ruinAtlas(atlas: HTMLImageElement, weathered: boolean): HTMLCanvasElement {
  const shade = weathered ? 1 : 0;
  let dark = ruin_atlases[shade].get(atlas);
  if (!dark) {
    dark = document.createElement('canvas');
    dark.width = atlas.naturalWidth;
    dark.height = atlas.naturalHeight;
    const dark_ctx = dark.getContext('2d')!;
    dark_ctx.drawImage(atlas, 0, 0);
    dark_ctx.globalCompositeOperation = 'source-atop';
    dark_ctx.fillStyle = RUIN_SHADE[shade];
    dark_ctx.fillRect(0, 0, dark.width, dark.height);
    ruin_atlases[shade].set(atlas, dark);
  }
  return dark;
}

export function damageAtlas(atlas: HTMLImageElement, damage: WallDamage): CanvasImageSource {
  return isRuin(damage) ? ruinAtlas(atlas, isWeathered(damage)) : atlas;
}

export function drawRubbleStone(
  ctx: CanvasRenderingContext2D,
  atlas: CanvasImageSource,
  at: Point2D,
  stone: RubbleStone
): void {
  const { cell, size, mirrored } = stone;
  ctx.save();
  ctx.translate(at.x, at.y);
  ctx.scale(mirrored ? -1 : 1, 1);
  const sx = (cell % RUBBLE_COLUMNS) * RUBBLE_CELL;
  const sy = Math.floor(cell / RUBBLE_COLUMNS) * RUBBLE_CELL;
  ctx.drawImage(atlas, sx, sy, RUBBLE_CELL, RUBBLE_CELL, -size / 2, -size * 0.8, size, size);
  ctx.restore();
}

function raggedOutline(points: Point2D[], center: Point2D, random: () => number): Point2D[] {
  return points.flatMap((p, i): Point2D[] => {
    const next = points[(i + 1) % points.length];
    return [0, 1 / 3, 2 / 3].map((t): Point2D => {
      const scale = 0.88 + 0.24 * random();
      const x = p.x + (next.x - p.x) * t;
      const y = p.y + (next.y - p.y) * t;
      return { x: center.x + (x - center.x) * scale, y: center.y + (y - center.y) * scale };
    });
  });
}

function strokeBeam(ctx: CanvasRenderingContext2D, from: Point2D, to: Point2D, width: number): void {
  for (const [color, w, dy] of [
    ['rgb(46, 42, 40)', width, 0],
    ['rgba(128, 118, 108, 0.75)', width * 0.3, -width * 0.25],
  ] as const) {
    ctx.strokeStyle = color;
    ctx.lineWidth = w;
    ctx.beginPath();
    ctx.moveTo(from.x, from.y + dy);
    ctx.lineTo(to.x, to.y + dy);
    ctx.stroke();
  }
}

export function drawCharredOpening(
  ctx: CanvasRenderingContext2D,
  corners: Point2D[],
  random: () => number,
  open = false
): void {
  const center = {
    x: corners.reduce((sum, p) => sum + p.x, 0) / corners.length,
    y: corners.reduce((sum, p) => sum + p.y, 0) / corners.length,
  };
  const outline = raggedOutline(corners, center, random);
  const radius = Math.max(...outline.map((p) => Math.hypot(p.x - center.x, p.y - center.y)));
  const char = ctx.createRadialGradient(center.x, center.y, 0, center.x, center.y, radius);
  char.addColorStop(0, 'rgb(12, 9, 7)');
  char.addColorStop(0.65, 'rgb(30, 22, 16)');
  char.addColorStop(1, 'rgb(62, 44, 28)');
  if (!open) {
    ctx.save();
    pathPolygon(ctx, outline);
    ctx.fillStyle = char;
    ctx.fill();
    ctx.clip();
    for (let i = 0; i < 4; i++) {
      const a = outline[Math.floor(random() * outline.length)];
      const b = outline[Math.floor(random() * outline.length)];
      strokeBeam(ctx, a, b, radius * 0.1);
    }
    ctx.restore();
  }
  pathPolygon(ctx, outline);
  ctx.strokeStyle = 'rgb(22, 16, 11)';
  ctx.lineWidth = Math.max(1, radius * (open ? 0.025 : 0.07));
  ctx.stroke();
  for (let i = 0; i < outline.length; i += 2) {
    const p = outline[i];
    const out = Math.hypot(p.x - center.x, p.y - center.y) || 1;
    ctx.strokeStyle = random() < 0.5 ? 'rgb(150, 120, 70)' : 'rgb(38, 28, 18)';
    ctx.lineWidth = Math.max(1, radius * 0.05);
    ctx.beginPath();
    ctx.moveTo(p.x, p.y);
    ctx.lineTo(p.x + ((p.x - center.x) / out) * radius * 0.2, p.y + ((p.y - center.y) / out) * radius * 0.2);
    ctx.stroke();
  }
}

export function drawSmoke(ctx: CanvasRenderingContext2D, base: Point2D, size: number, random: () => number): void {
  for (let i = 0; i < 6; i++) {
    const x = base.x + Math.sin(i * 1.3) * size * 0.3 + (random() - 0.5) * size * 0.2;
    const y = base.y - i * size * 0.55;
    const radius = size * (0.28 + 0.1 * i);
    const puff = ctx.createRadialGradient(x, y, 0, x, y, radius);
    puff.addColorStop(0, `rgba(72, 68, 64, ${0.9 * (1 - i / 7)})`);
    puff.addColorStop(1, 'rgba(72, 68, 64, 0)');
    ctx.fillStyle = puff;
    ctx.fillRect(x - radius, y - radius, 2 * radius, 2 * radius);
  }
}

function pointArgs(p: Point2D): [number, number] {
  return [p.x, p.y];
}

export function facePoint(a: Point2D, b: Point2D, height: number, u: number, v: number): Point2D {
  return { x: a.x + (b.x - a.x) * u, y: a.y + (b.y - a.y) * u - height * v };
}
