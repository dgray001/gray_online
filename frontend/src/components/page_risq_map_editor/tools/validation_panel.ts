import type { BoardTransformData } from '../../game/util/canvas_board/canvas_board';
import { RisqTerrainType } from '../../game/games/risq/rendering/terrain';
import { configDraw } from '../../game/util/canvas_components/canvas_component';
import { drawRect, drawText } from '../../game/util/canvas_util';
import type { Point2D } from '../../game/util/objects2d';
import type { EditorConfigPanel, EditorToolContext, PanelBounds } from '../editor_tool';
import { DEFAULT_TERRAIN_ID } from '../map_doc';

export interface ValidationIssue {
  severity: 'error' | 'warning' | 'info';
  message: string;
  space?: Point2D;
}

const PADDING = 6;
const GAP = 4;
const ROW_H = 26;
const INK = 'rgb(59, 36, 19)';
const FONT = '12px "Fira Sans", sans-serif';
const BOLD_FONT = 'bold 12px "Fira Sans", sans-serif';
function contains(rect: PanelBounds, p: Point2D): boolean {
  return p.x >= rect.x && p.x <= rect.x + rect.w && p.y >= rect.y && p.y <= rect.y + rect.h;
}

export class ValidationPanel implements EditorConfigPanel {
  private bounds: PanelBounds = { x: 0, y: 0, w: 0, h: 0 };
  private mouse: Point2D = { x: 0, y: 0 };
  private scroll_offset = 0;
  private issues: ValidationIssue[] = [];

  constructor(
    private context: EditorToolContext,
    private on_select_issue: (issue: ValidationIssue) => void
  ) {}

  setBounds(bounds: PanelBounds): void {
    this.bounds = bounds;
    this.refresh();
  }

