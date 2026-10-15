import { DwgElement } from '../dwg_element';
import { apiGet } from '../../scripts/api';
import { isDialogOpen } from '../../scripts/util';
import { DialogSize } from '../dialog_box/dialog_box';
import type { DwgCanvasBoard } from '../game/util/canvas_board/canvas_board';
import type { CanvasComponent } from '../game/util/canvas_components/canvas_component';
import { drawHexagon } from '../game/util/canvas_util';
import type { Point2D } from '../game/util/objects2d';
import { axialDistance, multiplyPoint2D, roundAxialCoordinate } from '../game/util/objects2d';
import { RisqSession } from '../game/games/risq/application/session';
import type { RisqResourceConfig } from '../game/games/risq/model/types';
import { RisqViewport } from '../game/games/risq/rendering/board/viewport';
import { RisqBoardRenderer } from '../game/games/risq/rendering/board/board_renderer';
import { EditHistory } from './edit_history';
import { describeTarget, EditorLeftPanel } from './editor_left_panel';
import { EditorRightPanel } from './editor_right_panel';
import type { EditorTarget, EditorToolContext } from './editor_tool';
import { createEditorTools, type EditorTool } from './editor_tool';
import { EditorDrawHost, fitBoardView, patchTerrainSpaces } from './editor_host';
import { MapEditorHeader } from './map_editor_header';
import type { EntityCatalog, TerrainEntry } from './map_api';
import { deleteMap, listEntities, listMaps, listTerrains, loadMap, previewMap, saveMap } from './map_api';
import type { MapDoc } from './map_doc';
import { MAP_NAME_PATTERN, newMapDoc, newMapError } from './map_doc';

import html from './page_risq_map_editor.html';

import './page_risq_map_editor.scss';
import '../game/util/canvas_board/canvas_board';
import '../dialog_box/confirm_dialog/confirm_dialog';

const INITIAL_BOARD_SIZE = 2;

export class DwgPageRisqMapEditor extends DwgElement {
  private board!: DwgCanvasBoard;

