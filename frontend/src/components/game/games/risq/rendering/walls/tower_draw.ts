import type { Point2D } from '../../../../util/objects2d';
import { drawHangingBanner } from './wall_cloth';
import type { SourceRect } from './textured_quad';
import { drawTexturedQuad, lightFor, pathPolygon } from './textured_quad';
import { drawTexturedTriangle, drawTexturedTrapezoid } from './textured_triangle';
import { HEIGHT_SCALE } from './wall_metrics';
import type { WallDamage } from './wall_damage';
import { drawCharredOpening, drawRubbleStone, drawSmoke, isRuin, isWeathered, seededRandom } from './wall_damage';
import {
  RUIN_COUNTS,
  WEATHERED_COUNTS,
  WEATHERED_HEIGHT,
  WEATHERED_LIGHT,
  drawRafters,
  drawRuinProps,
} from './tower_ruin';

export declare interface TowerTextures {
  stone: HTMLImageElement;
  frame_front: HTMLImageElement;
  frame_side: HTMLImageElement;
  pal: HTMLImageElement;
  top: HTMLImageElement;
  thatch: HTMLImageElement;
}

const UNIT = 0.36 / 1395;
const HEIGHT_UNIT = UNIT * HEIGHT_SCALE;
const STONE_H = 295 * HEIGHT_UNIT;
const FRAME_H = 420 * HEIGHT_UNIT;
const PAL_H = 190 * HEIGHT_UNIT;
const POST_H = 90 * HEIGHT_UNIT;
const ROOF_H = 270 * HEIGHT_UNIT;
const STONE_SIDE = 0.152;
const BODY_SIDE = 0.115;
const ROOF_HALF = 0.082;
const TAPER = 0.86;
const SILL_H = 0.011 * HEIGHT_SCALE;
const DECK_H = 0.016 * HEIGHT_SCALE;
const SILL_RATIO = 1.04;
const OVERHANG_RATIO = 1.17;
const ROOF_RINGS = 12;
const BURNT_ROOF_RINGS = 5;
const COLLAPSED_ROOF_RINGS = 3;
const RUIN_BASE_HEIGHTS = [0.7, 0.35, 0.55, 0.25];
const CAP_CORNERS: [number, number][] = [
  [-1, -1],
  [1, -1],
  [1, 1],
  [-1, 1],
];
const SMOKE_SIZE = 0.045;
const BURNT_SMOKE_SCALE = 1.4;
const RAFTER_END = 0.95;
const RAFTER_END_BROKEN = 0.7;
const RAFTER_TIES = [0.72, 0.86];
const HOLE_T = 0.4;
const HOLE_CORNERS: [number, number][] = [
  [-0.55, -0.45],
  [0.95, -0.7],
  [0.8, 0.95],
  [-0.6, 0.7],
];
const ROOF_CURVE = 0.75;
const ROOF_UV_X = 0.94;
const ROOF_UV_Y = 0.96;
const ROOF_DARKEN = 0.84;
const GARRISON_FLAG_SIZE = 0.12;
const FLAG_POLE_X = 0.194;
const FLAG_POLE_BOTTOM = 0.97;
const WOOD = 'rgb(82, 62, 42)';
const FLOOR = 'rgb(96, 74, 52)';
const POST = 'rgb(62, 48, 36)';

export const TOWER_REACH = STONE_SIDE;

export declare interface TowerCollapse {
  dir: Point2D;
  atlas: CanvasImageSource;
}

declare interface Side {
  normal: Point2D;
  a: [number, number];
  b: [number, number];
  flip: boolean;
  front_texture: boolean;
}

function rect(image: HTMLImageElement): SourceRect {
  return { x: 0, y: 0, w: image.naturalWidth, h: image.naturalHeight };
}

class TowerPainter {
  private readonly u: Point2D;
  private readonly sides: Side[];

