import type { BoardTransformData } from '../../game/util/canvas_board/canvas_board';
import { configDraw } from '../../game/util/canvas_components/canvas_component';
import { drawHexagon, drawRect, drawText } from '../../game/util/canvas_util';
import type { Point2D } from '../../game/util/objects2d';
import { terrainImage } from '../../game/games/risq/rendering/terrain';
import { drawHexImage } from '../../game/games/risq/rendering/space';
import type { EditorConfigPanel, EditorToolContext, PanelBounds } from '../editor_tool';
import { DEFAULT_TERRAIN_ID } from '../map_doc';

const PADDING = 6;
const GAP = 4;
const LABEL_H = 18;
const ROW_H = 24;
const TILE = 40;
const HEX_R = 18;
const WHEEL_STEP = TILE + GAP;
const INK = 'rgb(59, 36, 19)';
const FONT = '12px "Fira Sans", sans-serif';
const BRUSHES = [
  { label: '1', radius: 0 },
  { label: '7', radius: 1 },
  { label: '19', radius: 2 },
];

type PaletteTarget = { kind: 'brush'; index: number } | { kind: 'tile'; index: number };

function contains(rect: PanelBounds, m: Point2D): boolean {
  return m.x >= rect.x && m.x <= rect.x + rect.w && m.y >= rect.y && m.y <= rect.y + rect.h;
}

export class TerrainPalette implements EditorConfigPanel {
  private bounds: PanelBounds = { x: 0, y: 0, w: 0, h: 0 };
  private hover?: PaletteTarget;
  private pressed?: PaletteTarget;
  private selected_id = DEFAULT_TERRAIN_ID;
  private radius = 0;
  private scroll_offset = 0;
  private mouse: Point2D = { x: 0, y: 0 };

  constructor(private context: EditorToolContext) {}

  terrainId(): number {
    return this.selected_id;
  }

  brushRadius(): number {
    return this.radius;
  }

  setBounds(bounds: PanelBounds): void {
    this.bounds = bounds;
    this.scroll_offset = Math.min(this.scroll_offset, this.maxScroll());
  }

  private brushRect(index: number): PanelBounds {
    const w = (this.bounds.w - 2 * PADDING - (BRUSHES.length - 1) * GAP) / BRUSHES.length;
    return { x: this.bounds.x + PADDING + index * (w + GAP), y: this.bounds.y + PADDING + LABEL_H, w, h: ROW_H };
  }

  private gridRect(): PanelBounds {
    const y = this.bounds.y + 2 * PADDING + 2 * LABEL_H + ROW_H;
    return {
      x: this.bounds.x + PADDING,
      y,
      w: this.bounds.w - 2 * PADDING,
      h: this.bounds.y + this.bounds.h - PADDING - y,
    };
  }

  private columns(): number {
    return Math.max(1, Math.floor((this.gridRect().w + GAP) / (TILE + GAP)));
  }

  private contentH(): number {
    return Math.ceil(this.context.terrains().length / this.columns()) * (TILE + GAP);
  }

  private maxScroll(): number {
    return Math.max(0, this.contentH() - this.gridRect().h);
  }

  private tileRect(index: number): PanelBounds {
    const grid = this.gridRect();
    const column = index % this.columns();
    const row = Math.floor(index / this.columns());
    return { x: grid.x + column * (TILE + GAP), y: grid.y + row * (TILE + GAP) - this.scroll_offset, w: TILE, h: TILE };
  }

  private targetAt(m: Point2D): PaletteTarget | undefined {
    const brush = BRUSHES.findIndex((_, i) => contains(this.brushRect(i), m));
    if (brush >= 0) {
      return { kind: 'brush', index: brush };
    }
    if (!contains(this.gridRect(), m)) {
      return undefined;
    }
    const tile = this.context.terrains().findIndex((_, i) => contains(this.tileRect(i), m));
    return tile >= 0 ? { kind: 'tile', index: tile } : undefined;
  }

  private sameTarget(a?: PaletteTarget, b?: PaletteTarget): boolean {
    return !!a && !!b && a.kind === b.kind && a.index === b.index;
  }

  isHovering(): boolean {
    return this.hover !== undefined;
  }

  setHovering(hovering: boolean): void {
    if (!hovering) {
      this.hover = undefined;
    }
  }

  isClicking(): boolean {
    return this.pressed !== undefined;
  }

  setClicking(clicking: boolean): void {
    if (!clicking) {
      this.pressed = undefined;
    }
  }

