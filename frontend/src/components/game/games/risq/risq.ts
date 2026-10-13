import { DwgElement } from '../../../dwg_element';
import type { UpdateMessage } from '../../data_models';
import type { CanvasComponent } from '../../util/canvas_components/canvas_component';
import type { DwgCanvasBoard } from '../../util/canvas_board/canvas_board';
import type { Point2D } from '../../util/objects2d';
import { addPoint2D, multiplyPoint2D } from '../../util/objects2d';
import type { DwgGame } from '../../game';
import { messageDialog } from '../../game';
import { apiGet } from '../../../../scripts/api';
import { createImage, resolveImage } from '../../../../scripts/image';
import { flushPerf, perfEvent, perfRecord, startPerf } from '../../../../scripts/perf';
import { createLock } from '../../../../scripts/util';
import { getSettings } from '../../../../scripts/settings_store';
import { err, log } from '../../../../scripts/log';
import { ColorRGB } from '../../../../scripts/color_rgb';

import html from './risq.html';
import type {
  GameRisq,
  RisqPlayer,
  RisqResourceConfig,
  RisqUnit,
  RisqBuilding,
  RisqSpace,
  RisqCost,
} from './model/types';
import type {
  AutoRenewSetData,
  BuildingBehaviorSetData,
  DefaultUnitBehaviorSetData,
  GameRisqFromServer,
  GatherPointSetData,
  StartTurnData,
  SubmittedOrdersData,
  UnitBehaviorSetData,
  UnsubmittedOrdersData,
} from './transport/snapshot_types';
import type { RisqTurnReport } from './transport/turn_report';
import { RisqCorpseLayout } from './rendering/zones/corpses';
import { PLAYER_ICON_SIZE, RisqImageCache } from './rendering/assets/image_cache';
import { RisqViewport } from './rendering/board/viewport';
import { RisqOrderOverlays } from './rendering/board/order_overlays';
import { RisqBoardRenderer } from './rendering/board/board_renderer';
import { zoneCenterOffset } from './rendering/zones/geometry';
import { RisqSession } from './application/session';
import {
  patchAutoRenew,
  patchBuildingBehavior,
  patchDefaultUnitBehavior,
  patchGatherPoint,
  patchUnitBehavior,
} from './application/setting_patches';
import { RisqCommands } from './application/commands';
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
import type { IdleCategory } from './application/orders/idle_category';
import { RisqControlGroupGrid } from './canvas_components/bottom_panel/control_group_grid';
import { RisqIdleGrid } from './canvas_components/bottom_panel/idle_grid';
import { RISQ_MESSAGE_WARNING_COLOR, RisqMessageQueue } from './canvas_components/message_queue';

import './risq.scss';
import '../../util/canvas_board/canvas_board';
import './dialogs/turn_report_dialog/turn_report_dialog';
import './dialogs/tech_tree_dialog/tech_tree_dialog';

const PANEL_BACKGROUND = 'rgb(222, 184, 135)';

/** Composition root: owns the collaborators, wires them to the canvas board, and applies server updates */
export class DwgRisq extends DwgElement implements HotkeyHost {
  private board!: DwgCanvasBoard;
  private abstract_game?: DwgGame;
  private outcome_shown = false;
  private last_frame_time = Date.now();
  private initialization_controller = new AbortController();
  private icons = new Map<string, HTMLImageElement>();
  private image_cache = new RisqImageCache();
  readonly corpse_layout = new RisqCorpseLayout();

  readonly session = new RisqSession();
  readonly viewport = new RisqViewport(this.session, () => this.pointer.recalculate());
  readonly orders_model = new RisqOrdersModel(() => this.refreshPanels());
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
  private bottom_panel = new RisqBottomPanel(this, { background: PANEL_BACKGROUND, center_item: this.minimap }, [
    [new RisqControlGroupGrid(this)],
    [this.minimap],
    [new RisqIdleGrid(this)],
  ]);
  private message_queue = new RisqMessageQueue(this);
  private readonly canvas_components: CanvasComponent[] = [
    this.right_panel,
    this.left_panel,
    this.bottom_panel,
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
    new RisqBoardClicks(this.session, this.viewport, this.selection, this.left_panel, this.planning)
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
    this.canvas_components
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
    super.disconnectedCallback();
    this.hotkeys.detach();
    flushPerf();
  }

  /** This will replace an existing icon */
  private createIcon(name: string): HTMLImageElement {
    const el = createImage(`/images/${name}.png`);
    el.alt = name;
    this.icons.set(name, el);
    return el;
  }