  constructor(
    private ctx: CanvasRenderingContext2D,
    private tex: TowerTextures,
    private r: number,
    private center: Point2D,
    private f: Point2D,
    private color: string,
    private garrison_icon?: CanvasImageSource,
    private damage: WallDamage = 0,
    private collapse?: TowerCollapse
  ) {
    this.u = { x: -f.y, y: f.x };
    this.sides = [
      { normal: f, a: [-1, 1], b: [1, 1], flip: false, front_texture: true },
      { normal: { x: -f.x, y: -f.y }, a: [-1, -1], b: [1, -1], flip: false, front_texture: false },
      { normal: this.u, a: [1, -1], b: [1, 1], flip: false, front_texture: false },
      { normal: { x: -this.u.x, y: -this.u.y }, a: [-1, -1], b: [-1, 1], flip: true, front_texture: false },
    ];
  }

  private corner(a: number, b: number, half: number): Point2D {
    return {
      x: this.center.x + (this.u.x * a + this.f.x * b) * half,
      y: this.center.y + (this.u.y * a + this.f.y * b) * half,
    };
  }

  private lift(p: Point2D, z: number): Point2D {
    return { x: p.x, y: p.y - z };
  }

  private collapsing(side: Side): boolean {
    return !!this.collapse && side.normal.x * this.collapse.dir.x + side.normal.y * this.collapse.dir.y > 0.5;
  }

  private collapseSide(): number {
    return this.collapse ? Math.sign(this.collapse.dir.x * this.u.x + this.collapse.dir.y * this.u.y) || 1 : 1;
  }

  draw(): void {
    const r = this.r;
    const z1 = STONE_H * r;
    const zs = z1 + SILL_H * r;
    const z2 = z1 + FRAME_H * r;
    const zd = z2 + DECK_H * r;
    const z3 = z2 + PAL_H * r;
    const z4 = z3 + POST_H * r;
    const visible = this.sides.filter((s) => s.normal.y > 0.02);
    const hidden = this.sides.filter((s) => s.normal.y < -0.02);
    if (isRuin(this.damage)) {
      this.drawRuin(visible, z1);
      return;
    }
    this.drawBase(visible, z1);
    if (this.damage === 2) {
      this.drawBaseDamage(visible, z1);
    }
    this.drawBand(visible, SILL_RATIO, z1, zs, WOOD, 1.05);
    this.drawFrame(hidden, 0.8, zs, z2);
    if (this.damage !== 2) {
      this.drawFloor(z2);
    }
    this.drawFrame(visible, 1.12, zs, z2);
    this.drawBand(visible, OVERHANG_RATIO, z2, zd, WOOD, 0.85);
    this.drawBanners(visible, z2);
    this.drawPalisade(hidden, 0.6, zd, z3);
    this.drawPalisade(visible, 1.1, zd, z3);
    this.drawPosts(z3, z4);
    this.drawRoof(z4);
    this.drawRoofDamage(z4);
    this.drawGarrisonFlag(z4);
    this.drawCollapseRubble();
  }

  private drawRuin(visible: Side[], z1: number): void {
    this.drawBase(visible, z1);
    if (!isWeathered(this.damage)) {
      this.drawBaseDamage(visible, z1);
    }
    if (!this.collapse) {
      return;
    }
    const counts = isWeathered(this.damage) ? WEATHERED_COUNTS : RUIN_COUNTS;
    const random = seededRandom('tower-ruin');
    const half = (STONE_SIDE * this.r) / 2;
    const surface = (): Point2D => {
      const [a, b] = [-0.8 + 1.6 * random(), -0.8 + 1.6 * random()];
      return this.lift(this.corner(a, b, half), z1 * 0.45);
    };
    const ground = (): Point2D => {
      const sign = random() < 0.5 ? -1 : 1;
      return this.corner(sign * (1.0 + 1.0 * random()), -1.3 + 2.6 * random(), half);
    };
    const points = {
      beams: Array.from({ length: counts.beams }, surface),
      surface: Array.from({ length: counts.surface }, surface),
      ground: Array.from({ length: counts.ground }, ground),
      smoke: Array.from({ length: counts.smoke }, surface),
    };
    drawRuinProps(this.ctx, this.collapse.atlas, this.r, SMOKE_SIZE * this.r, points);
  }

