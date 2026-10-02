import { DwgElement } from '../../../dwg_element';
import type { UpdateMessage } from '../../data_models';
import type { CanvasComponent } from '../../util/canvas_components/canvas_component';
import type { DwgCanvasBoard } from '../../util/canvas_board/canvas_board';
import type { Point2D } from '../../util/objects2d';
import { addPoint2D, multiplyPoint2D } from '../../util/objects2d';
import type { DwgGame } from '../../game';
import { createLock } from '../../../../scripts/util';
import { err, log } from '../../../../scripts/log';
import { ColorRGB } from '../../../../scripts/color_rgb';

import html from './risq.html';
import type { GameRisq, RisqPlayer } from './model/types';
import type {
  GameRisqFromServer,
  GatherPointSetData,
  StartTurnData,
  SubmittedOrdersData,
  UnitBehaviorSetData,
  UnsubmittedOrdersData,
} from './transport/snapshot_types';
import type { RisqTurnReport } from './transport/turn_report';
import { PLAYER_ICON_SIZE, RisqImageCache } from './rendering/assets/image_cache';
import { RisqViewport } from './rendering/board/viewport';
import { RisqOrderPaths } from './rendering/board/order_paths';
import { RisqBoardRenderer } from './rendering/board/board_renderer';
import { zoneCenterOffset } from './rendering/zones/geometry';
import { RisqSession } from './application/session';
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
import { RisqHotkeyController } from './application/input/hotkey_controller';
import { RisqRightPanel } from './canvas_components/right_panel/right_panel';
import { RisqLeftPanel } from './canvas_components/left_panel/left_panel';
import { RisqMinimap } from './canvas_components/minimap/risq_minimap';
import { RisqBottomPanel } from './canvas_components/bottom_panel/bottom_panel';
import {
  RisqSummaryReportButton,
  RisqTechTreeButton,
  RisqViewModeButton,
} from './canvas_components/bottom_panel/bottom_panel_buttons';
import { RisqMercenaryGrid } from './canvas_components/bottom_panel/mercenary_grid';
import { RISQ_MESSAGE_WARNING_COLOR, RisqMessageQueue } from './canvas_components/message_queue';

import './risq.scss';
import '../../util/canvas_board/canvas_board';
import './dialogs/turn_report_dialog/turn_report_dialog';
import './dialogs/tech_tree_dialog/tech_tree_dialog';

const PANEL_BACKGROUND = 'rgb(222, 184, 135)';

/** Composition root: owns the collaborators, wires them to the canvas board, and applies server updates */
export class DwgRisq extends DwgElement implements HotkeyHost {
  private board!: DwgCanvasBoard;
  private icons = new Map<string, HTMLImageElement>();
  private image_cache = new RisqImageCache();

  readonly session = new RisqSession();
  readonly viewport = new RisqViewport(this.session);
  readonly orders_model = new RisqOrdersModel(() => this.refreshPanels());
  readonly planning = new RisqOrderPlanning(this.session, this.orders_model);
  readonly armed = new RisqArmedState(() => this.cursor.update(false));
  readonly hover = new RisqHover(this.session, this.viewport);

  readonly left_panel = new RisqLeftPanel(this, { w: 300, background: PANEL_BACKGROUND });
  readonly right_panel = new RisqRightPanel(this, { w: 300, is_open: true, background: new ColorRGB(222, 184, 135) });
  private minimap = new RisqMinimap(this, { target_w: 150, background: 'black' });
  private bottom_panel = new RisqBottomPanel(
    this,
    { background: PANEL_BACKGROUND, left_panel_w: 300, right_panel_w: 300 },
    [
      [new RisqMercenaryGrid(this, 32)],
      [new RisqSummaryReportButton(this, 32), new RisqViewModeButton(this, 32), new RisqTechTreeButton(this, 32)],
      [this.minimap],
    ]
  );
  private message_queue = new RisqMessageQueue(this);
  private readonly canvas_components: CanvasComponent[] = [
    this.right_panel,
    this.left_panel,
    this.bottom_panel,
    this.message_queue,
  ];

