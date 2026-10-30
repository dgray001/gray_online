import { DwgElement } from '../../../dwg_element';
import type { UpdateMessage } from '../../data_models';
import type { CanvasComponent } from '../../util/canvas_components/canvas_component';
import type { DwgCanvasBoard } from '../../util/canvas_board/canvas_board';
import type { Point2D } from '../../util/objects2d';
import type { DwgGame } from '../../game';
import { apiGet } from '../../../../scripts/api';
import { flushPerf, perfMeasure, perfRecord, startPerf } from '../../../../scripts/perf';
import { createLock } from '../../../../scripts/util';
import { getSettings, onSettingsChanged } from '../../../../scripts/settings_store';
import { err } from '../../../../scripts/log';
import { ColorRGB } from '../../../../scripts/color_rgb';

import html from './risq.html';
import type { GameRisq, RisqPlayer, RisqResourceConfig, RisqSpace, RisqCost } from './model/types';
import type { GameRisqFromServer } from './transport/snapshot_types';
import { RisqCorpseLayout } from './rendering/zones/corpses';
import { PLAYER_ICON_SIZE, RisqImageCache } from './rendering/assets/image_cache';
import { RisqViewport } from './rendering/board/viewport';
import { RisqOrderOverlays } from './rendering/board/order_overlays';
import { RisqBoardRenderer } from './rendering/board/board_renderer';
import { RisqSession } from './application/session';
import { RisqCommands } from './application/commands';
import { RisqUpdates } from './application/updates';
import { RisqNavigation } from './application/navigation';
import { RisqOrdersModel } from './application/orders/orders_model';
import { RisqOrderPlanning } from './application/orders/planning';
import { RisqOrderTargeting } from './application/orders/targeting';
import { RisqOrderDispatch } from './application/orders/dispatch';
import { RisqOrderSubmission } from './application/orders/submission';
import { RisqSelection } from './application/selection/selection';
import { RisqControlGroups } from './application/selection/control_groups';
import { RisqBoardClicks } from './application/selection/board_clicks';
import { RisqArmedState } from './application/input/armed_state';
import { RisqHover } from './application/input/hover';
import { RisqCursorController } from './application/input/cursor_controller';
import { RisqPointer } from './application/input/pointer';
import type { HotkeyHost } from './application/input/hotkey_controller';
import { RisqHotkeyAction } from './application/input/hotkeys';
import { RisqHotkeyController } from './application/input/hotkey_controller';
import { RisqRightPanel } from './canvas_components/right_panel/right_panel';
import { RisqLeftPanel } from './canvas_components/left_panel/left_panel';
import { RisqMinimap } from './canvas_components/minimap/risq_minimap';
import { RisqBottomPanel } from './canvas_components/bottom_panel/bottom_panel';
import { RisqMercenaryPanel } from './canvas_components/bottom_panel/mercenary_panel';
import { RisqDefaultBehaviorGrid } from './canvas_components/bottom_panel/default_behavior_grid';
import type { IdleCategory } from './application/orders/orders_model';
import { RisqControlGroupGrid } from './canvas_components/bottom_panel/control_group_grid';
import { RisqIdleGrid } from './canvas_components/bottom_panel/idle_grid';
import { RISQ_MESSAGE_WARNING_COLOR, RisqMessageQueue } from './canvas_components/message_queue';
import { RisqReplayPanel } from './canvas_components/replay_panel/replay_panel';

import './risq.scss';
import '../../util/canvas_board/canvas_board';
import './dialogs/tech_tree_dialog/tech_tree_dialog';

const PANEL_BACKGROUND = 'rgb(222, 184, 135)';

/** Composition root: owns the collaborators, wires them to the canvas board, and applies server updates */
export class DwgRisq extends DwgElement implements HotkeyHost {
  board!: DwgCanvasBoard;
  abstract_game?: DwgGame;
  private last_frame_time = Date.now();
  private hud_scale = getSettings().risq_display.hud_size;
  private last_board_size?: Point2D;
  private unsubscribe_settings?: () => void;
  private initialization_controller = new AbortController();
  private image_cache = new RisqImageCache();
  readonly corpse_layout = new RisqCorpseLayout();

  readonly session = new RisqSession();
  readonly updates = new RisqUpdates(this);
  readonly navigation = new RisqNavigation(this);
  readonly viewport = new RisqViewport(this.session, () => this.pointer.recalculate());
  readonly orders_model = new RisqOrdersModel(
    () => this.refreshPanels(),
    () => this.showMessage('Duplicate order', RISQ_MESSAGE_WARNING_COLOR),
    (kind) => this.showMessage(`${kind === 'unit' ? 'Unit' : 'Building'} is being deleted`, RISQ_MESSAGE_WARNING_COLOR)
  );
  readonly planning = new RisqOrderPlanning(this.session, this.orders_model, (): RisqCost[] =>
    this.commands.auto_renew.reservations()
  );
  readonly armed = new RisqArmedState(() => this.cursor.update(this.board?.getModifiers().ctrl ?? false));
  readonly hover = new RisqHover(this.session, this.viewport);

