import type { BoardTransformData } from '../../game/util/canvas_board/canvas_board';
import { configDraw } from '../../game/util/canvas_components/canvas_component';
import { drawRect, drawText } from '../../game/util/canvas_util';
import type { Point2D } from '../../game/util/objects2d';
import type { EditorConfigPanel, EditorToolContext, PanelBounds } from '../editor_tool';

export type ZoneToolMode = 'resource' | 'building' | 'unit' | 'terrain' | 'clear';

export interface ZoneConfigItem {
  id: number;
  label: string;
}
const RESOURCE_ITEMS: ZoneConfigItem[] = [
  { id: 1, label: 'Forage Bush' },
  { id: 2, label: 'Deer' },
  { id: 11, label: 'Cedar Tree' },
  { id: 41, label: 'Stone Mine' },
];

const BUILDING_ITEMS: ZoneConfigItem[] = [
  { id: 1, label: 'Village Center' },
  { id: 2, label: 'Housing' },
  { id: 3, label: 'Farm' },
  { id: 11, label: 'Blacksmith' },
  { id: 21, label: 'Outpost' },
  { id: 22, label: 'Barracks' },
  { id: 23, label: 'Redoubt' },
];
const UNIT_ITEMS: ZoneConfigItem[] = [
  { id: 1, label: 'Villager' },
  { id: 11, label: 'Blunt Infantry' },
  { id: 12, label: 'Piercing Infantry' },
  { id: 13, label: 'Heavy Infantry' },
];

const MODES: { mode: ZoneToolMode; label: string }[] = [
  { mode: 'resource', label: 'Res' },
  { mode: 'building', label: 'Bld' },
  { mode: 'unit', label: 'Unt' },
  { mode: 'terrain', label: 'Ter' },
  { mode: 'clear', label: 'Clr' },
];

const PADDING = 6;
const GAP = 4;
const ROW_H = 24;
const INK = 'rgb(59, 36, 19)';
const FONT = '12px "Fira Sans", sans-serif';

function contains(rect: PanelBounds, p: Point2D): boolean {
  return p.x >= rect.x && p.x <= rect.x + rect.w && p.y >= rect.y && p.y <= rect.y + rect.h;
}
export class ZonePalette implements EditorConfigPanel {
  private bounds: PanelBounds = { x: 0, y: 0, w: 0, h: 0 };
  private active_mode: ZoneToolMode = 'building';
  private selected_resource = 1;
  private selected_building = 1;
  private selected_unit = 1;
  private selected_terrain = 1;
  private selected_slot = 0;
  private unit_count = 1;
  private scroll_offset = 0;
  private mouse: Point2D = { x: 0, y: 0 };

  constructor(private context: EditorToolContext) {}

  mode(): ZoneToolMode {
    return this.active_mode;
  }

  resourceId(): number {
    return this.selected_resource;
  }
  buildingId(): number {
    return this.selected_building;
  }

  unitId(): number {
    return this.selected_unit;
  }

  terrainId(): number {
    return this.selected_terrain;
  }

  playerSlot(): number {
    return this.selected_slot;
  }

  unitCount(): number {
    return this.unit_count;
  }

  setBounds(bounds: PanelBounds): void {
    this.bounds = bounds;
  }
  private modeRect(index: number): PanelBounds {
    const w = (this.bounds.w - 2 * PADDING - (MODES.length - 1) * GAP) / MODES.length;
    return { x: this.bounds.x + PADDING + index * (w + GAP), y: this.bounds.y + PADDING, w, h: ROW_H };
  }

  private slotRect(slot: number, total: number): PanelBounds {
    const count = Math.max(1, total);
    const w = Math.min(28, (this.bounds.w - 2 * PADDING - (count - 1) * GAP) / count);
    return { x: this.bounds.x + PADDING + slot * (w + GAP), y: this.bounds.y + PADDING + ROW_H + GAP, w, h: ROW_H };
  }

  private itemRect(index: number, top_y: number): PanelBounds {
    return {
      x: this.bounds.x + PADDING,
      y: top_y + index * (ROW_H + GAP) - this.scroll_offset,
      w: this.bounds.w - 2 * PADDING,
      h: ROW_H,
    };
  }
  mousemove(_canvas: Point2D, screen: Point2D, _transform: BoardTransformData): boolean {
    this.mouse = screen;
    return contains(this.bounds, screen);
  }

  isHovering(): boolean {
    return contains(this.bounds, this.mouse);
  }

  setHovering(_hovering: boolean): void {}
  isClicking(): boolean {
    return false;
  }
  setClicking(_clicking: boolean): void {}

  scroll(dy: number): boolean {
    if (!contains(this.bounds, this.mouse)) {
      return false;
    }
    this.scroll_offset = Math.max(0, this.scroll_offset + dy);
    return true;
  }
  mousedown(e: MouseEvent): boolean {
    if (e.button !== 0 || !contains(this.bounds, this.mouse)) {
      return false;
    }
    for (let i = 0; i < MODES.length; i++) {
      if (contains(this.modeRect(i), this.mouse)) {
        this.active_mode = MODES[i].mode;
        this.scroll_offset = 0;
        return true;
      }
    }
    return this.handleSubmenuClick();
  }