  getIcon(name: string): HTMLImageElement {
    return resolveImage(this.icons.get(name) ?? this.createIcon(name));
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
    this.outcome_shown = false;
    this.initialization_controller.abort();
    this.initialization_controller = new AbortController();
    const signal = this.initialization_controller.signal;
    this.session.setPlayerId(abstract_game.isPlayer() ? abstract_game.playerId() : -1);
    this.submission.turnStarted();
    abstract_game.setPadding('0px');
    const resource_configs = await apiGet<RisqResourceConfig[]>('risq/resources', signal);
    if (signal.aborted) {
      return;
    }
    if (resource_configs.success) {
      this.session.setResourceConfigs(resource_configs.result);
    } else {
      err('Unable to load resource configs', resource_configs.error_message);
    }
    startPerf();
    this.setNewGameData(game);
    this.board
      .initialize({
        board_size: this.viewport.boardSize(game.board_size),
        max_scale: 1,
        fill_space: true,
        allow_side_move: true,
        draw: (ctx, transform) => {
          const draw_start = performance.now();
          this.prepareFrame();
          this.renderer.draw(ctx, transform, this.board.getMaxScale());
          perfRecord('draw', performance.now() - draw_start);
        },
        scroll: (dy, mode) => this.pointer.scroll(dy, mode),
        mousemove: (m, screen, transform, modifiers) => this.pointer.mousemove(m, screen, transform, modifiers),
        draggingCallback: () => this.pointer.draggingCallback(),
        mouseleave: () => this.pointer.mouseleave(),
        cancelInput: () => this.pointer.cancelInput(),
        mousedown: (e) => this.pointer.mousedown(e),
        mouseup: (e) => this.pointer.mouseup(e),
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
        this.goToVillageCenter(abstract_game.isPlayer() ? this.session.getPlayerId() : 0);
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

  private prepareFrame(): void {
    const now = Date.now();
    const dt = now - this.last_frame_time;
    this.last_frame_time = now;
    perfRecord('frame_gap', dt, 40);
    if (this.right_panel.prepareFrame(dt)) {
      this.pointer.recalculate();
    }
  }

  private refreshPanels(): void {
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

  toggleIdleCategory(category: IdleCategory): void {
    this.planning.toggleIdleCategory(category);
    this.refreshPanels();
  }

  reviewLastTurnReport(): void {
    const player = this.getPlayer();
    const report = this.session.getLastTurnReport();
    if (!report) {
      this.showMessage('No previous turn report to review', RISQ_MESSAGE_WARNING_COLOR);
    } else if (player) {
      this.openTurnReportDialog(player, report);
    }
  }

  private openTurnReportDialog(player: RisqPlayer, report: RisqTurnReport): void {
    const dialog = document.createElement('dwg-risq-turn-report-dialog');
    dialog.setData({
      risq: this,
      player,
      report,
      close_hotkey: getSettings().risq_hotkeys.actions[RisqHotkeyAction.SUMMARY_REPORT],
    });
    this.appendChild(dialog);
  }

  openTechTree(): void {
    const dialog = document.createElement('dwg-risq-tech-tree-dialog');
    dialog.setData({ risq: this, close_hotkey: getSettings().risq_hotkeys.actions[RisqHotkeyAction.TECH_TREE] });
    this.appendChild(dialog);
  }

  private goToVillageCenter(player_id: number): void {
    if (player_id < 0) {
      return;
    }
    const player = this.getGame()?.players[player_id];
    const buildings = [...(player?.buildings.values() ?? [])];
    const target = buildings.find((b) => b.building_id === 1) ?? buildings[0] ?? [...(player?.units.values() ?? [])][0];
    if (!target) {
      return;
    }
    const location = 'building_id' in target ? target : this.session.unitLocation(target);
    if (location) {
      this.goToCoordinate(location.space_coordinate);
    }
  }

  goToCoordinate(coordinate: Point2D): void {
    const scale = this.viewport.lastTransform().scale ?? 1;
    this.board.setView(multiplyPoint2D(scale, this.viewport.coordinateToCanvas(coordinate)));
  }

  selectNextIdleUnit(): void {
    if (!this.session.givingOrders()) {
      return;
    }
    const entity = this.planning.nextIdleOrderable();
    if (entity) {
      this.selectIdleOrderable(entity);
    }
  }

  selectIdleOrderable(entity: RisqUnit | RisqBuilding): void {
    if ('building_id' in entity) {
      this.selection.selectBuilding(entity);
    } else {
      this.selection.selectUnit(entity);
    }
    const location = 'building_id' in entity ? entity : this.session.unitLocation(entity);
    if (!location) {
      return;
    }
    const view = addPoint2D(
      this.viewport.coordinateToCanvas(location.space_coordinate),
      zoneCenterOffset(location.zone_coordinate, this.viewport.hexR())
    );
    this.board.setView(multiplyPoint2D(this.viewport.lastTransform().scale, view));
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
      this.left_panel.resolveSize();
      this.left_panel.prepareLayout();
      this.minimap.resolveSize();
      this.right_panel.toggle(this.right_panel.isOpen());
      this.bottom_panel.dataRefreshed();
      this.mercenary_panel.dataRefreshed();
      this.default_behavior_panel.dataRefreshed();
      this.pointer.recalculate();
    });
  }

  async gameUpdate(update: UpdateMessage): Promise<void> {
    try {
      switch (update.kind) {
        case 'start-turn':
          this.commands.auto_renew.reset();
          this.commands.gather_points.reset();
          this.applyStartTurn(update.content as StartTurnData);
          break;
        case 'submitted-orders':
          this.applySubmittedOrders(update.content as SubmittedOrdersData);
          break;
        case 'unsubmitted-orders':
          this.applyUnsubmittedOrders(update.content as UnsubmittedOrdersData);
          break;
        case 'unit-behavior-set':
          this.applySetting(() => patchUnitBehavior(this.session, update.content as UnitBehaviorSetData));
          break;
        case 'building-behavior-set':
          this.applySetting(() => patchBuildingBehavior(this.session, update.content as BuildingBehaviorSetData));
          break;
        case 'default-unit-behavior-set':
        case 'default-unit-stance-set':
          this.applySetting(() => patchDefaultUnitBehavior(this.session, update.content as DefaultUnitBehaviorSetData));
          break;
        case 'gather-point-set':
          this.applySetting(() => patchGatherPoint(this.session, update.content as GatherPointSetData));
          this.commands.gather_points.acknowledge((update.content as GatherPointSetData).building_id);
          break;
        case 'set-gather-point-failed':
          this.commands.gather_points.acknowledge(undefined, false);
          this.showMessage((update.content as { message: string }).message, RISQ_MESSAGE_WARNING_COLOR);
          break;
        case 'auto-renew-set':
          this.applySetting(() => patchAutoRenew(this.session, update.content as AutoRenewSetData));
          this.commands.auto_renew.acknowledge();
          break;
        case 'set-auto-renew-failed':
          this.commands.auto_renew.acknowledge(false);
          this.showMessage((update.content as { message: string }).message, RISQ_MESSAGE_WARNING_COLOR);
          this.refreshPanels();
          break;
        case 'set-default-unit-behavior-failed':
          this.showMessage((update.content as { message: string }).message, RISQ_MESSAGE_WARNING_COLOR);
          break;
        default:
          log(`Unknown game update type ${update.kind}`);
          break;
      }
    } catch (e) {
      err(`Error during game update ${JSON.stringify(update)}: ${e}`);
    }
  }

  private applySetting(patch: () => void): void {
    patch();
    this.refreshPanels();
  }

  private setNewGameData(new_game: GameRisqFromServer): void {
    const apply_start = performance.now();
    this.session.replaceSnapshot(new_game);
    const convert_ms = performance.now() - apply_start;
    if (!this.session.givingOrders() || this.getPlayer()?.orders_submitted) {
      this.commands.gather_points.reset();
    }
    const shell_game = this.abstract_game?.getGame();
    if (shell_game) {
      shell_game.game_base.game_ended = new_game.game_base.game_ended;
    }
    if (new_game.game_base.game_ended) {
      this.commands.auto_renew.reset();
      this.armed.disarmAll();
      this.orders_model.clearPending();
    }
    this.hover.forget();
    this.orders_model.setSubmitted(this.getPlayer()?.active_orders ?? []);
    this.session.setLastTurnReport(this.getPlayer()?.turn_report ?? this.session.getLastTurnReport());
    this.refreshPanels();
    this.showGameOutcome();
    this.commands.gather_points.reconcile();
    perfEvent('snapshot_apply', {
      turn: new_game.turn_number,
      convert_ms,
      total_ms: performance.now() - apply_start,
    });
  }

  private showGameOutcome(): void {
    const game = this.getGame();
    if (!game?.game_base.game_ended || !game.outcome || this.outcome_shown) {
      return;
    }
    this.outcome_shown = true;
    const winners = game.outcome.winner_player_ids.map(
      (id: number): string =>
        game.players.find((player: RisqPlayer): boolean => player.player.player_id === id)!.player.nickname
    );
    const message =
      winners.length === 0
        ? 'No players remaining'
        : `${winners.length > 1 ? 'The winners are: ' : 'The winner is: '}${winners.join(', ')}`;
    messageDialog.call(this, { message });
  }

  private applyStartTurn(data: StartTurnData): void {
    this.submission.turnStarted();
    this.armed.disarmAll();
    this.setNewGameData(data.game);
    const player = this.getPlayer();
    if (!data.game.game_base.game_ended && this.getPlayerId() > -1 && player?.turn_report) {
      this.openTurnReportDialog(player, player.turn_report);
    }
  }

  private applySubmittedOrders(data: SubmittedOrdersData): void {
    if (data.player_id === this.getPlayerId()) {
      this.submission.submitted();
    }
    this.setNewGameData(data.game);
  }

  private applyUnsubmittedOrders(data: UnsubmittedOrdersData): void {
    if (data.player_id === this.getPlayerId()) {
      this.submission.unsubmitted();
    }
    this.setNewGameData(data.game);
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