  readonly left_panel = new RisqLeftPanel(this, { w: 300, background: PANEL_BACKGROUND });
  readonly right_panel = new RisqRightPanel(this, { w: 300, is_open: true, background: new ColorRGB(222, 184, 135) });
  readonly selection = new RisqSelection(this.session, this.left_panel);
  readonly control_groups = new RisqControlGroups(this.session, this.left_panel, this.selection, () =>
    this.refreshPanels()
  );
  private minimap = new RisqMinimap(
    this,
    { target_w: 150, background: 'black' },
    (space: RisqSpace, ctrl_held: boolean): void => {
      this.dispatch.spaceOrder(space, this.selection.subjectUnitIds(), this.selection.selectedBuildings(), ctrl_held);
    }
  );
  private mercenary_panel = new RisqMercenaryPanel(this, () => this.minimap);
  private default_behavior_panel = new RisqMercenaryPanel(this, () => this.minimap, new RisqDefaultBehaviorGrid(this));
  readonly bottom_panel = new RisqBottomPanel(this, { background: PANEL_BACKGROUND, center_item: this.minimap }, [
    [new RisqControlGroupGrid(this)],
    [this.minimap],
    [new RisqIdleGrid(this)],
  ]);
  readonly replay_panel = new RisqReplayPanel(this, () => this.bottom_panel);
  private message_queue = new RisqMessageQueue(this);
  private readonly canvas_components: CanvasComponent[] = [
    this.right_panel,
    this.left_panel,
    this.bottom_panel,
    this.replay_panel,
    this.mercenary_panel,
    this.default_behavior_panel,
    this.message_queue,
  ];

  private targeting = new RisqOrderTargeting(
    this.session,
    this.viewport,
    this.hover,
    this.armed,
    this.left_panel,
    this.planning
  );
  private cursor = new RisqCursorController(this, () => this.board, this.armed, this.targeting, this.selection);
  readonly commands = new RisqCommands(this, this.session, this.orders_model, this.planning, this.armed, (t, c) =>
    this.showMessage(t, c)
  );
  readonly submission = new RisqOrderSubmission(
    this,
    this.session,
    this.orders_model,
    this.planning,
    this.right_panel,
    this.commands.auto_renew,
    this.commands.gather_points
  );
  private dispatch = new RisqOrderDispatch(
    this.session,
    this.viewport,
    this.hover,
    this.armed,
    this.targeting,
    this.planning,
    this.orders_model,
    this.commands,
    (t, c) => this.showMessage(t, c)
  );
  readonly pointer = new RisqPointer(
    () => this.board,
    this.canvas_components,
    () =>
      this.left_panel.isHovering() ||
      this.right_panel.isHovering() ||
      this.bottom_panel.isHovering() ||
      this.replay_panel.isHovering() ||
      this.mercenary_panel.isHovering() ||
      this.default_behavior_panel.isHovering(),
    this.session,
    this.viewport,
    this.hover,
    this.armed,
    this.cursor,
    this.left_panel,
    this.planning,
    this.dispatch,
    new RisqBoardClicks(this.session, this.viewport, this.selection, this.left_panel, this.planning),
    () => this.hud_scale
  );
  private hotkeys = new RisqHotkeyController(
    this,
    this.session,
    this.viewport,
    this.left_panel,
    this.selection,
    this.control_groups,
    this.commands,
    this.submission
  );
  readonly order_overlays = new RisqOrderOverlays(
    this,
    this.session,
    this.viewport,
    this.orders_model,
    this.planning,
    this.selection
  );
  private renderer = new RisqBoardRenderer(
    this,
    this.session,
    this.viewport,
    this.order_overlays,
    () => this.pointer.dragRect(),
    this.canvas_components,
    () => this.hud_scale
  );

  constructor() {
    super();
    this.html_string = html;
    this.configureElement('board');
  }

  private handleDialogOpened = (): void => {
    this.armed.disarmAll();
  };

  override async connectedCallback(): Promise<void> {
    document.addEventListener('dialog_opened', this.handleDialogOpened);
    await super.connectedCallback();
  }

  override disconnectedCallback(): void {
    document.removeEventListener('dialog_opened', this.handleDialogOpened);
    this.commands.auto_renew.reset();
    this.commands.gather_points.reset();
    this.initialization_controller.abort();
    this.unsubscribe_settings?.();
    super.disconnectedCallback();
    this.hotkeys.detach();
    flushPerf();
  }

