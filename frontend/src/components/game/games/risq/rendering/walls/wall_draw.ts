import type { Point2D } from '../../../../util/objects2d';
import type { WallGeometry, WallSegmentInput } from './wall_geometry';
import { wallGeometry } from './wall_geometry';
import type { WallMetrics } from './wall_metrics';
import { HEIGHT_SCALE, TOWER_SCALE, WALL_MERLONS } from './wall_metrics';
import type { SourceRect } from './textured_quad';
import { drawTexturedQuad, drawTiledQuad, lightFor, pathPolygon } from './textured_quad';
import type { DecorContext } from './wall_decor';
import { drawWallBanners, drawWallFlags } from './wall_decor';
import { drawPillar } from './wall_pillar';
import { drawSupports } from './wall_supports';
import type { BreachSample } from './wall_damage';
import {
  convexHull,
  damageAtlas,
  drawCracks,
  drawScorch,
  isRuin,
  isWeathered,
  merlonScales,
  seededRandom,
} from './wall_damage';
import { buildBreaches, floorRockCount } from './wall_ruin';
import type { Breach, BreachDrawing } from './wall_breach';
import { breachPieces, drawBreachSurfaces, rubbleItems } from './wall_breach';
import type { TowerTextures } from './tower_draw';
import { TOWER_REACH, drawTower } from './tower_draw';

export declare interface WallTextures {
  stone: HTMLImageElement;
  deck: HTMLImageElement;
  rubble: HTMLImageElement;
  tower: TowerTextures;
}

export declare interface WallDrawItem {
  y: number;
  anchor: Point2D;
  alpha: number;
  bounds: { x: number; y: number; w: number; h: number };
  key: string;
  draw: (ctx: CanvasRenderingContext2D) => void;
}

const TOP_COLOR = 'rgb(168, 158, 138)';
const TOP_EDGE = 'rgb(100, 95, 83)';
const BACKING = 'rgb(70, 66, 58)';
const WEATHERED_LIGHT = 1.15;

function lift(point: Point2D, z: number): Point2D {
  return { x: point.x, y: point.y - z };
}

function along(a: Point2D, b: Point2D, u: number, shift?: Point2D): Point2D {
  return { x: a.x + (b.x - a.x) * u + (shift?.x ?? 0), y: a.y + (b.y - a.y) * u + (shift?.y ?? 0) };
}

function faceRect(image: HTMLImageElement, top: number, bottom: number): SourceRect {
  return { x: 0, y: top * image.naturalHeight, w: image.naturalWidth, h: (bottom - top) * image.naturalHeight };
}

class WallPainter {
  private readonly geometry: WallGeometry;
  private readonly edge: Point2D;
  private readonly edge_unit: Point2D;
  private readonly n: Point2D;
  private readonly inward_normal: Point2D;
  private readonly front: boolean;
  private readonly back: boolean;
  private readonly seed: string;
  private readonly merlon_scale: number[];
  private readonly breaches: Breach[];

  private ctx!: CanvasRenderingContext2D;
  private span: [number, number] = [0, 1];

  private partOf(line: [Point2D, Point2D]): [Point2D, Point2D] {
    return [along(line[0], line[1], this.span[0]), along(line[0], line[1], this.span[1])];
  }

  private spans(u: number, half = 0): boolean {
    return u - half >= this.span[0] - 1e-9 && u + half <= this.span[1] + 1e-9;
  }

  constructor(
    private tex: WallTextures,
    private m: WallMetrics,
    private segment: WallSegmentInput
  ) {
    this.geometry = wallGeometry(segment, m.parapet_depth, m.deck_depth);
    const [a, b] = this.geometry.outer;
    this.edge = { x: b.x - a.x, y: b.y - a.y };
    const length = Math.hypot(this.edge.x, this.edge.y);
    this.edge_unit = { x: this.edge.x / length, y: this.edge.y / length };
    this.n = segment.normal;
    this.inward_normal = { x: -this.n.x, y: -this.n.y };
    this.front = this.n.y > 0.02;
    this.back = this.n.y < -0.02;
    this.seed = `face${segment.face}|${segment.damage}`;
    this.merlon_scale = merlonScales(segment.damage, WALL_MERLONS, seededRandom(this.seed));
    this.breaches = buildBreaches(segment.damage, segment.face, m.wall_h, m.wall_h / HEIGHT_SCALE);
  }

