import { drawHexagon } from '../../game/util/canvas_util';
import type { EditCommand } from '../edit_history';
import type { EditorConfigPanel, EditorTarget, EditorToolContext } from '../editor_tool';
import { EditorTool } from '../editor_tool';
import type { MapDoc, MapDocRegion } from '../map_doc';
import { RegionPanel } from './region_panel';

const REGION_COLORS = [
  'rgba(255, 100, 100, 0.25)',
  'rgba(100, 200, 255, 0.25)',
  'rgba(100, 255, 120, 0.25)',
  'rgba(255, 200, 80, 0.25)',
  'rgba(200, 120, 255, 0.25)',
  'rgba(255, 160, 200, 0.25)',
];
class RegionEditCommand implements EditCommand {
  constructor(
    private before: MapDocRegion[],
    private after: MapDocRegion[]
  ) {}

  isEmpty(): boolean {
    return JSON.stringify(this.before) === JSON.stringify(this.after);
  }

  apply(doc: MapDoc): void {
    doc.regions = JSON.parse(JSON.stringify(this.after));
  }

  revert(doc: MapDoc): void {
    doc.regions = JSON.parse(JSON.stringify(this.before));
  }
}
export class RegionTool extends EditorTool {
  readonly id = 'region';
  readonly label = 'Regions';
  readonly hotkey = 'r';
  private panel: RegionPanel;

  constructor(context: EditorToolContext) {
    super(context);
    this.panel = new RegionPanel(
      context,
      () => this.addRegion(),
      (i) => this.deleteRegion(i),
      (i, d) => this.changeBonus(i, d)
    );
  }

  config(): EditorConfigPanel {
    return this.panel;
  }

  activated(): void {}
  deactivated(): void {}
  private addRegion(): void {
    const doc = this.context.doc();
    if (!doc) {
      return;
    }
    const before = JSON.parse(JSON.stringify(doc.regions ?? []));
    doc.regions = doc.regions ?? [];
    let number = 1;
    while (doc.regions.some((region) => region.name === `Region ${number}`)) {
      number++;
    }
    doc.regions.push({ name: `Region ${number}`, gold_bonus: 3, spaces: [] });
    const after = JSON.parse(JSON.stringify(doc.regions));
    this.context.history.record(new RegionEditCommand(before, after));
    this.context.edited();
  }

  private deleteRegion(index: number): void {
    const doc = this.context.doc();
    if (!doc || !doc.regions || index < 0 || index >= doc.regions.length) {
      return;
    }
    const before = JSON.parse(JSON.stringify(doc.regions));
    doc.regions.splice(index, 1);
    const after = JSON.parse(JSON.stringify(doc.regions));
    this.context.history.record(new RegionEditCommand(before, after));
    this.context.edited();
  }
  private changeBonus(index: number, delta: number): void {
    const doc = this.context.doc();
    if (!doc || !doc.regions || index < 0 || index >= doc.regions.length) {
      return;
    }
    const before = JSON.parse(JSON.stringify(doc.regions));
    const region = doc.regions[index];
    region.gold_bonus = Math.max(0, (region.gold_bonus ?? 0) + delta);
    const after = JSON.parse(JSON.stringify(doc.regions));
    this.context.history.record(new RegionEditCommand(before, after));
    this.context.edited();
  }

  private toggleSpace(target: EditorTarget): void {
    const doc = this.context.doc();
    if (
      !doc ||
      !doc.regions ||
      doc.regions.length === 0 ||
      !doc.spaces.some((space) => space.x === target.space.x && space.y === target.space.y)
    ) {
      return;
    }
    const active_idx = this.panel.activeIndex();
    const region = doc.regions[active_idx];
    if (!region) {
      return;
    }
    const before = JSON.parse(JSON.stringify(doc.regions));
    const coord: [number, number] = [target.space.x, target.space.y];
    const exists_in_active = region.spaces.some(([x, y]) => x === coord[0] && y === coord[1]);
    for (const r of doc.regions) {
      r.spaces = r.spaces.filter(([x, y]) => !(x === coord[0] && y === coord[1]));
    }
    if (!exists_in_active) {
      region.spaces.push(coord);
    }
    const after = JSON.parse(JSON.stringify(doc.regions));
    this.context.history.record(new RegionEditCommand(before, after));
    this.context.edited();
  }
  mousedown(target: EditorTarget | undefined, e: MouseEvent): boolean {
    if (e.button !== 0 || !target) {
      return false;
    }
    this.toggleSpace(target);
    return true;
  }

  mousemove(_target: EditorTarget | undefined): void {}
  mouseup(_target: EditorTarget | undefined, _e: MouseEvent): void {}

  drawOverlay(ctx: CanvasRenderingContext2D, target: EditorTarget | undefined): void {
    const doc = this.context.doc();
    if (!doc || !doc.regions) {
      return;
    }
    const hex_r = this.context.hexR();
    const active_idx = this.panel.activeIndex();
    doc.regions.forEach((region, r_idx) => {
      ctx.fillStyle = REGION_COLORS[r_idx % REGION_COLORS.length];
      ctx.strokeStyle = r_idx === active_idx ? 'rgba(255, 215, 0, 0.9)' : 'rgba(255, 255, 255, 0.3)';
      ctx.lineWidth = r_idx === active_idx ? 2.5 : 1;
      for (const [x, y] of region.spaces) {
        drawHexagon(ctx, this.context.toCanvas({ x, y }), hex_r);
      }
    });
    if (target) {
      ctx.strokeStyle = 'rgba(255, 215, 0, 0.8)';
      ctx.lineWidth = 2;
      drawHexagon(ctx, this.context.toCanvas(target.space), hex_r);
    }
  }
}