  private drawBaseDamage(visible: Side[], z1: number): void {
    const half = (STONE_SIDE * this.r) / 2;
    const random = seededRandom('tower-base');
    for (const s of visible) {
      const [b0, b1] = [this.corner(...s.a, half), this.corner(...s.b, half)];
      const [t0, t1] = [this.topPoint(s.a, z1, TAPER), this.topPoint(s.b, z1, TAPER)];
      const at = (u: number, v: number): Point2D => {
        const low = { x: b0.x + (b1.x - b0.x) * u, y: b0.y + (b1.y - b0.y) * u };
        const high = { x: t0.x + (t1.x - t0.x) * u, y: t0.y + (t1.y - t0.y) * u };
        return { x: low.x + (high.x - low.x) * v, y: low.y + (high.y - low.y) * v };
      };
      this.ctx.save();
      pathPolygon(this.ctx, [b0, b1, t1, t0]);
      this.ctx.clip();
      const soot = this.ctx.createLinearGradient(0, t0.y, 0, b0.y);
      soot.addColorStop(0, 'rgba(18, 12, 8, 0.65)');
      soot.addColorStop(0.6, 'rgba(18, 12, 8, 0.2)');
      soot.addColorStop(1, 'rgba(18, 12, 8, 0)');
      this.ctx.fillStyle = soot;
      this.ctx.fillRect(this.center.x - this.r, this.center.y - 1.5 * this.r, 2 * this.r, 3 * this.r);
      this.ctx.strokeStyle = 'rgba(24, 18, 12, 0.9)';
      this.ctx.lineWidth = Math.max(1, 0.006 * this.r);
      for (let k = 0; k < (this.collapsing(s) ? 5 : 2); k++) {
        let u = 0.1 + 0.8 * random();
        let v = 1;
        this.ctx.beginPath();
        this.ctx.moveTo(at(u, v).x, at(u, v).y);
        for (let step = 0; step < 4; step++) {
          u += (random() - 0.5) * 0.18;
          v -= 0.15 + 0.2 * random();
          this.ctx.lineTo(at(u, v).x, at(u, v).y);
        }
        this.ctx.stroke();
      }
      this.ctx.restore();
    }
  }

  private drawCollapseRubble(): void {
    if (!this.collapse) {
      return;
    }
    const side = Math.sign(this.collapse.dir.x * this.u.x + this.collapse.dir.y * this.u.y) || 1;
    const half = (STONE_SIDE * this.r) / 2;
    const random = seededRandom('tower-rubble');
    for (let i = 0; i < 3; i++) {
      const stone = { u: 0, across: 0, cell: Math.floor(random() * 8), size: 0.045 * this.r, mirrored: random() < 0.5 };
      drawRubbleStone(
        this.ctx,
        this.collapse.atlas,
        this.corner(side * (1.25 + 0.5 * random()), -0.9 + 1.9 * random(), half),
        stone
      );
    }
  }

  private drawRoofDamage(z4: number): void {
    if (this.damage === 0) {
      return;
    }
    const t = this.damage === 2 ? BURNT_ROOF_RINGS / ROOF_RINGS : HOLE_T;
    const corners = this.damage === 2 ? CAP_CORNERS : HOLE_CORNERS;
    drawCharredOpening(
      this.ctx,
      corners.map(([a, b]) => this.roofPoint(a, b, t, z4)),
      seededRandom('roof-hole'),
      this.damage === 2
    );
    if (this.damage === 2) {
      this.drawRafters(z4, t);
    }
    const plume = SMOKE_SIZE * this.r * (this.damage === 2 ? BURNT_SMOKE_SCALE : 1);
    drawSmoke(this.ctx, this.roofPoint(0.15, 0.15, t, z4), plume, seededRandom('roof-smoke'));
    if (this.damage === 2) {
      drawSmoke(this.ctx, this.roofPoint(-0.7, 0.5, t, z4), plume * 0.8, seededRandom('roof-smoke-2'));
      drawSmoke(this.ctx, this.roofPoint(0.6, -0.6, t, z4), plume * 0.65, seededRandom('roof-smoke-3'));
    }
  }