  refresh(): void {
    this.issues = this.runValidation();
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

  mouseup(_e: MouseEvent): void {}
  mousedown(e: MouseEvent): boolean {
    if (e.button !== 0 || !contains(this.bounds, this.mouse)) {
      return false;
    }
    const start_y = this.bounds.y + PADDING + ROW_H + GAP;
    for (let i = 0; i < this.issues.length; i++) {
      const rect: PanelBounds = {
        x: this.bounds.x + PADDING,
        y: start_y + i * (ROW_H + GAP) - this.scroll_offset,
        w: this.bounds.w - 2 * PADDING,
        h: ROW_H,
      };
      if (contains(rect, this.mouse)) {
        this.on_select_issue(this.issues[i]);
        return true;
      }
    }
    return true;
  }
  private validateStarts(issues: ValidationIssue[]): void {
    const doc = this.context.doc();
    if (!doc) {
      return;
    }
    if (doc.player_start) {
      this.validateStartSlots(issues);
      return;
    }
    const owners = new Set<number>();
    for (const space of doc.spaces) {
      for (const zone of space.zones ?? []) {
        if (zone.building) {
          owners.add(zone.building.player);
        }
        for (const unit of zone.units ?? []) {
          if (unit.count > 0) {
            owners.add(unit.player);
          }
        }
      }
    }
    for (let p = 0; p < doc.players; p++) {
      if (!owners.has(p)) {
        issues.push({ severity: 'error', message: `Slot P${p + 1} has no starting units or buildings` });
      }
    }
  }
  private validateStartSlots(issues: ValidationIssue[]): void {
    const doc = this.context.doc()!;
    const slots = doc.spaces.filter((space) => space.player_slot !== undefined);
    for (let player = 0; player < doc.players; player++) {
      if (slots.filter((space) => space.player_slot === player).length !== 1) {
        issues.push({ severity: 'error', message: `Slot P${player + 1} must have exactly one start marker` });
      }
    }
    for (const space of slots) {
      if (!Number.isInteger(space.player_slot) || space.player_slot! < 0 || space.player_slot! >= doc.players) {
        issues.push({ severity: 'error', message: 'Invalid player slot marker', space });
      }
    }
    if (
      !doc.player_start!.spaces.some((space) =>
        space.zones?.some((zone) => zone.building || zone.units?.some((unit) => unit.count > 0))
      )
    ) {
      issues.push({ severity: 'error', message: 'Start template has no starting units or buildings' });
    }
  }

  private validateConnectivity(issues: ValidationIssue[]): void {
    const doc = this.context.doc();
    if (!doc || doc.spaces.length === 0) {
      return;
    }
    const blocked = new Set(
      this.context
        .terrains()
        .filter(
          (terrain) =>
            terrain.terrain_type === RisqTerrainType.WATER || terrain.terrain_type === RisqTerrainType.DEEP_WATER
        )
        .map((terrain) => terrain.terrain_id)
    );
    const spaces = doc.spaces.filter((space) => !blocked.has(space.terrain ?? DEFAULT_TERRAIN_ID));
    if (spaces.length === 0) {
      return;
    }
    const space_map = new Map<string, Point2D>();
    spaces.forEach((s) => space_map.set(`${s.x},${s.y}`, { x: s.x, y: s.y }));
    const visited = new Set<string>();
    const queue: Point2D[] = [{ x: spaces[0].x, y: spaces[0].y }];
    visited.add(`${spaces[0].x},${spaces[0].y}`);
    while (queue.length > 0) {
      const curr = queue.shift()!;
      for (const [dx, dy] of [
        [1, 0],
        [1, -1],
        [0, -1],
        [-1, 0],
        [-1, 1],
        [0, 1],
      ]) {
        const nx = curr.x + dx;
        const ny = curr.y + dy;
        const key = `${nx},${ny}`;
        if (space_map.has(key) && !visited.has(key)) {
          visited.add(key);
          queue.push({ x: nx, y: ny });
        }
      }
      for (const connection of doc.connections ?? []) {
        const key = `${curr.x},${curr.y}`;
        const target =
          connection.from.join(',') === key
            ? connection.to
            : connection.to.join(',') === key
              ? connection.from
              : undefined;
        if (target && space_map.has(target.join(',')) && !visited.has(target.join(','))) {
          visited.add(target.join(','));
          queue.push({ x: target[0], y: target[1] });
        }
      }
    }
    if (visited.size < spaces.length) {
      issues.push({
        severity: 'info',
        message: `${spaces.length - visited.size} playable spaces are outside the first land component`,
      });
    }
  }
  private validateFairnessAndIds(issues: ValidationIssue[]): void {
    const doc = this.context.doc();
    if (!doc) {
      return;
    }
    const known_buildings = new Set([1, 2, 3, 11, 21, 22, 23]);
    const known_units = new Set([1, 11, 12, 13]);
    for (const space of [...doc.spaces, ...(doc.player_start?.spaces ?? [])]) {
      for (const zone of space.zones ?? []) {
        if (zone.building && !known_buildings.has(zone.building.id)) {
          issues.push({
            severity: 'warning',
            message: `Unknown building ID: ${zone.building.id}`,
            space: { x: space.x, y: space.y },
          });
        }
        for (const u of zone.units ?? []) {
          if (!known_units.has(u.id)) {
            issues.push({
              severity: 'warning',
              message: `Unknown unit ID: ${u.id}`,
              space: { x: space.x, y: space.y },
            });
          }
        }
      }
    }
  }
  private runValidation(): ValidationIssue[] {
    const issues: ValidationIssue[] = [];
    const doc = this.context.doc();
    if (!doc) {
      return issues;
    }
    this.validateStarts(issues);
    this.validateConnectivity(issues);
    this.validateFairnessAndIds(issues);
    if (issues.length === 0) {
      issues.push({ severity: 'info', message: 'Map passed all validation checks!' });
    }
    return issues;
  }
  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, _dt: number): void {
    if (this.bounds.w <= 0 || this.bounds.h <= 0) {
      return;
    }
    const draw_config = { fill_style: 'transparent', stroke_width: 1, fixed_position: true };
    configDraw(ctx, transform, draw_config, false, false, () => {
      const x = this.bounds.x + PADDING;
      let y = this.bounds.y + PADDING;
      drawText(ctx, 'Map Validation', { p: { x, y: y + 10 }, w: this.bounds.w, fill_style: INK, font: BOLD_FONT });
      y += ROW_H + GAP;
      this.drawIssues(ctx, y);
    });
  }

  private drawIssues(ctx: CanvasRenderingContext2D, start_y: number): void {
    this.issues.forEach((issue, i) => {
      const rect: PanelBounds = {
        x: this.bounds.x + PADDING,
        y: start_y + i * (ROW_H + GAP) - this.scroll_offset,
        w: this.bounds.w - 2 * PADDING,
        h: ROW_H,
      };
      if (rect.y + rect.h < this.bounds.y || rect.y > this.bounds.y + this.bounds.h) {
        return;
      }
      ctx.fillStyle =
        issue.severity === 'error'
          ? 'rgba(255, 100, 100, 0.25)'
          : issue.severity === 'warning'
            ? 'rgba(255, 200, 100, 0.25)'
            : 'rgba(100, 220, 100, 0.25)';
      drawRect(ctx, rect, rect.w, rect.h, 3);
      drawText(ctx, issue.message, {
        p: { x: rect.x + 6, y: rect.y + rect.h / 2 },
        w: rect.w - 12,
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