  private drawLow(floor: BreachSample[]): void {
    if (Math.abs(this.n.y) <= 0.3) {
      this.drawBacking(Math.min(...floor.map((s) => s.z)));
    }
    const face = this.front ? this.geometry.outer : this.back ? this.geometry.inner : undefined;
    const normal = this.front ? this.n : this.inward_normal;
    for (let i = 0; face && i < floor.length - 1; i++) {
      this.span = [floor[i].u, floor[i + 1].u];
      this.drawFace(face, normal, [floor[i].z, floor[i + 1].z]);
    }
  }

  draw(drawings: BreachDrawing[]): void {
    const ranges = this.breaches.map((b) => b.spec.range);
    for (const { span, breach } of breachPieces(ranges, this.geometry.outer, this.geometry.inner)) {
      this.span = span;
      if (breach === undefined) {
        this.drawSpan();
      } else {
        this.drawLow(this.breaches[breach].floor);
        drawBreachSurfaces(drawings[breach]);
      }
    }
    this.span = [0, 1];
  }

  private drawSpan(): void {
    const { outer, parapet, inner } = this.geometry;
    if (Math.abs(this.n.y) <= 0.3) {
      this.drawBacking();
    }
    if (this.front) {
      this.drawDeck(parapet, inner);
      this.drawFace(outer, this.n);
      this.drawBanners(outer, this.n);
      this.drawParapetBand(outer, this.n);
      this.drawParapetTop();
      this.drawMerlons();
    } else if (this.back) {
      this.drawParapetTop();
      this.drawMerlons();
      this.drawParapetBand(parapet, this.inward_normal);
      this.drawDeck(parapet, inner);
      this.drawFace(inner, this.inward_normal);
      drawSupports(this.ctx, this.m, inner, this.span);
      this.drawBanners(inner, this.inward_normal);
    } else {
      this.drawDeck(parapet, inner);
      this.drawParapetTop();
      this.drawMerlons();
    }
    this.drawFlags();
    this.drawPillars();
  }

  private decor(): DecorContext {
    const { color, damage } = this.segment;
    return { ctx: this.ctx, m: this.m, color, damage, span: this.span, seed: this.seed };
  }

  private drawBanners(line: [Point2D, Point2D], normal: Point2D): void {
    drawWallBanners(this.decor(), line, normal);
  }

  private drawFlags(): void {
    drawWallFlags(this.decor(), this.geometry.outer, this.geometry.inward);
  }

  private drawPillars(): void {
    const { parapet_depth, deck_depth } = this.m;
    const total = parapet_depth + deck_depth;
    const [a, b] = this.geometry.outer;
    const reach = total / Math.hypot(b.x - a.x, b.y - a.y);
    const ends: [boolean, number, number, -1 | 1][] = [
      [this.segment.start_neighbor_normal === undefined && this.span[0] === 0, 0, reach, -1],
      [this.segment.end_neighbor_normal === undefined && this.span[1] === 1, 1 - reach, 1, 1],
    ];
    const big = this.breaches.find((b) => b.spec.big);
    const ruined = isRuin(this.segment.damage);
    const weathered = isWeathered(this.segment.damage);
    const heading = ruined
      ? this.segment.face % 2 === 0
        ? -1
        : 1
      : big && big.spec.range[0] + big.spec.range[1] > 1
        ? 1
        : -1;
    for (const [free, u0, u1, side] of ends) {
      if (!free) {
        continue;
      }
      const a_side = side === -1 ? 1 : 0;
      const broken =
        ruined || (big && heading === side)
          ? {
              a_side,
              atlas: damageAtlas(this.tex.rubble, this.segment.damage),
              ruin: ruined && heading === side,
              weathered,
            }
          : undefined;
      const at = (u: number, depth: number): Point2D =>
        along(a, b, u, { x: this.geometry.inward.x * depth, y: this.geometry.inward.y * depth });
      const normals = [this.n, this.edge_unit, this.inward_normal, { x: -this.edge_unit.x, y: -this.edge_unit.y }];
      drawPillar(this.ctx, this.tex.stone, this.m, {
        corners: [at(u0, 0), at(u1, 0), at(u1, total), at(u0, total)],
        normals: normals as [Point2D, Point2D, Point2D, Point2D],
        color: ruined ? undefined : this.segment.color,
        broken,
      });
    }
  }