  private drawRafters(z4: number, t: number): void {
    const apex = this.roofPoint(0, 0, this.collapse ? RAFTER_END_BROKEN : RAFTER_END, z4);
    const standing = CAP_CORNERS.filter(([a]) => !this.collapse || a !== this.collapseSide());
    const rings = RAFTER_TIES.map((level) => standing.map(([a, b]) => this.roofPoint(a, b, level, z4)));
    drawRafters(
      this.ctx,
      this.r,
      apex,
      standing.map(([a, b]) => this.roofPoint(a, b, t, z4)),
      rings
    );
  }

  private drawGarrisonFlag(z4: number): void {
    if (!this.garrison_icon || this.damage === 2) {
      return;
    }
    const size = GARRISON_FLAG_SIZE * this.r;
    const apex = { x: this.center.x, y: this.center.y - z4 - ROOF_H * this.r };
    this.ctx.drawImage(this.garrison_icon, apex.x - FLAG_POLE_X * size, apex.y - FLAG_POLE_BOTTOM * size, size, size);
  }

  private cornerTop(a: number, b: number, z1: number): number {
    const worn = isWeathered(this.damage) ? WEATHERED_HEIGHT : 1;
    return isRuin(this.damage) ? z1 * worn * RUIN_BASE_HEIGHTS[(a > 0 ? 1 : 0) + (b > 0 ? 2 : 0)] : z1;
  }

  private topPoint(corner: [number, number], z1: number, scale: number): Point2D {
    const half = (STONE_SIDE * this.r) / 2;
    return this.lift(this.corner(...corner, half * scale), this.cornerTop(...corner, z1));
  }

  private edgeJag(p: [number, number], q: [number, number], z1: number): Point2D[] {
    const forward = `${p}` < `${q}`;
    const [from, to] = forward ? [p, q] : [q, p];
    const random = seededRandom(`rim|${from}|${to}`);
    const start = this.topPoint(from, z1, TAPER);
    const end = this.topPoint(to, z1, TAPER);
    const points = [1, 2, 3, 4].map((k) => ({
      x: start.x + ((end.x - start.x) * k) / 5,
      y: start.y + ((end.y - start.y) * k) / 5 + z1 * 0.45 * random(),
    }));
    return forward ? points : points.reverse();
  }

  private drawBase(visible: Side[], z1: number): void {
    const half = (STONE_SIDE * this.r) / 2;
    const ruined = isRuin(this.damage);
    for (const s of [...visible].sort((p, q) => p.normal.y - q.normal.y)) {
      this.ctx.save();
      if (ruined) {
        const rim = [this.topPoint(s.b, z1, TAPER), ...this.edgeJag(s.b, s.a, z1), this.topPoint(s.a, z1, TAPER)];
        pathPolygon(this.ctx, [this.corner(...s.a, half), this.corner(...s.b, half), ...rim]);
        this.ctx.clip();
      }
      drawTexturedTrapezoid(
        this.ctx,
        this.tex.stone,
        [this.corner(...s.a, half), this.corner(...s.b, half)],
        [this.lift(this.corner(...s.a, half * TAPER), z1), this.lift(this.corner(...s.b, half * TAPER), z1)],
        lightFor(s.normal) * (isWeathered(this.damage) ? WEATHERED_LIGHT : 1 - 0.2 * Math.min(this.damage, 2))
      );
      this.ctx.restore();
    }
    const top = CAP_CORNERS.map((c) => this.topPoint(c, z1, TAPER));
    const outline = ruined
      ? CAP_CORNERS.flatMap((c, i) => [this.topPoint(c, z1, TAPER), ...this.edgeJag(c, CAP_CORNERS[(i + 1) % 4], z1)])
      : top;
    const image = ruined ? this.tex.stone : this.tex.top;
    const source = rect(image);
    drawTexturedQuad(
      this.ctx,
      image,
      ruined ? { ...source, y: 0.4 * source.h, h: 0.6 * source.h } : source,
      top[0],
      { x: top[1].x - top[0].x, y: top[1].y - top[0].y },
      { x: top[3].x - top[0].x, y: top[3].y - top[0].y },
      outline,
      1
    );
  }