  getIcon(name: string): HTMLImageElement {
    return this.image_cache.getIcon(name);
  }

  /** Returns the unit/building icon at `name` with its color-key pixels swapped for the given player color */
  getPlayerColoredIcon(name: string, color: ColorRGB): HTMLImageElement | HTMLCanvasElement {
    const icon = this.getIcon(name);
    return this.image_cache.getPlayerColoredIcon(name, icon, PLAYER_ICON_SIZE, color) ?? icon;
  }

  getImageCache(): RisqImageCache {
    return this.image_cache;
  }

  getGame(): GameRisq | undefined {
    return this.session.getGame();
  }

  getPlayer(): RisqPlayer | undefined {
    return this.session.getPlayer();
  }

  toggleMercenaryPanel(): void {
    this.default_behavior_panel.toggle(false);
    this.mercenary_panel.toggle();
  }

  toggleDefaultBehaviorPanel(): void {
    this.mercenary_panel.toggle(false);
    this.default_behavior_panel.toggle();
  }

  closeBottomSubpanels(): void {
    this.mercenary_panel.toggle(false);
    this.default_behavior_panel.toggle(false);
  }

  getPlayerId(): number {
    return this.session.getPlayerId();
  }

  showMessage(text: string, color: ColorRGB): void {
    this.message_queue.enqueue(text, color);
  }

  async initialize(abstract_game: DwgGame, game: GameRisqFromServer): Promise<void> {
    this.commands.auto_renew.reset();
    this.commands.gather_points.reset();
    this.abstract_game = abstract_game;
    this.corpse_layout.reset();
    this.updates.reset();
    this.initialization_controller.abort();
    this.initialization_controller = new AbortController();
    const signal = this.initialization_controller.signal;
    this.unsubscribe_settings?.();
    this.unsubscribe_settings = onSettingsChanged((): void => this.settingsChanged());
    this.hud_scale = getSettings().risq_display.hud_size;
    this.session.setPlayerId(abstract_game.isPlayer() ? abstract_game.playerId() : -1);
    this.submission.turnStarted();
    abstract_game.setPadding('0px');
    const resource_configs = await apiGet<RisqResourceConfig[]>(`risq/resources/${game.game_base.game_id}`, signal);
    if (signal.aborted) {
      return;
    }
    if (resource_configs.success) {
      this.session.setResourceConfigs(resource_configs.result);
    } else {
      err('Unable to load resource configs', resource_configs.error_message);
    }
    startPerf();
    this.updates.setNewGameData(game);
    this.board
      .initialize({
        board_size: this.viewport.boardSize(game.board_size),
        max_scale: 1,
        fill_space: true,
        allow_edge_pan: getSettings().risq_display.edge_pan,
        draw: (ctx, transform) => {
          const draw_start = performance.now();
          this.prepareFrame();
          this.renderer.draw(ctx, transform, this.board.getMaxScale());
          perfRecord('draw', performance.now() - draw_start);
        },
        scroll: (dy, mode) => this.pointer.scroll(dy, mode),
        mousemove: (m, screen, transform, modifiers) =>
          perfMeasure('mousemove', () => this.pointer.mousemove(m, screen, transform, modifiers)),
        draggingCallback: () => this.pointer.draggingCallback(),
        mouseleave: () => this.pointer.mouseleave(),
        cancelInput: () => this.pointer.cancelInput(),
        mousedown: (e) => perfMeasure('mousedown', () => this.pointer.mousedown(e)),
        mouseup: (e) => perfMeasure('mouseup', () => this.pointer.mouseup(e)),
        extraMouseButton: (e) => this.hotkeys.extraMouseButton(e),
        zoom_config: {
          zoom_constant: 650,
          max_zoom: 1.3,
          min_zoom: 0.7,
        },
      })
      .then(async (size_data): Promise<void> => {
        if (signal.aborted) {
          return;
        }
        if (!size_data) {
          throw new Error('Unable to initialize the Risq board');
        }
        this.board.addEventListener('canvas_zoom', (e) => this.viewport.updateCenterSlots(e.detail), { signal });
        await this.boardResize(size_data.board_size, size_data.el_size);
        if (signal.aborted) {
          return;
        }
        this.board.setScale(this.viewport.zoneDetailScale() * 1.01);
        this.viewport.setTransform(this.board.getTransform());
        this.navigation.goToVillageCenter(abstract_game.isPlayer() ? this.session.getPlayerId() : 0);
        this.hotkeys.attach();
        this.board.addEventListener('canvas_resize', (e) => this.boardResize(e.detail.board_size, e.detail.el_size), {
          signal,
        });
      })
      .catch((error: unknown): void => {
        if (!signal.aborted) {
          this.dispatchEvent(new CustomEvent<unknown>('game_initialization_failed', { detail: error, bubbles: true }));
        }
      });
  }