  towerItem(anchor: Point2D): WallDrawItem {
    const [a, b] = this.geometry.outer;
    const r = this.m.hex_r * TOWER_SCALE;
    const total = this.m.parapet_depth + this.m.deck_depth;
    const mid = along(a, b, 0.5, { x: this.geometry.inward.x * total, y: this.geometry.inward.y * total });
    const half = (TOWER_REACH * r) / 2;
    const center = { x: mid.x + this.inward_normal.x * half, y: mid.y + this.inward_normal.y * half };
    const facing = this.inward_normal;
    const big = this.breaches.find((b) => b.spec.big);
    const heading = big && big.spec.range[0] + big.spec.range[1] > 1 ? 1 : -1;
    const dir = { x: this.edge_unit.x * heading, y: this.edge_unit.y * heading };
    const collapse = big ? { dir, atlas: damageAtlas(this.tex.rubble, this.segment.damage) } : undefined;
    const wide = collapse ? 0.1 * r : 0;
    return {
      y: anchor.y + center.y,
      anchor: { x: anchor.x + center.x, y: anchor.y + center.y },
      alpha: this.segment.alpha,
      bounds: { x: -0.14 * r - wide, y: -0.7 * r, w: 0.28 * r + 2 * wide, h: 0.82 * r },
      key: `T|${r}|${shapeKey(facing)}|${this.segment.color}|${this.segment.garrison_icon ? 'g' : '-'}|${this.segment.damage}|${collapse ? shapeKey(dir) : '-'}`,
      draw: (ctx): void =>
        drawTower(
          ctx,
          this.tex.tower,
          r,
          { x: 0, y: 0 },
          facing,
          this.segment.color,
          this.segment.garrison_icon,
          this.segment.damage,
          collapse
        ),
    };
  }

  wallItem(anchor: Point2D): WallDrawItem {
    const { outer, inner } = this.geometry;
    const r = this.m.hex_r;
    const xs = [...outer, ...inner].map((p) => p.x);
    const ys = [...outer, ...inner].map((p) => p.y);
    const rise = this.m.wall_h + this.m.merlon_h + 2.8 * this.m.unit;
    const left = Math.min(...xs) - 0.06 * r;
    const top = Math.min(...ys) - rise;
    const nb = (n?: Point2D): string => (n ? shapeKey(n) : '-');
    const pad = this.segment.damage > 0 ? 0.08 * r : 0;
    const bounds = {
      x: left - pad,
      y: top,
      w: Math.max(...xs) + 0.16 * r - left + 2 * pad,
      h: Math.max(...ys) + 0.04 * r - top + pad,
    };
    return {
      y: anchor.y + (outer[0].y + outer[1].y) / 2,
      anchor,
      alpha: this.segment.alpha,
      bounds,
      key: `W|${r}|${shapeKey(this.edge)}|${shapeKey(this.n)}|${nb(this.segment.start_neighbor_normal)}|${nb(this.segment.end_neighbor_normal)}|${this.segment.color}|${this.segment.damage}|${this.segment.face}`,
      draw: (ctx): void => {
        this.ctx = ctx;
        this.draw(this.breachDrawings());
      },
    };
  }

  rubbleItems(anchor: Point2D): WallDrawItem[] {
    const [a, b] = this.geometry.outer;
    const key = `R|${this.m.hex_r}|${shapeKey(this.edge)}|${shapeKey(this.n)}|${this.segment.face}|${this.segment.damage}`;
    return rubbleItems(this.breachDrawings(), anchor, (a.y + b.y) / 2, key, this.segment.alpha);
  }

