import { drawHexagon } from '../../game/util/canvas_util';
import type { Point2D } from '../../game/util/objects2d';
import type { EditCommand } from '../edit_history';
import type { EditorConfigPanel, EditorTarget, EditorToolContext } from '../editor_tool';
import { EditorTool } from '../editor_tool';
import type { MapDoc, MapDocRegion, MapDocValue } from '../map_doc';
import { RegionPanel } from './region_panel';

const REGION_COLORS = [
  'rgba(255, 100, 100, 0.25)',
  'rgba(100, 200, 255, 0.25)',
  'rgba(100, 255, 120, 0.25)',
  'rgba(255, 200, 80, 0.25)',
  'rgba(200, 120, 255, 0.25)',
  'rgba(255, 160, 200, 0.25)',
];
class RegionListCommand implements EditCommand {
  constructor(
    private index: number,
    private region: MapDocRegion,
    private adding: boolean
  ) {}

  apply(doc: MapDoc): void {
    this.adding ? this.insert(doc) : this.remove(doc);
  }

  revert(doc: MapDoc): void {
    this.adding ? this.remove(doc) : this.insert(doc);
  }

  private insert(doc: MapDoc): void {
    doc.regions = doc.regions ?? [];
    doc.regions.splice(this.index, 0, structuredClone(this.region));
  }

  private remove(doc: MapDoc): void {
    doc.regions?.splice(this.index, 1);
  }
}
class RegionBonusCommand implements EditCommand {
  constructor(
    private index: number,
    private before: MapDocValue | undefined,
    private after: MapDocValue | undefined
  ) {}

  apply(doc: MapDoc): void {
    this.set(doc, this.after);
  }

  revert(doc: MapDoc): void {
    this.set(doc, this.before);
  }

  private set(doc: MapDoc, bonus: MapDocValue | undefined): void {
    const region = doc.regions?.[this.index];
    if (region) {
      region.gold_bonus = bonus;
    }
  }
}

class RegionMembershipCommand implements EditCommand {
  constructor(
    private coord: Point2D,
    private from: number | undefined,
    private to: number | undefined
  ) {}

  apply(doc: MapDoc): void {
    this.move(doc, this.from, this.to);
  }

  revert(doc: MapDoc): void {
    this.move(doc, this.to, this.from);
  }

  private move(doc: MapDoc, source: number | undefined, target: number | undefined): void {
    const left = source === undefined ? undefined : doc.regions?.[source];
    if (left) {
      left.spaces = left.spaces.filter(([x, y]) => x !== this.coord.x || y !== this.coord.y);
    }
    const joined = target === undefined ? undefined : doc.regions?.[target];
    joined?.spaces.push([this.coord.x, this.coord.y]);
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
    const regions = doc.regions ?? [];
    let number = 1;
    while (regions.some((region) => region.name === `Region ${number}`)) {
      number++;
    }
    const region = { name: `Region ${number}`, gold_bonus: 3, spaces: [] };
    this.commit(new RegionListCommand(regions.length, region, true));
  }

  private commit(command: EditCommand): void {
    const doc = this.context.doc();
    if (doc) {
      command.apply(doc);
      this.context.history.record(command);
      this.context.edited();
    }
  }

  private deleteRegion(index: number): void {
    const doc = this.context.doc();
    if (!doc || !doc.regions || index < 0 || index >= doc.regions.length) {
      return;
    }
    this.commit(new RegionListCommand(index, doc.regions[index], false));
  }
  private changeBonus(index: number, delta: number): void {
    const doc = this.context.doc();
    if (!doc || !doc.regions || index < 0 || index >= doc.regions.length) {
      return;
    }
    const before = doc.regions[index].gold_bonus;
    if (typeof before === 'string') {
      return;
    }
    const after = Math.max(0, (before ?? 0) + delta);
    if (after !== before) {
      this.commit(new RegionBonusCommand(index, before, after));
    }
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
    const owner = doc.regions.findIndex((r) => r.spaces.some(([x, y]) => x === target.space.x && y === target.space.y));
    this.commit(
      new RegionMembershipCommand(
        target.space,
        owner < 0 ? undefined : owner,
        owner === active_idx ? undefined : active_idx
      )
    );
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