  private session = new RisqSession();
  private viewport = new RisqViewport(this.session);
  private host = new EditorDrawHost(this.session, this.viewport);
  private history = new EditHistory();
  private catalog_config_dir?: string;
  private terrains: TerrainEntry[] = [];
  private entities: EntityCatalog = { buildings: [], units: [], resources: [] };
  private selection?: EditorTarget;
  private dragged = false;
  private context: EditorToolContext = {
    history: this.history,
    doc: () => this.doc,
    edited: () => this.requestPreview(),
    patchTerrain: (coordinates, terrain_id) => patchTerrainSpaces(this.session, this.terrains, coordinates, terrain_id),
    select: (target) => this.select(target),
    dragged: () => this.dragged,
    terrains: () => this.terrains,
    entities: () => this.entities,
    panTo: (coordinate) => this.panTo(coordinate),
    getIcon: (name) => this.host.getIcon(name),
    toCanvas: (coordinate) => this.viewport.coordinateToCanvas(coordinate),
    hexR: () => this.viewport.hexR(),
  };
  private tools: EditorTool[] = createEditorTools(this.context);
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
    { draw: () => {}, drawRanges: (): void => {} },
    () => undefined,
    []
  );
  private doc?: MapDoc;
  private board_pixels: Point2D = { x: 0, y: 0 };
  private fitted_board_size = -1;
  private last_frame = Date.now();
  private preview_running = false;
  private preview_dirty = false;
  private preview_revision = 0;
  private load_revision = 0;
  private saved_name?: string;
  private unsaved_new = false;
  private revert_notice?: string;

  constructor() {
    super();
    this.html_string = html;
    this.configureElement('board');
  }

  protected override parsedCallback(): void {
    document.addEventListener('keydown', this.handleKeydown);
    window.addEventListener('beforeunload', this.handleBeforeUnload);
    this.start();
  }

  override disconnectedCallback(): void {
    super.disconnectedCallback();
    document.removeEventListener('keydown', this.handleKeydown);
    window.removeEventListener('beforeunload', this.handleBeforeUnload);
  }

  private handleBeforeUnload = (e: BeforeUnloadEvent): void => {
    if (this.isDirty()) {
      e.preventDefault();
    }
  };

  private isTyping(): boolean {
    return this.header.isBusy() || isDialogOpen() || !!this.tool().config()?.isTyping?.();
  }

  private handleKeydown = (e: KeyboardEvent): void => {
    const key = e.key.toLowerCase();
    const modifier = e.ctrlKey || e.metaKey;
    if (modifier && key === 's') {
      e.preventDefault();
      this.save(this.header.name());
      return;
    }
    if (this.isTyping() || !this.doc) {
      return;
    }
    if (modifier) {
      if (key === 'z' || key === 'y') {
        e.preventDefault();
        this.undoRedo(key === 'y' || e.shiftKey);
      }
      return;
    }
    if (e.altKey || e.repeat) {
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
    await this.loadCatalogs(undefined);
    await this.startBoard();
    await this.refreshMapList();
    this.header.setStatus('Select a map or create a new one');
  }

  private async loadCatalogs(config_dir: string | undefined): Promise<void> {
    this.catalog_config_dir = config_dir;
    const terrains = await listTerrains(config_dir);
    if (terrains.success) {
      this.terrains = terrains.result;
    }
    const entities = await listEntities(config_dir);
    if (entities.success) {
      this.entities = entities.result;
    }
  }

  private isDirty(): boolean {
    return !!this.doc && (this.unsaved_new || this.history.isDirty());
  }

  private confirm(question: string, on_confirmed: () => void): void {
    const dialog = document.createElement('dwg-confirm-dialog');
    dialog.setData({ question, size: DialogSize.SMALL });
    dialog.addEventListener('confirmed', on_confirmed);
    this.appendChild(dialog);
  }

  private discardThen(action: () => void): void {
    if (this.isDirty()) {
      this.confirm('Discard unsaved changes?', action);
    } else {
      action();
    }
  }

  private panTo(coordinate: Point2D): void {
    const scale = this.viewport.lastTransform().scale ?? 1;
    this.board.setView(multiplyPoint2D(scale, this.viewport.coordinateToCanvas(coordinate)));
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
    this.header.setDirty(this.isDirty());
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
    const description = this.selection && describeTarget(this.session, this.selection);
    if (description) {
      this.left_panel.show(description.title, description.lines);
    } else {
      this.selection = undefined;
      this.left_panel.hide();
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

  private createNew(name: string, size: number): void {
    const error = newMapError(name, size, this.header.mapNames());
    if (error) {
      this.header.setStatus(error, true);
      return;
    }
    this.discardThen(() => this.startNewMap(name, size));
  }

  private async startNewMap(name: string, size: number): Promise<void> {
    await this.setDoc(newMapDoc(size, 1), name);
    this.unsaved_new = true;
  }

  private openMap(name: string): void {
    if (this.isDirty()) {
      this.header.setSelected(this.saved_name);
    }
    this.discardThen(() => this.loadSavedMap(name));
  }

  private async loadSavedMap(name: string): Promise<void> {
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
      this.history.markSaved();
      this.unsaved_new = false;
      this.saved_name = name;
      await this.refreshMapList();
      this.header.setSelected(name);
      this.header.showDelete(true);
    }
  }

  private deleteSaved(name: string): void {
    this.confirm(`Permanently delete ${name}?`, () => this.deleteConfirmed(name));
  }

  private async deleteConfirmed(name: string): Promise<void> {
    const response = await deleteMap(name);
    if (!response.success) {
      this.header.setStatus(response.error_message, true);
      return;
    }
    this.doc = undefined;
    this.saved_name = undefined;
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
    if (doc.config_dir !== this.catalog_config_dir) {
      await this.loadCatalogs(doc.config_dir);
    }
    this.history.clear();
    this.select(undefined);
    this.header.setName(name);
    this.header.setSelected(this.header.mapNames().includes(name) ? name : undefined);
    this.header.showSave(true);
    this.header.showDelete(this.header.mapNames().includes(name));
    this.unsaved_new = false;
    this.saved_name = this.header.mapNames().includes(name) ? name : undefined;
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
      const reverted = this.history.undo(doc);
      this.revert_notice = reverted ? `Edit reverted: ${response.error_message}` : undefined;
      this.header.setStatus(this.revert_notice ?? response.error_message, true);
      if (reverted) {
        this.requestPreview();
      }
      return;
    }
    this.session.replaceSnapshot(response.result);
    this.board.setDefaultRotation(this.session.getGame()?.rotation ?? 0);
    this.host.hover.forget();
    this.refreshSelection();
    this.header.setStatus(this.revert_notice ?? `${doc.spaces.length} spaces`, !!this.revert_notice);
    this.revert_notice = undefined;
    this.fitBoard(this.board_pixels, this.board.getBoundingRect());
  }

  private fitBoard(board_size: Point2D, canvas_size: DOMRect): void {
    this.fitted_board_size = fitBoardView(
      this.session,
      this.viewport,
      this.board,
      board_size,
      canvas_size,
      this.fitted_board_size
    );
  }
}

customElements.define('dwg-page-risq-map-editor', DwgPageRisqMapEditor);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-page-risq-map-editor': DwgPageRisqMapEditor;
  }
}
