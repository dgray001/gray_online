import { DwgElement } from '../dwg_element';
import { apiGet } from '../../scripts/api';
import type { DwgCanvasBoard } from '../game/util/canvas_board/canvas_board';
import type { CanvasComponent } from '../game/util/canvas_components/canvas_component';
import { drawHexagon } from '../game/util/canvas_util';
import type { Point2D } from '../game/util/objects2d';
import { axialDistance, multiplyPoint2D, roundAxialCoordinate } from '../game/util/objects2d';
import { RisqSession } from '../game/games/risq/application/session';
import type { RisqResourceConfig } from '../game/games/risq/model/types';
import { RisqViewport } from '../game/games/risq/rendering/board/viewport';
import { RisqBoardRenderer } from '../game/games/risq/rendering/board/board_renderer';
import { terrainTypeLabel } from '../game/games/risq/rendering/terrain';
import { getRisqZone } from '../game/games/risq/rendering/zones/geometry';
import { EditHistory } from './edit_history';
import { EditorLeftPanel } from './editor_left_panel';
import { EditorRightPanel } from './editor_right_panel';
import type { EditorTarget, EditorToolContext } from './editor_tool';
import type { EditorTool } from './editor_tool';
import { EditorDrawHost } from './editor_host';
import { MapEditorHeader } from './map_editor_header';
import type { TerrainEntry } from './map_api';
import { deleteMap, listMaps, listTerrains, loadMap, previewMap, saveMap } from './map_api';
import type { MapDoc } from './map_doc';
import { newMapDoc } from './map_doc';
import { ConnectionsTool } from './tools/connections_tool';
import { RegionTool } from './tools/region_tool';
import { SelectTool } from './tools/select_tool';
import { SlotsTool } from './tools/slots_tool';
import { SpacesTool } from './tools/spaces_tool';
import { TerrainTool } from './tools/terrain_tool';
import { ValidationTool } from './tools/validation_tool';
import { ZoneTool } from './tools/zone_tool';

import html from './page_risq_map_editor.html';

import './page_risq_map_editor.scss';
import '../game/util/canvas_board/canvas_board';

const MAP_NAME_PATTERN = /^[a-z0-9_]{1,40}$/;
const MAX_BOARD_SIZE = 10;
const INITIAL_BOARD_SIZE = 2;

export class DwgPageRisqMapEditor extends DwgElement {
  private board!: DwgCanvasBoard;

  private session = new RisqSession();
  private viewport = new RisqViewport(this.session);
  private host = new EditorDrawHost(this.session, this.viewport);
  private history = new EditHistory();
  private terrains: TerrainEntry[] = [];
  private selection?: EditorTarget;
  private dragged = false;
  private context: EditorToolContext = {
    history: this.history,
    doc: () => this.doc,
    edited: () => this.requestPreview(),
    patchTerrain: (coordinates, terrain_id) => this.patchTerrain(coordinates, terrain_id),
    select: (target) => this.select(target),
    dragged: () => this.dragged,
    terrains: () => this.terrains,
    getIcon: (name) => this.host.getIcon(name),
    toCanvas: (coordinate) => this.viewport.coordinateToCanvas(coordinate),
    hexR: () => this.viewport.hexR(),
  };
  private tools: EditorTool[] = [
    new SelectTool(this.context),
    new TerrainTool(this.context),
    new SpacesTool(this.context),
    new ZoneTool(this.context),
    new SlotsTool(this.context),
    new RegionTool(this.context),
    new ConnectionsTool(this.context),
    new ValidationTool(this.context),
  ];
  private active_tool = 0;
  private header = new MapEditorHeader({
    load: (name) => this.openMap(name),
    create: (name, size) => this.createNew(name, size),
    save: (name) => this.save(name),
    remove: (name) => this.deleteSaved(name),
  });
  private left_panel = new EditorLeftPanel(
    () => this.canvasSize(),
    () => this.select(undefined)
  );
  private right_panel = new EditorRightPanel(
    this.tools,
    () => this.canvasSize(),
    (index) => this.setTool(index)
  );
  private renderer = new RisqBoardRenderer(
    this.host,
    this.session,
    this.viewport,
    { draw: () => {} },
    () => undefined,
    []
  );
  private doc?: MapDoc;
  private armed_delete?: string;
  private board_pixels: Point2D = { x: 0, y: 0 };
  private fitted_board_size = -1;
  private last_frame = Date.now();
  private preview_running = false;
  private preview_dirty = false;
  private preview_revision = 0;
  private load_revision = 0;

