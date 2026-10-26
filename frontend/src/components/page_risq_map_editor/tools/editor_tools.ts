import type { EditorTool, EditorToolContext } from '../editor_tool';
import { BackgroundTool } from './background_tool';
import { ConnectionsTool } from './connections_tool';
import { RegionTool } from './region_tool';
import { SelectTool } from './select_tool';
import { SlotsTool } from './slots_tool';
import { SpacesTool } from './spaces_tool';
import { TerrainTool } from './terrain_tool';
import { ValidationTool } from './validation_tool';
import { ZoneTool } from './zone_tool';

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