  private drawBand(visible: Side[], ratio: number, z0: number, z1: number, color: string, light: number): void {
    const half = ((BODY_SIDE * this.r) / 2) * ratio;
    for (const s of visible) {
      const a = this.corner(...s.a, half);
      const b = this.corner(...s.b, half);
      pathPolygon(this.ctx, [this.lift(a, z0), this.lift(b, z0), this.lift(b, z1), this.lift(a, z1)]);
      this.ctx.fillStyle = color;
      this.ctx.fill();
      if (lightFor(s.normal) * light < 1) {
        this.ctx.fillStyle = `rgba(0, 0, 0, ${1 - lightFor(s.normal) * light})`;
        this.ctx.fill();
      }
    }
  }

  private drawQuad(s: Side, image: HTMLImageElement, half: number, z0: number, z1: number, light: number): void {
    const a_index = s.flip ? s.b : s.a;
    const b_index = s.flip ? s.a : s.b;
    const a = this.corner(...a_index, half);
    const b = this.corner(...b_index, half);
    const drop = (index: [number, number]): number =>
      this.collapse && index[0] === this.collapseSide() ? 0.5 * (z1 - z0) : 0;
    const random = seededRandom(`tower-break|${z0}`);
    const top = [1, 2].map((k) => {
      const at = { x: b.x + (a.x - b.x) * (k / 3), y: b.y + (a.y - b.y) * (k / 3) };
      const sloped = drop(b_index) + (drop(a_index) - drop(b_index)) * (k / 3);
      return this.lift(at, z1 - sloped - (drop(a_index) + drop(b_index) > 0 ? 0.15 * (z1 - z0) * random() : 0));
    });
    drawTexturedQuad(
      this.ctx,
      image,
      rect(image),
      this.lift(a, z1),
      { x: b.x - a.x, y: b.y - a.y },
      { x: 0, y: z1 - z0 },
      [this.lift(a, z0), this.lift(b, z0), this.lift(b, z1 - drop(b_index)), ...top, this.lift(a, z1 - drop(a_index))],
      light
    );
  }

  private drawFrame(sides: Side[], mult: number, z0: number, z1: number): void {
    const half = (BODY_SIDE * this.r) / 2;
    for (const s of sides) {
      const image = s.front_texture ? this.tex.frame_front : this.tex.frame_side;
      this.drawQuad(s, image, half, z0, z1, lightFor(s.normal) * mult);
    }
  }

  private drawFloor(z2: number): void {
    const half = (BODY_SIDE * this.r) / 2 + 2;
    pathPolygon(
      this.ctx,
      [this.corner(-1, -1, half), this.corner(1, -1, half), this.corner(1, 1, half), this.corner(-1, 1, half)].map(
        (p) => this.lift(p, z2)
      )
    );
    this.ctx.fillStyle = FLOOR;
    this.ctx.fill();
  }

  private drawPalisade(sides: Side[], mult: number, z0: number, z1: number): void {
    const half = ((BODY_SIDE * this.r) / 2) * OVERHANG_RATIO;
    for (const s of sides) {
      this.drawQuad(s, this.tex.pal, half, z0, z1, lightFor(s.normal) * mult);
    }
  }