  private breachDrawings(): BreachDrawing[] {
    return this.breaches.map((b) => ({
      ctx: this.ctx,
      stone: this.tex.stone,
      atlas: damageAtlas(this.tex.rubble, this.segment.damage),
      outer: this.geometry.outer,
      inner: this.geometry.inner,
      wall_h: this.m.wall_h,
      top_z: this.m.wall_h + this.m.crenel_h,
      edge_unit: this.edge_unit,
      floor: b.floor,
      stones: b.stones,
      floor_rocks: floorRockCount(this.segment.damage),
      caps: !isRuin(this.segment.damage),
      seed: b.seed,
    }));
  }

  private drawBacking(top = this.m.wall_h): void {
    const { outer, inner } = this.geometry;
    const points = [...this.partOf(outer), ...this.partOf(inner)].flatMap((p) => [p, lift(p, top)]);
    pathPolygon(this.ctx, convexHull(points));
    this.ctx.fillStyle = BACKING;
    this.ctx.fill();
  }

  private drawFace(line: [Point2D, Point2D], normal: Point2D, top?: [number, number]): void {
    const [a, b] = line;
    const h = this.m.wall_h;
    const [p, q] = this.partOf(line);
    const source = faceRect(this.tex.stone, 1 - HEIGHT_SCALE, 1);
    drawTiledQuad(
      this.ctx,
      this.tex.stone,
      source,
      lift(a, h),
      { x: b.x - a.x, y: b.y - a.y },
      { x: 0, y: h },
      (h * source.w) / source.h,
      [p, q, lift(q, top?.[1] ?? h), lift(p, top?.[0] ?? h)],
      lightFor(normal) * (isWeathered(this.segment.damage) ? WEATHERED_LIGHT : 1)
    );
    const damage = this.segment.damage;
    if (damage > 0 && !top) {
      const edges = this.breaches.flatMap((b) => b.spec.range);
      if (!isWeathered(damage)) {
        drawScorch(this.ctx, line, h, damage, seededRandom(`${this.seed}|scorch`), this.span, edges);
      }
      drawCracks(this.ctx, line, h, damage, seededRandom(`${this.seed}|cracks`), this.span);
    }
  }

  private drawParapetBand(line: [Point2D, Point2D], normal: Point2D): void {
    const [a, b] = line;
    const [p, q] = this.partOf(line);
    const { wall_h: h, crenel_h: c } = this.m;
    const full = this.tex.stone;
    const source: SourceRect = {
      x: 0,
      y: 0.3 * full.naturalHeight,
      w: full.naturalWidth,
      h: HEIGHT_SCALE * (c / h) * full.naturalHeight,
    };
    drawTiledQuad(
      this.ctx,
      full,
      source,
      lift(a, h + c),
      { x: b.x - a.x, y: b.y - a.y },
      { x: 0, y: c },
      (c * source.w) / source.h,
      [lift(p, h), lift(q, h), lift(q, h + c), lift(p, h + c)],
      lightFor(normal)
    );
  }

  private drawDeck(parapet: [Point2D, Point2D], inner: [Point2D, Point2D]): void {
    const lifted = [parapet[0], parapet[1], inner[1], inner[0]].map((p) => lift(p, this.m.wall_h));
    const [pa, pb] = this.partOf(parapet);
    const [ia, ib] = this.partOf(inner);
    const clip = [pa, pb, ib, ia].map((p) => lift(p, this.m.wall_h));
    const deck = this.tex.deck;
    const source: SourceRect = { x: 0, y: 0, w: deck.naturalWidth, h: deck.naturalHeight };
    const across = { x: this.geometry.inward.x * this.m.deck_depth, y: this.geometry.inward.y * this.m.deck_depth };
    const margin = 0.1 * this.m.hex_r;
    const reach = this.m.hex_r + 2 * margin;
    drawTiledQuad(
      this.ctx,
      deck,
      source,
      { x: lifted[0].x - this.edge_unit.x * margin, y: lifted[0].y - this.edge_unit.y * margin },
      { x: this.edge_unit.x * reach, y: this.edge_unit.y * reach },
      across,
      (this.m.deck_depth * source.w) / source.h,
      clip,
      1
    );
  }