  hudScale(): number {
    return this.hud_scale;
  }

  /** Canvas size as the HUD sees it: the HUD is drawn scaled, so it lays out against the real size divided by that */
  hudCanvasSize(): { width: number; height: number } {
    const size = this.viewport.canvasSize();
    return { width: size.width / this.hud_scale, height: size.height / this.hud_scale };
  }

  private settingsChanged(): void {
    this.board.setAllowEdgePan(getSettings().risq_display.edge_pan);
    this.hud_scale = getSettings().risq_display.hud_size;
    let scale = this.viewport.lastTransform().scale;
    if (this.board.isInitialized() && this.last_board_size) {
      const zoom = this.viewport.zoomLimits(this.last_board_size);
      this.board.setMaxScale(zoom.max, zoom.min); // rescales the zoom with the limit, so put it back (clamped)
      scale = this.board.setScale(scale);
    }
    this.viewport.updateCenterSlots(scale);
    this.relayoutHud();
    this.refreshPanels();
  }

  private prepareFrame(): void {
    const now = Date.now();
    const dt = now - this.last_frame_time;
    this.last_frame_time = now;
    perfRecord('frame_gap', dt, 40);
    const right_moved = this.right_panel.prepareFrame(dt);
    const bottom_moved = this.bottom_panel.prepareFrame(dt);
    if (right_moved || bottom_moved) {
      this.pointer.recalculate();
    }
  }

  refreshPanels(): void {
    const refresh_start = performance.now();
    this.refreshPanelsUntimed();
    perfRecord('refresh_panels', performance.now() - refresh_start);
  }

  private refreshPanelsUntimed(): void {
    this.planning.refresh();
    this.right_panel.dataRefreshed();
    this.left_panel.dataRefreshed();
    this.bottom_panel.dataRefreshed();
    this.mercenary_panel.dataRefreshed();
    this.default_behavior_panel.dataRefreshed();
    this.pointer.recalculate();
  }

  toggleReplay(): void {
    const journal = this.getGame()?.last_turn_journal;
    const replaying = !this.session.isReplaying() && !!journal;
    this.session.setReplaying(replaying);
    this.armed.disarmAll();
    this.hover.forget();
    if (replaying) {
      this.replay_panel.open(journal.tick_count);
      this.showReplayTick(0);
    } else {
      this.replay_panel.close();
      this.session.endReplay();
      this.hover.forget();
      this.refreshPanels();
    }
    this.showMessage(replaying ? 'Replay mode started' : 'Replay mode exited', RISQ_MESSAGE_WARNING_COLOR);
  }

  showReplayTick(value: number): void {
    this.session.showReplayTick(value);
    this.hover.forget();
    this.refreshPanels();
  }

  toggleIdleCategory(category: IdleCategory): void {
    this.planning.toggleIdleCategory(category);
    this.refreshPanels();
  }

  getGameId(): number {
    return this.abstract_game?.getGame()?.game_base.game_id ?? 0;
  }

  openTechTree(): void {
    const dialog = document.createElement('dwg-risq-tech-tree-dialog');
    dialog.setData({ risq: this, close_hotkey: getSettings().risq_hotkeys.actions[RisqHotkeyAction.TECH_TREE] });
    this.appendChild(dialog);
  }

  private board_resize_lock = createLock();
  private async boardResize(board_size: Point2D, canvas_size: DOMRect): Promise<void> {
    await this.board_resize_lock(async () => {
      const game = this.getGame();
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
      this.last_board_size = board_size;
      this.relayoutHud();
    });
  }

  private relayoutHud(): void {
    this.left_panel.resolveSize();
    this.left_panel.prepareLayout();
    this.minimap.resolveSize();
    this.right_panel.toggle(this.right_panel.isOpen());
    this.bottom_panel.dataRefreshed();
    this.mercenary_panel.dataRefreshed();
    this.default_behavior_panel.dataRefreshed();
    this.pointer.recalculate();
  }

  async gameUpdate(update: UpdateMessage): Promise<void> {
    await this.updates.gameUpdate(update);
  }

  reviewLastTurnReport(): void {
    this.updates.reviewLastTurnReport();
  }

  goToCoordinate(coordinate: Point2D): void {
    this.navigation.goToCoordinate(coordinate);
  }

  selectNextIdleUnit(): void {
    this.navigation.selectNextIdleUnit();
  }

  updateDialogComponent(update: UpdateMessage): HTMLElement {
    const update_el = document.createElement('div');
    update_el.innerText = `ID: ${update.update_id}, Kind: ${update.kind}, data: ${JSON.stringify(update.content)}`;
    return update_el;
  }
}

customElements.define('dwg-risq', DwgRisq);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-risq': DwgRisq;
  }
}