  mousemove(_canvas: Point2D, screen: Point2D, _transform: BoardTransformData): boolean {
    this.mouse = screen;
    this.hover = this.targetAt(screen);
    return this.hover !== undefined;
  }

  scroll(dy: number): boolean {
    if (!contains(this.gridRect(), this.mouse)) {
      return false;
    }
    this.scroll_offset = Math.min(Math.max(this.scroll_offset + Math.sign(dy) * WHEEL_STEP, 0), this.maxScroll());
    return true;
  }

  mousedown(e: MouseEvent): boolean {
    if (e.button !== 0 || !this.hover) {
      return false;
    }
    this.pressed = this.hover;
    return true;
  }

  mouseup(e: MouseEvent): void {
    const target = this.pressed;
    this.pressed = undefined;
    if (e.button !== 0 || !target || !this.sameTarget(target, this.hover)) {
      return;
    }
    if (target.kind === 'brush') {
      this.radius = BRUSHES[target.index].radius;
    } else {
      this.selected_id = this.context.terrains()[target.index].terrain_id;
    }
  }

  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, _dt: number): void {
    const draw_config = { fill_style: 'transparent', stroke_width: 1, fixed_position: true };
    configDraw(ctx, transform, draw_config, false, false, () => {
      this.drawBrushes(ctx);
      this.drawTiles(ctx);
    });
  }

  private label(ctx: CanvasRenderingContext2D, text: string, y: number): void {
    drawText(ctx, text, {
      p: { x: this.bounds.x + PADDING, y },
      w: this.bounds.w - 2 * PADDING,
      fill_style: INK,
      font: FONT,
      baseline: 'top',
    });
  }

  private drawBrushes(ctx: CanvasRenderingContext2D): void {
    this.label(ctx, 'Brush size (spaces)', this.bounds.y + PADDING);
    BRUSHES.forEach((brush, i) => {
      const rect = this.brushRect(i);
      const active = brush.radius === this.radius;
      const hovered = this.hover?.kind === 'brush' && this.hover.index === i;
      ctx.fillStyle = active ? 'rgb(255, 232, 160)' : hovered ? 'rgb(255, 243, 210)' : 'rgb(241, 226, 196)';
      ctx.strokeStyle = INK;
      ctx.lineWidth = active ? 1.6 : 0.8;
      drawRect(ctx, rect, rect.w, rect.h, 3);
      drawText(ctx, brush.label, {
        p: { x: rect.x + 0.5 * rect.w, y: rect.y + 0.5 * rect.h },
        w: rect.w,
        fill_style: INK,
        font: FONT,
        align: 'center',
        baseline: 'middle',
      });
    });
  }

  private drawTiles(ctx: CanvasRenderingContext2D): void {
    const terrains = this.context.terrains();
    const shown =
      this.hover?.kind === 'tile'
        ? terrains[this.hover.index]
        : terrains.find((t) => t.terrain_id === this.selected_id);
    this.label(
      ctx,
      shown ? `${shown.display_name} (#${shown.terrain_id})` : 'Terrain',
      this.gridRect().y - LABEL_H - 2
    );
    const grid = this.gridRect();
    ctx.save();
    ctx.beginPath();
    ctx.rect(grid.x, grid.y, grid.w, grid.h);
    ctx.clip();
    terrains.forEach((terrain, i) => {
      const rect = this.tileRect(i);
      const center = { x: rect.x + 0.5 * rect.w, y: rect.y + 0.5 * rect.h };
      drawHexImage(ctx, this.context.getIcon(terrainImage(terrain.terrain_id)), center, HEX_R);
      const hovered = this.hover?.kind === 'tile' && this.hover.index === i;
      if (terrain.terrain_id === this.selected_id || hovered) {
        ctx.fillStyle = hovered ? 'rgba(255, 255, 255, 0.25)' : 'transparent';
        ctx.strokeStyle = terrain.terrain_id === this.selected_id ? 'rgb(255, 255, 255)' : INK;
        ctx.lineWidth = 2;
        drawHexagon(ctx, center, HEX_R);
      }
    });
    ctx.restore();
  }

  xi(): number {
    return this.bounds.x;
  }
  yi(): number {
    return this.bounds.y;
  }
  xf(): number {
    return this.bounds.x + this.bounds.w;
  }
  yf(): number {
    return this.bounds.y + this.bounds.h;
  }
  w(): number {
    return this.bounds.w;
  }
  h(): number {
    return this.bounds.h;
  }
}