  private drawBanners(visible: Side[], z2: number): void {
    const half = ((BODY_SIDE * this.r) / 2) * SILL_RATIO;
    const drop = FRAME_H * this.r * 0.85;
    for (const s of visible) {
      const a = this.corner(...s.a, half);
      const b = this.corner(...s.b, half);
      const left = { x: a.x + (b.x - a.x) * 0.33, y: a.y + (b.y - a.y) * 0.33 };
      const right = { x: a.x + (b.x - a.x) * 0.67, y: a.y + (b.y - a.y) * 0.67 };
      drawHangingBanner(
        this.ctx,
        this.color,
        this.lift(left, z2 - 0.01 * this.r),
        { x: right.x - left.x, y: right.y - left.y },
        drop,
        lightFor(s.normal)
      );
    }
  }

  private drawPosts(z3: number, z4: number): void {
    const half = (BODY_SIDE * this.r) / 2;
    this.ctx.strokeStyle = POST;
    this.ctx.lineWidth = Math.max(2, 0.012 * this.r);
    for (const [a, b] of [
      [-1, -1],
      [1, -1],
      [1, 1],
      [-1, 1],
    ]) {
      const c = this.corner(a, b, half);
      const snapped = this.collapse && a === this.collapseSide();
      this.ctx.beginPath();
      this.ctx.moveTo(c.x, c.y - z3);
      this.ctx.lineTo(c.x, c.y - (snapped ? z3 + 0.35 * (z4 - z3) : z4));
      this.ctx.stroke();
    }
  }

  private roofPoint(a: number, b: number, t: number, z4: number): Point2D {
    const profile = (1 - ROOF_CURVE) * t + ROOF_CURVE * (1 - (1 - t) ** 2);
    return this.lift(this.corner(a, b, ROOF_HALF * this.r * (1 - t)), z4 + ROOF_H * this.r * profile);
  }

  private drawRoof(z4: number): void {
    const image = this.tex.thatch;
    const w = image.naturalWidth;
    const h = image.naturalHeight;
    const faces = [...this.sides].sort((p, q) => p.normal.y - q.normal.y);
    const rings = this.damage === 2 ? BURNT_ROOF_RINGS : ROOF_RINGS;
    for (const s of faces) {
      for (let j = 0; j < (this.collapsing(s) ? COLLAPSED_ROOF_RINGS : rings); j++) {
        const t0 = j / ROOF_RINGS;
        const t1 = (j + 1) / ROOF_RINGS;
        const A0 = this.roofPoint(...s.a, t0, z4);
        const A1 = this.roofPoint(...s.b, t0, z4);
        const B0 = this.roofPoint(...s.a, t1, z4);
        const B1 = this.roofPoint(...s.b, t1, z4);
        const uv = (t: number, side: -1 | 1): Point2D => ({
          x: w / 2 + side * (w / 2) * (1 - t) * ROOF_UV_X,
          y: (1 - t) * h * ROOF_UV_Y,
        });
        const light = (0.95 + 0.2 * s.normal.y + 0.08 * t0) * ROOF_DARKEN * (1 - 0.22 * this.damage);
        drawTexturedTriangle(this.ctx, image, [uv(t0, -1), uv(t0, 1), uv(t1, 1)], [A0, A1, B1], light);
        drawTexturedTriangle(this.ctx, image, [uv(t0, -1), uv(t1, 1), uv(t1, -1)], [A0, B1, B0], light);
      }
    }
  }
}

export function drawTower(
  ctx: CanvasRenderingContext2D,
  textures: TowerTextures,
  hex_r: number,
  center: Point2D,
  facing: Point2D,
  color: string,
  garrison_icon?: CanvasImageSource,
  damage: WallDamage = 0,
  collapse?: TowerCollapse
): void {
  new TowerPainter(ctx, textures, hex_r, center, facing, color, garrison_icon, damage, collapse).draw();
}