  readonly selection = new RisqSelection(this.session, this.left_panel);
  private control_groups = new RisqControlGroups(this.session, this.left_panel, this.selection);
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
  readonly submission = new RisqOrderSubmission(this, this.session, this.orders_model, this.planning, this.right_panel);
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
    () => this.left_panel.isHovering() || this.right_panel.isHovering() || this.bottom_panel.isHovering(),
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
    this.armed,
    this.left_panel,
    this.selection,
    this.control_groups,
    this.orders_model,
    this.commands,
    this.submission
  );
  private renderer = new RisqBoardRenderer(
    this,
    this.session,
    this.viewport,
    new RisqOrderPaths(this, this.session, this.viewport, this.orders_model, this.planning, this.selection),
    () => this.pointer.dragRect(),
    this.canvas_components
  );

  constructor() {
    super();
    this.html_string = html;
    this.configureElement('board');
  }

  override disconnectedCallback(): void {
    super.disconnectedCallback();
    this.hotkeys.detach();
  }

  /** This will replace an existing icon */
  private createIcon(name: string): HTMLImageElement {
    const el = document.createElement('img');
    el.src = `/images/${name}.png`;
    el.draggable = false;
    el.alt = name;
    this.icons.set(name, el);
    return el;
  }

  getIcon(name: string): HTMLImageElement {
    return this.icons.get(name) ?? this.createIcon(name);
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

  getPlayerId(): number {
    return this.session.getPlayerId();
  }

  showMessage(text: string, color: ColorRGB) {
    this.message_queue.enqueue(text, color);
  }

  async initialize(abstract_game: DwgGame, game: GameRisqFromServer): Promise<void> {
    this.session.setPlayerId(abstract_game.isPlayer() ? abstract_game.playerId() : -1);
    abstract_game.setPadding('0px');
    this.setNewGameData(game);
    this.hotkeys.attach();
    this.board
      .initialize({
        board_size: this.viewport.boardSize(game.board_size),
        max_scale: 1,
        fill_space: true,
        allow_side_move: true,
        draw: (ctx, transform) => this.renderer.draw(ctx, transform, this.board.getMaxScale()),
        scroll: (dy, mode) => this.pointer.scroll(dy, mode),
        mousemove: (m, screen, transform, modifiers) => this.pointer.mousemove(m, screen, transform, modifiers),
        draggingCallback: () => this.pointer.draggingCallback(),
        mouseleave: () => this.pointer.mouseleave(),
        mousedown: (e) => this.pointer.mousedown(e),
        mouseup: (e) => this.pointer.mouseup(e),
        extraMouseButton: (e) => this.hotkeys.extraMouseButton(e),
        zoom_config: {
          zoom_constant: 650,
          max_zoom: 1.3,
          min_zoom: 0.7,
        },
      })
      .then((size_data) => {
        if (!size_data) {
          err('Not able to initialize game board');
          return;
        }
        this.boardResize(size_data.board_size, size_data.el_size);
        this.goToVillageCenter(abstract_game.isPlayer() ? this.session.getPlayerId() : 0);
        this.board.addEventListener('canvas_resize', (e) => {
          this.boardResize(e.detail.board_size, e.detail.el_size);
        });
      });
  }

  private refreshPanels() {
    this.planning.refresh();
    this.right_panel.dataRefreshed();
    this.left_panel.dataRefreshed();
    this.pointer.recalculate();
  }

  reviewLastTurnReport() {
    const player = this.getPlayer();
    const report = this.session.getLastTurnReport();
    if (!report) {
      this.showMessage('No previous turn report to review', RISQ_MESSAGE_WARNING_COLOR);
    } else if (player) {
      this.openTurnReportDialog(player, report);
    }
  }

  private openTurnReportDialog(player: RisqPlayer, report: RisqTurnReport) {
    const dialog = document.createElement('dwg-risq-turn-report-dialog');
    dialog.setData({ risq: this, player, report });
    this.appendChild(dialog);
  }

  openTechTree() {
    const dialog = document.createElement('dwg-risq-tech-tree-dialog');
    dialog.setData({ risq: this });
    this.appendChild(dialog);
  }

  private goToVillageCenter(player_id: number) {
    const village_center = [...(this.getGame()?.players[player_id]?.buildings.values() ?? [])].find(
      (b) => b.building_id === 1
    );
    if (player_id >= 0 && village_center) {
      this.goToCoordinate(village_center.space_coordinate);
    }
  }

  goToCoordinate(coordinate: Point2D) {
    const scale = this.viewport.lastTransform().scale ?? 1;
    this.board.setView(multiplyPoint2D(scale, this.viewport.coordinateToCanvas(coordinate)));
  }

  selectNextIdleUnit() {
    const unit = this.planning.nextIdleUnit();
    if (!unit) {
      return;
    }
    this.selection.selectUnit(unit);
    const location = this.session.unitLocation(unit);
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
  private boardResize(board_size: Point2D, canvas_size: DOMRect) {
    this.board_resize_lock(async () => {
      const game = this.getGame();
      if (!game) {
        return;
      }
      const { ratio, center } = this.viewport.resize(board_size, canvas_size, game.board_size);
      this.board.setMaxScale((0.45 * canvas_size.height) / this.viewport.hexR());
      this.board.scaleView(ratio);
      this.board.setOffset(center);
      for (const zone of game.spaces.flat().flatMap((space) => space?.zones?.flat() ?? [])) {
        zone.reset_hovered_data = true;
      }
      this.left_panel.resolveSize();
      this.minimap.resolveSize();
      this.right_panel.toggle(this.right_panel.isOpen());
    });
  }

  async gameUpdate(update: UpdateMessage): Promise<void> {
    try {
      switch (update.kind) {
        case 'start-turn':
          this.applyStartTurn(update.content as StartTurnData);
          break;
        case 'submitted-orders':
          this.applySubmittedOrders(update.content as SubmittedOrdersData);
          break;
        case 'unsubmitted-orders':
          this.applyUnsubmittedOrders(update.content as UnsubmittedOrdersData);
          break;
        case 'unit-behavior-set':
          this.applyUnitBehaviorSet(update.content as UnitBehaviorSetData);
          break;
        case 'gather-point-set':
          this.applyGatherPointSet(update.content as GatherPointSetData);
          break;
        default:
          log(`Unknown game update type ${update.kind}`);
          break;
      }
    } catch (e) {
      err(`Error during game update ${JSON.stringify(update)}: ${e}`);
    }
  }

  private setNewGameData(new_game: GameRisqFromServer) {
    this.session.replaceSnapshot(new_game);
    this.hover.forget();
    this.orders_model.setSubmitted(this.getPlayer()?.active_orders ?? []);
    this.refreshPanels();
  }

  private applyStartTurn(data: StartTurnData) {
    this.submission.turnStarted();
    this.left_panel.close();
    this.armed.disarmOrder();
    this.setNewGameData(data.game);
    const player = this.getPlayer();
    this.session.setLastTurnReport(player?.turn_report);
    if (this.getPlayerId() > -1 && player?.turn_report) {
      this.openTurnReportDialog(player, player.turn_report);
    }
  }

  private applySubmittedOrders(data: SubmittedOrdersData) {
    if (data.player_id === this.getPlayerId()) {
      this.submission.submitted();
    }
    this.setNewGameData(data.game);
  }

  private applyUnsubmittedOrders(data: UnsubmittedOrdersData) {
    if (data.player_id === this.getPlayerId()) {
      this.submission.unsubmitted();
    }
    this.setNewGameData(data.game);
  }

  private applyUnitBehaviorSet(data: UnitBehaviorSetData) {
    for (const player of this.getGame()?.players ?? []) {
      for (const internal_id of data.internal_ids) {
        const unit = player.units.get(internal_id);
        if (!unit) {
          continue;
        }
        if (data.stance !== undefined) {
          unit.stance = data.stance;
        }
        if (data.interrupt_current !== undefined) {
          unit.interrupt_current = data.interrupt_current;
        }
        if (data.attack_back !== undefined) {
          unit.attack_back = data.attack_back;
        }
        if (data.target_priority !== undefined) {
          unit.target_priority = data.target_priority;
        }
      }
    }
    this.refreshPanels();
  }

  private applyGatherPointSet(data: GatherPointSetData) {
    for (const player of this.getGame()?.players ?? []) {
      const building = player.buildings.get(data.building_id);
      if (building) {
        building.gather_point = data.gather_point;
        break;
      }
    }
    this.refreshPanels();
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