  constructor() {
    super();
    this.html_string = html;
    this.configureElement('board');
  }

  protected override parsedCallback(): void {
    document.addEventListener('keydown', this.handleKeydown);
    this.start();
  }

  override disconnectedCallback(): void {
    super.disconnectedCallback();
    document.removeEventListener('keydown', this.handleKeydown);
  }

  private handleKeydown = (e: KeyboardEvent): void => {
    if (this.header.isTyping() || !this.doc) {
      return;
    }
    const key = e.key.toLowerCase();
    if (e.ctrlKey || e.metaKey) {
      if (key === 'z' || key === 'y') {
        e.preventDefault();
        this.undoRedo(key === 'y' || e.shiftKey);
      }
      return;
    }
    if (key === 'escape') {
      this.setTool(0);
      return;
    }
    const index = this.tools.findIndex((tool) => tool.hotkey === key);
    if (index >= 0) {
      this.setTool(index);
    }
  };

  private async start(): Promise<void> {
    const resource_configs = await apiGet<RisqResourceConfig[]>('risq/resources');
    if (resource_configs.success) {
      this.session.setResourceConfigs(resource_configs.result);
    }
    const terrains = await listTerrains();
    if (terrains.success) {
      this.terrains = terrains.result;
    }
    await this.startBoard();
    await this.refreshMapList();
    this.header.setStatus('Select a map or create a new one');
  }

  private components(): CanvasComponent[] {
    return [this.header, this.right_panel, this.left_panel];
  }

  private canvasSize(): { w: number; h: number } {
    const rect = this.board.getBoundingRect();
    return { w: rect.width, h: rect.height };
  }

  private tool(): EditorTool {
    return this.tools[this.active_tool];
  }

  private setTool(index: number): void {
    if (index === this.active_tool) {
      return;
    }
    this.tool().deactivated();
    this.active_tool = index;
    this.right_panel.setActive(index);
    this.tool().activated();
  }

  private hoverTarget(): EditorTarget | undefined {
    const space = this.host.hover.space();
    if (space) {
      return { space: space.coordinate, zone: this.host.hover.zone()?.coordinate };
    }
    const game = this.session.getGame();
    if (!game) {
      return undefined;
    }
    const mouse = this.viewport.canvasToCoordinate(this.host.hover.mouseCanvas(), game.board_size);
    const axial = roundAxialCoordinate(mouse);
    return axialDistance(axial, { x: 0, y: 0 }) <= game.board_size ? { space: axial } : undefined;
  }

  private async startBoard(): Promise<void> {
    this.board_pixels = this.viewport.boardSize(INITIAL_BOARD_SIZE);
    const size = await this.board.initialize({
      board_size: this.board_pixels,
      max_scale: 1,
      fill_space: true,
      allow_side_move: true,
      draw: (ctx, transform) => this.draw(ctx, transform),
      scroll: (dy, mode, dx) => this.components().some((c) => c.scroll?.(dy, mode, dx)),
      draggingCallback: () => {
        this.dragged = true;
      },
      mousemove: (canvas, screen, transform) => {
        const over_ui = this.components()
          .map((c) => c.mousemove(canvas, screen, transform))
          .some(Boolean);
        this.host.hover.setMouse(canvas, screen);
        this.host.hover.update(over_ui || !this.doc);
        this.tool().mousemove(this.doc && !over_ui ? this.hoverTarget() : undefined);
      },
      mouseleave: () => {
        this.components().forEach((c) => c.setHovering(false));
        this.host.hover.leave();
      },
      mousedown: (e) => {
        this.dragged = false;
        const consumed = this.components()
          .map((c) => c.mousedown(e))
          .some(Boolean);
        return consumed || (!!this.doc && this.tool().mousedown(this.hoverTarget(), e));
      },
      mouseup: (e) => {
        this.components().forEach((c) => c.mouseup(e));
        if (this.doc) {
          this.tool().mouseup(this.hoverTarget(), e);
        }
      },
      zoom_config: { zoom_constant: 650, max_zoom: 1.3, min_zoom: 0.7 },
    });
    if (!size) {
      this.header.setStatus('Unable to initialize board', true);
      return;
    }
    this.board.addEventListener('canvas_resize', (e) => this.fitBoard(e.detail.board_size, e.detail.el_size));
  }