  private drawParapetTop(): void {
    const { outer, parapet } = this.geometry;
    const z = this.m.wall_h + this.m.crenel_h;
    const [o0, o1] = this.partOf(outer);
    const [p0, p1] = this.partOf(parapet);
    pathPolygon(
      this.ctx,
      [o0, o1, p1, p0].map((p) => lift(p, z))
    );
    this.ctx.fillStyle = TOP_COLOR;
    this.ctx.fill();
    this.ctx.strokeStyle = TOP_EDGE;
    this.ctx.lineWidth = 1 / 2;
    this.ctx.stroke();
  }

  private merlonQuad(i: number): { o0: Point2D; o1: Point2D; q0: Point2D; q1: Point2D } {
    const [a, b] = this.geometry.outer;
    const half = this.m.merlon_w / this.m.hex_r / 2;
    const center = (i + 0.5) / WALL_MERLONS;
    const shift = { x: this.geometry.inward.x * this.m.merlon_depth, y: this.geometry.inward.y * this.m.merlon_depth };
    const o0 = along(a, b, center - half);
    const o1 = along(a, b, center + half);
    return {
      o0,
      o1,
      q0: { x: o0.x + shift.x, y: o0.y + shift.y },
      q1: { x: o1.x + shift.x, y: o1.y + shift.y },
    };
  }

  private drawMerlons(): void {
    const half = this.m.merlon_w / this.m.hex_r / 2;
    const order = Array.from({ length: WALL_MERLONS }, (_, i) => i)
      .filter((i) => this.spans((i + 0.5) / WALL_MERLONS, half))
      .sort((i, j) => this.merlonQuad(i).q0.y - this.merlonQuad(j).q0.y);
    for (const i of order) {
      this.drawMerlon(i);
    }
  }

  private drawMerlon(i: number): void {
    const { o0, o1, q0, q1 } = this.merlonQuad(i);
    const scale = this.merlon_scale[i];
    if (scale === 0) {
      return;
    }
    const { wall_h: h, crenel_h: c } = this.m;
    const mh = c + (this.m.merlon_h - c) * scale;
    const stone = this.tex.stone;
    const rows = HEIGHT_SCALE * ((mh - c) / h) * stone.naturalHeight;
    const cols = HEIGHT_SCALE * (this.m.merlon_w / h) * stone.naturalHeight;
    const source: SourceRect = {
      x: (i * 37) % Math.max(1, stone.naturalWidth - cols),
      y: 0.3 * stone.naturalHeight,
      w: cols,
      h: rows,
    };
    const backward = { x: -this.edge_unit.x, y: -this.edge_unit.y };
    const faces: [Point2D, Point2D, Point2D, boolean][] = [
      [o0, o1, this.n, this.front],
      [q0, q1, this.inward_normal, this.back],
      [o0, q0, backward, backward.y > 0.02],
      [o1, q1, this.edge_unit, this.edge_unit.y > 0.02],
    ];
    const z0 = h + c;
    const z1 = h + mh;
    for (const [p, q, normal, visible] of faces) {
      if (!visible) {
        continue;
      }
      drawTexturedQuad(
        this.ctx,
        stone,
        source,
        lift(p, z1),
        { x: q.x - p.x, y: q.y - p.y },
        { x: 0, y: z1 - z0 },
        [lift(p, z0), lift(q, z0), lift(q, z1), lift(p, z1)],
        lightFor(normal)
      );
    }
    pathPolygon(
      this.ctx,
      [o0, o1, q1, q0].map((p) => lift(p, z1))
    );
    this.ctx.fillStyle = TOP_COLOR;
    this.ctx.fill();
    this.ctx.strokeStyle = TOP_EDGE;
    this.ctx.stroke();
  }
}

function shapeKey(v: Point2D): string {
  return `${Math.round(v.x * 100) / 100},${Math.round(v.y * 100) / 100}`;
}

export function wallDrawItems(textures: WallTextures, metrics: WallMetrics, segment: WallSegmentInput): WallDrawItem[] {
  const anchor = segment.a;
  const relative = { ...segment, a: { x: 0, y: 0 }, b: { x: segment.b.x - anchor.x, y: segment.b.y - anchor.y } };
  const painter = new WallPainter(textures, metrics, relative);
  return [painter.wallItem(anchor), painter.towerItem(anchor), ...painter.rubbleItems(anchor)];
}