  mouseup(_e: MouseEvent): void {}
  private handleSubmenuClick(): boolean {
    const doc = this.context.doc();
    const players = doc?.players ?? 1;
    if (this.active_mode === 'building' || this.active_mode === 'unit') {
      for (let s = 0; s < players; s++) {
        if (contains(this.slotRect(s, players), this.mouse)) {
          this.selected_slot = s;
          return true;
        }
      }
    }
    const top_y =
      this.bounds.y +
      PADDING +
      (this.active_mode === 'building' || this.active_mode === 'unit' ? (ROW_H + GAP) * 2 : ROW_H + GAP);
    const items = this.currentItems();
    for (let i = 0; i < items.length; i++) {
      if (contains(this.itemRect(i, top_y), this.mouse)) {
        this.selectItem(items[i].id);
        return true;
      }
    }
    return false;
  }
  private currentItems(): ZoneConfigItem[] {
    if (this.active_mode === 'resource') {
      return RESOURCE_ITEMS;
    }
    if (this.active_mode === 'building') {
      return BUILDING_ITEMS;
    }
    if (this.active_mode === 'unit') {
      return UNIT_ITEMS;
    }
    if (this.active_mode === 'terrain') {
      return this.context.terrains().map((t) => ({ id: t.terrain_id, label: t.display_name }));
    }
    return [];
  }

  private selectItem(id: number): void {
    if (this.active_mode === 'resource') {
      this.selected_resource = id;
    } else if (this.active_mode === 'building') {
      this.selected_building = id;
    } else if (this.active_mode === 'unit') {
      this.selected_unit = id;
    } else if (this.active_mode === 'terrain') {
      this.selected_terrain = id;
    }
  }
  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, _dt: number): void {
    if (this.bounds.w <= 0 || this.bounds.h <= 0) {
      return;
    }
    const draw_config = { fill_style: 'transparent', stroke_width: 1, fixed_position: true };
    configDraw(ctx, transform, draw_config, false, false, () => {
      this.drawModes(ctx);
      const doc = this.context.doc();
      const players = doc?.players ?? 1;
      let top_y = this.bounds.y + PADDING + ROW_H + GAP;
      if (this.active_mode === 'building' || this.active_mode === 'unit') {
        this.drawSlots(ctx, players, top_y);
        top_y += ROW_H + GAP;
      }
      this.drawItemList(ctx, top_y);
    });
  }

  private drawModes(ctx: CanvasRenderingContext2D): void {
    MODES.forEach((m, i) => {
      const rect = this.modeRect(i);
      const active = this.active_mode === m.mode;
      ctx.fillStyle = active
        ? 'rgb(200, 160, 120)'
        : contains(rect, this.mouse)
          ? 'rgb(220, 200, 180)'
          : 'rgb(240, 230, 215)';
      drawRect(ctx, rect, rect.w, rect.h, 3);
      drawText(ctx, m.label, {
        p: { x: rect.x + rect.w / 2, y: rect.y + rect.h / 2 },
        w: rect.w,
        fill_style: INK,
        font: FONT,
        align: 'center',
        baseline: 'middle',
      });
    });
  }
  private drawSlots(ctx: CanvasRenderingContext2D, players: number, y: number): void {
    for (let s = 0; s < players; s++) {
      const rect = this.slotRect(s, players);
      rect.y = y;
      const active = this.selected_slot === s;
      ctx.fillStyle = active
        ? 'rgb(200, 160, 120)'
        : contains(rect, this.mouse)
          ? 'rgb(220, 200, 180)'
          : 'rgb(240, 230, 215)';
      drawRect(ctx, rect, rect.w, rect.h, 3);
      drawText(ctx, `P${s + 1}`, {
        p: { x: rect.x + rect.w / 2, y: rect.y + rect.h / 2 },
        w: rect.w,
        fill_style: INK,
        font: FONT,
        align: 'center',
        baseline: 'middle',
      });
    }
  }

  private drawItemList(ctx: CanvasRenderingContext2D, top_y: number): void {
    const items = this.currentItems();
    const active_id =
      this.active_mode === 'resource'
        ? this.selected_resource
        : this.active_mode === 'building'
          ? this.selected_building
          : this.active_mode === 'unit'
            ? this.selected_unit
            : this.selected_terrain;
    items.forEach((item, i) => {
      const rect = this.itemRect(i, top_y);
      if (rect.y + rect.h < this.bounds.y || rect.y > this.bounds.y + this.bounds.h) {
        return;
      }
      const active = active_id === item.id;
      ctx.fillStyle = active
        ? 'rgb(200, 160, 120)'
        : contains(rect, this.mouse)
          ? 'rgb(220, 200, 180)'
          : 'rgb(240, 230, 215)';
      drawRect(ctx, rect, rect.w, rect.h, 3);
      drawText(ctx, item.label, {
        p: { x: rect.x + 8, y: rect.y + rect.h / 2 },
        w: rect.w - 16,
        fill_style: INK,
        font: FONT,
        baseline: 'middle',
      });
    });
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