  private draw(ctx: CanvasRenderingContext2D, transform: Parameters<RisqBoardRenderer['draw']>[1]): void {
    const now = Date.now();
    const dt = now - this.last_frame;
    this.last_frame = now;
    if (this.doc) {
      this.renderer.draw(ctx, transform, this.board.getMaxScale());
      this.drawSelection(ctx);
      this.tool().drawOverlay(ctx, this.hoverTarget());
    }
    this.left_panel.draw(ctx, transform, dt);
    this.right_panel.draw(ctx, transform, dt);
    this.header.draw(ctx, transform, dt);
  }

  private drawSelection(ctx: CanvasRenderingContext2D): void {
    if (!this.selection) {
      return;
    }
    ctx.fillStyle = 'transparent';
    ctx.strokeStyle = 'rgb(255, 255, 255)';
    ctx.lineWidth = 3;
    drawHexagon(ctx, this.viewport.coordinateToCanvas(this.selection.space), this.viewport.hexR());
  }

  private select(target?: EditorTarget): void {
    this.selection = target;
    this.refreshSelection();
  }

  private refreshSelection(): void {
    const description = this.selection && this.describe(this.selection);
    if (description) {
      this.left_panel.show(description.title, description.lines);
    } else {
      this.selection = undefined;
      this.left_panel.hide();
    }
  }

  private describe(target: EditorTarget): { title: string; lines: string[] } | undefined {
    const space = this.session.spaceAt(target.space);
    if (!space) {
      return undefined;
    }
    const lines = [
      `Terrain: ${space.display_name} (#${space.terrain_id})`,
      `Type: ${terrainTypeLabel(space.terrain_type)}`,
    ];
    const zone = target.zone ? getRisqZone(space, target.zone) : undefined;
    if (!zone) {
      lines.push(`Resources: ${space.resources?.size ?? 0}`, `Buildings: ${space.buildings?.size ?? 0}`);
      lines.push(`Units: ${space.units?.size ?? 0}`);
      return { title: `Space (${space.coordinate.x}, ${space.coordinate.y})`, lines };
    }
    if (zone.terrain_override) {
      lines.push(`Zone terrain: ${zone.terrain_override_display_name ?? zone.terrain_override}`);
    }
    if (zone.resource) {
      lines.push(`Resource: ${zone.resource.display_name} (${Math.round(zone.resource.resources_left)})`);
    }
    if (zone.building) {
      lines.push(`Building: ${zone.building.display_name} (slot ${zone.building.player_id + 1})`);
    }
    for (const unit of zone.units.values()) {
      lines.push(`Unit: ${unit.display_name} (slot ${unit.player_id + 1})`);
    }
    return { title: `Zone (${zone.coordinate.x}, ${zone.coordinate.y})`, lines };
  }

  private patchTerrain(coordinates: Point2D[], terrain_id: number): void {
    const terrain = this.terrains.find((entry) => entry.terrain_id === terrain_id);
    for (const coordinate of coordinates) {
      const space = this.session.spaceAt(coordinate);
      if (space && terrain) {
        space.terrain_id = terrain_id;
        space.terrain_type = terrain.terrain_type;
        space.display_name = terrain.display_name;
      }
    }
  }

  private undoRedo(redo: boolean): void {
    if (this.doc && (redo ? this.history.redo(this.doc) : this.history.undo(this.doc))) {
      this.requestPreview();
    }
  }

  private async requestPreview(): Promise<void> {
    this.preview_revision++;
    if (this.preview_running) {
      this.preview_dirty = true;
      return;
    }
    this.preview_running = true;
    do {
      this.preview_dirty = false;
      await this.refreshPreview();
    } while (this.preview_dirty);
    this.preview_running = false;
  }

  private async refreshMapList(): Promise<void> {
    const response = await listMaps();
    if (!response.success) {
      this.header.setStatus(response.error_message, true);
      return;
    }
    this.header.setMaps(response.result);
  }

  private async createNew(name: string, size: number): Promise<void> {
    const error = this.newMapError(name, size);
    if (error) {
      this.header.setStatus(error, true);
      return;
    }
    await this.setDoc(newMapDoc(size, 1), name);
  }

