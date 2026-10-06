import type { CanvasComponent } from '../game/util/canvas_components/canvas_component';
import type { Point2D } from '../game/util/objects2d';
import type { EditHistory } from './edit_history';
import type { TerrainEntry } from './map_api';
import type { MapDoc } from './map_doc';

export interface EditorTarget {
  space: Point2D;
  zone?: Point2D;
}

export interface PanelBounds {
  x: number;
  y: number;
  w: number;
  h: number;
}

export interface EditorConfigPanel extends CanvasComponent {
  setBounds: (bounds: PanelBounds) => void;
}

export interface EditorToolContext {
  history: EditHistory;
  doc: () => MapDoc | undefined;
  edited: () => void;
  patchTerrain: (coordinates: Point2D[], terrain_id: number) => void;
  select: (target?: EditorTarget) => void;
  dragged: () => boolean;
  terrains: () => TerrainEntry[];
  getIcon: (name: string) => HTMLImageElement;
  toCanvas: (coordinate: Point2D) => Point2D;
  hexR: () => number;
}

export abstract class EditorTool {
  abstract readonly id: string;
  abstract readonly label: string;
  abstract readonly hotkey: string;

  constructor(protected context: EditorToolContext) {}

  abstract config(): EditorConfigPanel | undefined;
  abstract activated(): void;
  abstract deactivated(): void;
  abstract mousedown(target: EditorTarget | undefined, e: MouseEvent): boolean;
  abstract mousemove(target: EditorTarget | undefined): void;
  abstract mouseup(target: EditorTarget | undefined, e: MouseEvent): void;
  abstract drawOverlay(ctx: CanvasRenderingContext2D, target: EditorTarget | undefined): void;
}
