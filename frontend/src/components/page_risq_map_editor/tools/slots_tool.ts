import type { EditCommand } from '../edit_history';
import type { EditorConfigPanel, EditorTarget, EditorToolContext } from '../editor_tool';
import { EditorTool } from '../editor_tool';
import type { MapDoc, MapDocBank } from '../map_doc';
import { SlotsPanel } from './slots_panel';

interface SlotsSnapshot {
  players: number;
  starting_bank?: MapDocBank;
  spaces_json: string;
}
class SlotsEditCommand implements EditCommand {
  constructor(
    private before: SlotsSnapshot,
    private after: SlotsSnapshot
  ) {}

  isEmpty(): boolean {
    return (
      this.before.players === this.after.players &&
      JSON.stringify(this.before.starting_bank) === JSON.stringify(this.after.starting_bank) &&
      this.before.spaces_json === this.after.spaces_json
    );
  }

  apply(doc: MapDoc): void {
    doc.players = this.after.players;
    doc.starting_bank = this.after.starting_bank ? JSON.parse(JSON.stringify(this.after.starting_bank)) : undefined;
    doc.spaces = JSON.parse(this.after.spaces_json);
  }

  revert(doc: MapDoc): void {
    doc.players = this.before.players;
    doc.starting_bank = this.before.starting_bank ? JSON.parse(JSON.stringify(this.before.starting_bank)) : undefined;
    doc.spaces = JSON.parse(this.before.spaces_json);
  }
}
export class SlotsTool extends EditorTool {
  readonly id = 'slots';
  readonly label = 'Slots';
  readonly hotkey = 'l';
  private panel: SlotsPanel;

  constructor(context: EditorToolContext) {
    super(context);
    this.panel = new SlotsPanel(
      context,
      (delta) => this.changeSlots(delta),
      (res, delta) => this.changeBank(res, delta)
    );
  }

  config(): EditorConfigPanel {
    return this.panel;
  }

  activated(): void {}
  deactivated(): void {}
  private snapshot(doc: MapDoc): SlotsSnapshot {
    return {
      players: doc.players,
      starting_bank: doc.starting_bank ? JSON.parse(JSON.stringify(doc.starting_bank)) : undefined,
      spaces_json: JSON.stringify(doc.spaces),
    };
  }

  private changeSlots(delta: number): void {
    const doc = this.context.doc();
    if (!doc) {
      return;
    }
    const next_players = Math.max(1, Math.min(12, doc.players + delta));
    if (next_players === doc.players) {
      return;
    }
    const before = this.snapshot(doc);
    doc.players = next_players;
    if (delta < 0) {
      this.stripPlayerEntities(doc, next_players);
    }
    const after = this.snapshot(doc);
    this.context.history.record(new SlotsEditCommand(before, after));
    this.context.edited();
  }
  private changeBank(resource: 'food' | 'wood' | 'stone' | 'gold', delta: number): void {
    const doc = this.context.doc();
    if (!doc) {
      return;
    }
    const before = this.snapshot(doc);
    doc.starting_bank = doc.starting_bank ?? { food: 0, wood: 0, stone: 0, gold: 0 };
    doc.starting_bank[resource] = Math.max(0, doc.starting_bank[resource] + delta);
    const after = this.snapshot(doc);
    this.context.history.record(new SlotsEditCommand(before, after));
    this.context.edited();
  }

  private stripPlayerEntities(doc: MapDoc, max_players: number): void {
    for (const space of doc.spaces) {
      if (space.player_slot !== undefined && space.player_slot >= max_players) {
        delete space.player_slot;
      }
      if (!space.zones) {
        continue;
      }
      for (const zone of space.zones) {
        if (zone.building && zone.building.player >= max_players) {
          delete zone.building;
        }
        if (zone.units) {
          zone.units = zone.units.filter((u) => u.player < max_players);
          if (zone.units.length === 0) {
            delete zone.units;
          }
        }
      }
      space.zones = space.zones.filter((z) => z.resource || z.building || z.units?.length || z.terrain_override);
    }
  }
  mousedown(_target: EditorTarget | undefined, _e: MouseEvent): boolean {
    return false;
  }
  mousemove(_target: EditorTarget | undefined): void {}
  mouseup(_target: EditorTarget | undefined, _e: MouseEvent): void {}

  drawOverlay(ctx: CanvasRenderingContext2D, target: EditorTarget | undefined): void {
    if (!target) {
      return;
    }
    const canvas_pt = this.context.toCanvas(target.space);
    ctx.strokeStyle = 'rgba(255, 255, 255, 0.4)';
    ctx.lineWidth = 1.5;
    ctx.strokeRect(canvas_pt.x - 20, canvas_pt.y - 20, 40, 40);
  }
}