  private newMapError(name: string, size: number): string | undefined {
    if (!MAP_NAME_PATTERN.test(name)) {
      return 'Enter a map name: lowercase letters, digits and underscores';
    }
    if (this.header.mapNames().includes(name)) {
      return `A map named ${name} already exists`;
    }
    if (!(size >= 0 && size <= MAX_BOARD_SIZE)) {
      return `Size must be 0 to ${MAX_BOARD_SIZE}`;
    }
    return undefined;
  }

  private async openMap(name: string): Promise<void> {
    const revision = ++this.load_revision;
    const response = await loadMap(name);
    if (revision !== this.load_revision) {
      return;
    }
    if (!response.success) {
      this.header.setStatus(response.error_message, true);
      return;
    }
    await this.setDoc(response.result, name);
  }

  private async save(name: string): Promise<void> {
    if (!this.doc) {
      this.header.setStatus('No map to save', true);
      return;
    }
    if (!MAP_NAME_PATTERN.test(name)) {
      this.header.setStatus('Enter a map name: lowercase letters, digits and underscores', true);
      return;
    }
    const response = await saveMap(name, this.doc);
    this.header.setStatus(response.success ? `Saved ${name}` : response.error_message, !response.success);
    if (response.success) {
      await this.refreshMapList();
      this.header.setSelected(name);
      this.header.showDelete(true);
    }
  }

  private async deleteSaved(name: string): Promise<void> {
    if (this.armed_delete !== name) {
      this.armed_delete = name;
      this.header.setStatus(`Press Delete again to permanently delete ${name}`, true);
      return;
    }
    this.armed_delete = undefined;
    const response = await deleteMap(name);
    if (!response.success) {
      this.header.setStatus(response.error_message, true);
      return;
    }
    this.doc = undefined;
    this.history.clear();
    this.select(undefined);
    this.host.hover.forget();
    this.header.setName('');
    this.header.showSave(false);
    this.header.showDelete(false);
    await this.refreshMapList();
    this.header.setStatus(`Deleted ${name}`);
  }

  private async setDoc(doc: MapDoc, name: string): Promise<void> {
    this.load_revision++;
    this.doc = doc;
    this.history.clear();
    this.select(undefined);
    this.header.setName(name);
    this.header.setSelected(this.header.mapNames().includes(name) ? name : undefined);
    this.header.showSave(true);
    this.header.showDelete(this.header.mapNames().includes(name));
    this.armed_delete = undefined;
    await this.refreshPreview();
  }

  private async refreshPreview(): Promise<void> {
    const doc = this.doc;
    if (!doc) {
      return;
    }
    const revision = ++this.preview_revision;
    const response = await previewMap(doc);
    if (doc !== this.doc || revision !== this.preview_revision) {
      return;
    }
    if (!response.success) {
      this.header.setStatus(response.error_message, true);
      return;
    }
    this.session.replaceSnapshot(response.result);
    this.host.hover.forget();
    this.refreshSelection();
    this.header.setStatus(`${doc.spaces.length} spaces`);
    this.fitBoard(this.board_pixels, this.board.getBoundingRect());
  }

  private fitBoard(board_size: Point2D, canvas_size: DOMRect): void {
    const game = this.session.getGame();
    if (!game) {
      return;
    }
    const { ratio, center } = this.viewport.resize(board_size, canvas_size, game.board_size);
    const zoom = this.viewport.zoomLimits(board_size);
    this.board.setMaxScale(zoom.max, zoom.min);
    this.board.scaleView(ratio);
    this.board.setOffset(center);
    for (const zone of game.spaces.flat().flatMap((space) => space?.zones?.flat() ?? [])) {
      zone.reset_hovered_data = true;
    }
    if (this.fitted_board_size !== game.board_size) {
      this.fitted_board_size = game.board_size;
      const scale = this.viewport.lastTransform().scale ?? 1;
      this.board.setView(multiplyPoint2D(scale, this.viewport.coordinateToCanvas({ x: 0, y: 0 })));
    }
  }
}

customElements.define('dwg-page-risq-map-editor', DwgPageRisqMapEditor);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-page-risq-map-editor': DwgPageRisqMapEditor;
  }
}
