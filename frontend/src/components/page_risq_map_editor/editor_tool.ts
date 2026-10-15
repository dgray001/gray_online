import type { CanvasComponent } from '../game/util/canvas_components/canvas_component';
import type { Point2D } from '../game/util/objects2d';
import type { EditHistory } from './edit_history';
import type { EntityCatalog, TerrainEntry } from './map_api';
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
  isTyping?: () => boolean;
}

export interface EditorToolContext {
  history: EditHistory;
  doc: () => MapDoc | undefined;
  edited: () => void;
  patchTerrain: (coordinates: Point2D[], terrain_id: number) => void;
  select: (target?: EditorTarget) => void;
  dragged: () => boolean;
  terrains: () => TerrainEntry[];
  entities: () => EntityCatalog;
  panTo: (coordinate: Point2D) => void;
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

import { BackgroundTool } from './tools/background_tool';
import { ConnectionsTool } from './tools/connections_tool';
import { RegionTool } from './tools/region_tool';
import { SelectTool } from './tools/select_tool';
import { SlotsTool } from './tools/slots_tool';
import { SpacesTool } from './tools/spaces_tool';
import { TerrainTool } from './tools/terrain_tool';
import { ValidationTool } from './tools/validation_tool';
import { ZoneTool } from './tools/zone_tool';

export function createEditorTools(context: EditorToolContext): EditorTool[] {
  return [
    new SelectTool(context),
    new SpacesTool(context),
    new TerrainTool(context),
    new ZoneTool(context),
    new SlotsTool(context),
    new RegionTool(context),
    new ConnectionsTool(context),
    new BackgroundTool(context),
    new ValidationTool(context),
  ];
}
