import { DwgElement } from '../../../dwg_element';
import type { UpdateMessage } from '../../data_models';
import { drawArrow, drawCircle, drawRect } from '../../util/canvas_util';
import type { CanvasComponent } from '../../util/canvas_components/canvas_component';
import { configDraw } from '../../util/canvas_components/canvas_component';
import type { BoardTransformData, DwgCanvasBoard, ModifierKeys } from '../../util/canvas_board/canvas_board';
import { canvasToScreen, screenToCanvas } from '../../util/canvas_board/canvas_board';
import type { Point2D } from '../../util/objects2d';
import {
  addPoint2D,
  equalsPoint2D,
  hexagonalBoardNeighbors,
  hexagonalBoardRows,
  multiplyPoint2D,
  normalizeRect,
  pointInRect,
  roundAxialCoordinate,
  subtractPoint2D,
} from '../../util/objects2d';
import type { DwgGame } from '../../game';
import { DEV, createLock, isTypingInInput } from '../../../../scripts/util';
import { ColorRGB } from '../../../../scripts/color_rgb';

import html from './risq.html';
import type {
  GameRisq,
  GameRisqFromServer,
  RisqBuilding,
  RisqFrontendOrder,
  RisqPlayer,
  RisqSpace,
  RisqUnit,
  RisqZone,
  UnitByTypeData,
} from './risq_data';
import {
  RisqAttackType,
  RisqGatherObjectType,
  RisqGatherPointLocationKind,
  RisqOrderType,
  RisqProducibleKind,
  RisqResourceType,
  RisqVisibilityLevel,
  canAffordCost,
  canHaveGatherPoint,
  serverToGameRisq,
} from './risq_data';
import type { RisqGatherPoint, RisqUnitStance } from './risq_data';
import { cantorPair, coordinateToIndex, getSpace, invertBuildKey, invertPair, invertZoneKey } from './risq_coordinates';
import type {
  GatherPointSetData,
  StartTurnData,
  SubmittedOrdersData,
  UnitBehaviorSetData,
  UnsubmittedOrdersData,
} from './risq_updates';
import { cursorImageForOrderType, DEFAULT_CURSOR_IMAGE, resolveBuildCursorUrl } from './risq_cursor';
import { PLAYER_ICON_SIZE, RisqImageCache } from './risq_image_cache';
import { RisqRightPanel } from './canvas_components/right_panel/right_panel';
import type { DrawRisqSpaceConfig } from './risq_space';
import { DrawRisqSpaceDetail, drawRisqSpace, drawRisqSpaceBorder } from './risq_space';
import { RisqLeftPanel } from './canvas_components/left_panel/left_panel';
import type { UnitToggleField } from './canvas_components/left_panel/action_button/unit_toggle_button';
import { RisqMinimap } from './canvas_components/minimap/risq_minimap';
import { RISQ_MESSAGE_WARNING_COLOR, RisqMessageQueue } from './canvas_components/message_queue';
import { RisqOrdersModel, isBuildingOrder, isUnitOrder, orderArrowColor } from './risq_orders';
import {
  UNIT_SLOT_CIRCLE_RADIUS_MULTIPLIER,
  getRisqZone,
  groupUnitsByType,
  hoveredZoneObject,
  resolveHoveredZones,
  unhoverRisqZone,
  unitSlotWorldPosition,
  zoneApproachPoint,
  zoneCenterOffset,
} from './risq_zone';
import { RisqViewMode, nextViewMode } from './risq_terrain';
import type {
  BuildingData,
  EconomicUnitsData,
  MilitaryUnitsData,
  UnitData,
  UnitsByTypeData,
} from './canvas_components/left_panel/left_panel_data';
import { LeftPanelDataType } from './canvas_components/left_panel/left_panel_data';

import './risq.scss';
import '../../util/canvas_board/canvas_board';
import '../../../dialog_box/confirm_dialog/confirm_dialog';
import './turn_report_dialog/turn_report_dialog';
import { DialogSize } from '../../../dialog_box/dialog_box';
import { createMessage } from '../../../lobby/data_models';

const DEFAULT_HEXAGON_RADIUS = 60;

const DRAW_CENTER_DOT = false;

const DRAG_SELECT_THRESHOLD = 3;

export declare interface LocalRisqFoundation {
  coordinate_key: number;
  building_id: number;
  display_name: string;
  order: RisqFrontendOrder;
}

export class DwgRisq extends DwgElement {
  private board!: DwgCanvasBoard;

  private game?: GameRisq;
  private player_id: number = -1;
  private hex_r = DEFAULT_HEXAGON_RADIUS;
  private hex_a = 0.5 * 1.732 * DEFAULT_HEXAGON_RADIUS;
  private canvas_center: Point2D = { x: 0, y: 0 };
  private last_transform: BoardTransformData = {
    view: { x: 0, y: 0 },
    offset: { x: 0, y: 0 },
    rotation: 0,
    scale: 1,
  };
  private canvas_size: DOMRect = DOMRect.fromRect();
  private mouse_canvas: Point2D = { x: 0, y: 0 };
  private mouse_screen: Point2D = { x: 0, y: 0 };
  private mouse_coordinate: Point2D = { x: 0, y: 0 };
  private dragging_selection = false;
  private drag_additive = false;
  private drag_start: Point2D = { x: 0, y: 0 };
  private drag_current: Point2D = { x: 0, y: 0 };
  private hovered_space?: RisqSpace;
  private hovered_zone?: RisqZone;
  private icons = new Map<string, HTMLImageElement>();
  private image_cache = new RisqImageCache();
  private last_time = Date.now();
  private draw_detail: DrawRisqSpaceDetail = DrawRisqSpaceDetail.SPACE_DETAILS;
  private view_mode: RisqViewMode = RisqViewMode.ALL;
  private toggling_submit_orders_button = false;
  private orders_submitted_times = 0;
  private armed_order = RisqOrderType.NONE;
  private armed_building?: { id: number; display_name: string };
  private armed_button_callback?: () => void;
  private gather_point_armed = false;
  private building_attack_armed = false;
  private local_foundations = new Map<number, LocalRisqFoundation>();
  // control groups 1-10 ('0' is group 10)
  private control_groups = new Map<number, { kind: 'unit' | 'building'; ids: number[] }>();
  private orders_model = new RisqOrdersModel(() => this.ordersChanged());
  private idle_units: RisqUnit[] = [];
  private last_idle_selected?: number;

  private handleKeydown = (e: KeyboardEvent) => {
    if (isTypingInInput()) {
      return;
    }
    if (/^[0-9]$/.test(e.key)) {
      const group = e.key === '0' ? 10 : parseInt(e.key, 10);
      if (e.ctrlKey) {
        this.assignControlGroup(group);
      } else {
        this.selectControlGroup(group);
      }
      return;
    }
    switch (e.key.toLowerCase()) {
      case 'v':
        this.view_mode = nextViewMode(this.view_mode);
        break;
      default:
        break;
    }
  };

  private left_panel = new RisqLeftPanel(this, {
    w: 300,
    background: 'rgb(222,184,135)',
  });
  private right_panel = new RisqRightPanel(this, {
    w: 300,
    is_open: true,
    background: new ColorRGB(222, 184, 135),
  });
  private minimap = new RisqMinimap(this, {
    target_w: 150,
    background: 'rgb(222,184,135)',
  });
  private message_queue = new RisqMessageQueue(this);
  private readonly canvas_components: CanvasComponent[] = [
    this.right_panel,
    this.left_panel,
    this.minimap,
    this.message_queue,
  ];

  constructor() {
    super();
    this.html_string = html;
    this.configureElement('board');
  }

  override disconnectedCallback(): void {
    super.disconnectedCallback();
    document.body.removeEventListener('keydown', this.handleKeydown);
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
    const icon = this.icons.get(name);
    if (!icon) {
      return this.createIcon(name);
    }
    return icon;
  }

  /** Returns the unit/building icon at `name` with its color-key pixels swapped for the given player color */
  getPlayerColoredIcon(name: string, color: ColorRGB): HTMLImageElement | HTMLCanvasElement {
    const icon = this.getIcon(name);
    return this.image_cache.getPlayerColoredIcon(name, icon, PLAYER_ICON_SIZE, color) ?? icon;
  }

  async initialize(abstract_game: DwgGame, game: GameRisqFromServer): Promise<void> {
    this.player_id = abstract_game.isPlayer() ? abstract_game.playerId() : -1;
    abstract_game.setPadding('0px');
    this.setNewGameData(game);
    document.body.addEventListener('keydown', this.handleKeydown);
    const board_size: Point2D = {
      x: 1.732 * this.hex_r * (2 * game.board_size + 1),
      y: 1.5 * this.hex_r * (2 * game.board_size + 1) + 0.5 * this.hex_r,
    };
    this.board
      .initialize({
        board_size,
        max_scale: 1,
        fill_space: true,
        allow_side_move: true,
        draw: this.draw.bind(this),
        scroll: this.scrollDwg.bind(this),
        mousemove: this.mousemove.bind(this),
        // eslint-disable-next-line @typescript-eslint/naming-convention
        draggingCallback: this.draggingCallback.bind(this),
        mouseleave: this.mouseleave.bind(this),
        mousedown: this.mousedown.bind(this),
        mouseup: this.mouseup.bind(this),
        zoom_config: {
          zoom_constant: 650,
          max_zoom: 1.3,
          min_zoom: 0.7,
        },
      })
      .then((size_data) => {
        if (!size_data) {
          console.error('Not able to initialize game board');
          return;
        }
        this.boardResize(size_data.board_size, size_data.el_size);
        if (abstract_game.isPlayer()) {
          this.goToVillageCenter(this.player_id);
        } else {
          this.goToVillageCenter(0);
        }
        this.board.addEventListener('canvas_resize', (e) => {
          this.boardResize(e.detail.board_size, e.detail.el_size);
        });
      });
  }

  getGame(): GameRisq | undefined {
    return this.game;
  }

  getPlayerId(): number {
    return this.player_id;
  }

  getOrdersModel(): RisqOrdersModel {
    return this.orders_model;
  }

  getLeftPanel(): RisqLeftPanel {
    return this.left_panel;
  }

  getRightPanel(): RisqRightPanel {
    return this.right_panel;
  }

  getLocalFoundation(key: number): LocalRisqFoundation | undefined {
    return this.local_foundations.get(key);
  }

  removeLocalFoundation(key: number): void {
    this.local_foundations.delete(key);
  }

  getLocalFoundations(): Map<number, LocalRisqFoundation> {
    return this.local_foundations;
  }

  private syncLocalFoundations() {
    const all_orders = new Set(this.orders_model.all());
    for (const [key, f] of this.local_foundations.entries()) {
      if (!all_orders.has(f.order)) {
        this.local_foundations.delete(key);
      }
    }
  }

  private ordersChanged() {
    this.refreshPanels();
  }

  private refreshPanels() {
    this.syncLocalFoundations();
    this.updateResourceSpending();
    this.recomputeIdleUnits();
    this.right_panel.dataRefreshed();
    this.left_panel.dataRefreshed();
    this.recalculateHover();
  }

  private recomputeIdleUnits() {
    const player = this.getPlayer();
    if (!player) {
      this.idle_units = [];
      return;
    }
    this.idle_units = [...player.units.values()]
      .filter((u) => this.orders_model.effectiveForSubject(u.internal_id, 'unit').length === 0)
      .sort((a, b) => a.internal_id - b.internal_id);
  }

  idleUnitCount(): number {
    return this.idle_units.length;
  }

  idleBuildingCount(): number {
    const player = this.getPlayer();
    if (!player) {
      return 0;
    }
    return [...player.buildings.values()].filter(
      (b) =>
        !b.under_construction &&
        b.produces.length > 0 &&
        this.orders_model.effectiveForSubject(b.internal_id, 'building').length === 0
    ).length;
  }

  getImageCache(): RisqImageCache {
    return this.image_cache;
  }

  getPlayer(): RisqPlayer | undefined {
    if (!this.game) {
      return undefined;
    }
    return this.player_id > -1 ? this.game.players[this.player_id] : undefined;
  }

  private goToVillageCenter(player_id: number) {
    if (player_id < 0 || !this.game) {
      return;
    }
    for (const building of this.game.players[player_id].buildings.values()) {
      if (building.building_id !== 1) {
        continue;
      }
      this.goToCoordinate(building.space_coordinate);
      return;
    }
  }

  goToCoordinate(coordinate: Point2D) {
    const scale = this.last_transform.scale ?? 1;
    const view = this.coordinateToCanvas(coordinate);
    this.board.setView(multiplyPoint2D(scale, view));
  }

  private board_resize_lock = createLock();
  private boardResize(board_size: Point2D, canvas_size: DOMRect) {
    this.board_resize_lock(async () => {
      if (!this.game) {
        return;
      }
      // Update canvas dependencies
      const canvas_ratio = (0.5 * Math.min(board_size.x, canvas_size.width)) / this.canvas_center.x;
      this.canvas_center = {
        x: 0.5 * Math.min(board_size.x, canvas_size.width),
        y: 0.5 * Math.min(board_size.y, canvas_size.height),
      };
      this.canvas_size = canvas_size;
      this.hex_r = board_size.x / (1.732 * (2 * this.game.board_size + 1));
      this.hex_a = 0.5 * 1.732 * this.hex_r;
      this.board.setMaxScale((0.45 * canvas_size.height) / this.hex_r);
      this.board.scaleView(canvas_ratio);
      this.board.setOffset(this.canvas_center);
      // Update other dependencies
      for (const row of this.game?.spaces ?? []) {
        for (const space of row) {
          for (const zone_row of space.zones ?? []) {
            for (const zone of zone_row) {
              zone.reset_hovered_data = true;
            }
          }
        }
      }
      this.left_panel.resolveSize();
      this.minimap.resolveSize();
      this.toggleRightPanel(this.right_panel.isOpen());
    });
  }

  canvasSize(): DOMRect {
    return this.canvas_size;
  }

  drawDetail(): DrawRisqSpaceDetail {
    return this.draw_detail;
  }

  viewMode(): RisqViewMode {
    return this.view_mode;
  }

  toggleRightPanel(open?: boolean) {
    this.right_panel.toggle(open);
  }

  closeLeftPanel() {
    this.left_panel.close();
  }

  async gameUpdate(update: UpdateMessage): Promise<void> {
    try {
      switch (update.kind) {
        case 'start-turn':
          const start_turn_data = update.content as StartTurnData;
          await this.applyStartTurn(start_turn_data);
          break;
        case 'submitted-orders':
          const submitted_orders_data = update.content as SubmittedOrdersData;
          await this.applySubmittedOrders(submitted_orders_data);
          break;
        case 'unsubmitted-orders':
          const unsubmitted_orders_data = update.content as UnsubmittedOrdersData;
          await this.applyUnsubmittedOrders(unsubmitted_orders_data);
          break;
        case 'unit-behavior-set':
          const unit_behavior_set_data = update.content as UnitBehaviorSetData;
          this.applyUnitBehaviorSet(unit_behavior_set_data);
          break;
        case 'gather-point-set':
          const gather_point_set_data = update.content as GatherPointSetData;
          this.applyGatherPointSet(gather_point_set_data);
          break;
        default:
          console.log(`Unknown game update type ${update.kind}`);
          break;
      }
    } catch (e) {
      console.log(`Error during game update ${JSON.stringify(update)}: ${e}`);
    }
  }

  private setNewGameData(new_game: GameRisqFromServer) {
    this.game = serverToGameRisq(new_game);
    this.hovered_space = undefined;
    this.hovered_zone = undefined;
    this.orders_model.setSubmitted(this.getPlayer()?.active_orders ?? []);
    this.refreshPanels();
  }

  private async applyStartTurn(data: StartTurnData) {
    if (this.player_id > -1) {
      this.orders_submitted_times = 0;
    }
    this.clearSelection();
    this.setNewGameData(data.game);
    const player = this.getPlayer();
    if (this.player_id > -1 && player?.turn_report) {
      const dialog = document.createElement('dwg-risq-turn-report-dialog');
      dialog.setData({ risq: this, player, report: player.turn_report });
      this.appendChild(dialog);
    }
  }

  private clearSelection() {
    this.left_panel.close();
    this.disarmOrder();
  }

  private async applySubmittedOrders(data: SubmittedOrdersData) {
    if (data.player_id === this.player_id) {
      this.toggling_submit_orders_button = false;
      this.orders_model.clearPending();
    }
    this.setNewGameData(data.game);
  }

  private async applyUnsubmittedOrders(data: UnsubmittedOrdersData) {
    if (data.player_id === this.player_id) {
      this.toggling_submit_orders_button = false;
      this.orders_model.revertSubmittedToPending();
    }
    this.setNewGameData(data.game);
  }

  private applyUnitBehaviorSet(data: UnitBehaviorSetData) {
    for (const player of this.game?.players ?? []) {
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
    for (const player of this.game?.players ?? []) {
      const building = player.buildings.get(data.building_id);
      if (building) {
        building.gather_point = data.gather_point;
        break;
      }
    }
    this.refreshPanels();
  }

  private draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData) {
    if (!this.game) {
      return;
    }
    const now = Date.now();
    const dt = now - this.last_time;
    this.last_time = now;
    // set config
    this.last_transform = transform;
    // the inset rect must stay inside the hex's incircle (radius = hex_a) so it never pokes out as the map rotates
    const inset_ratio = 1.0392;
    const inset_h = (2 * this.hex_a) / Math.sqrt(inset_ratio * inset_ratio + 1);
    const inset_w = inset_ratio * inset_h;
    const inset_row = inset_h / 4 - 4;
    this.draw_detail = this.getDrawDetail(transform.scale);
    const draw_config: DrawRisqSpaceConfig = {
      hex_r: this.hex_r,
      inset_w,
      inset_h,
      inset_row,
      draw_detail: this.draw_detail,
      view_mode: this.view_mode,
      rotation: transform.rotation,
    };
    // draw spaces
    const on_screen_spaces: RisqSpace[] = [];
    for (const row of this.game.spaces) {
      for (const space of row) {
        space.center = this.coordinateToCanvas(space.coordinate);
        if (!this.isSpaceOnScreen(space)) {
          continue;
        }
        on_screen_spaces.push(space);
        drawRisqSpace(ctx, this, space, draw_config);
      }
    }
    // borders are drawn in their own pass after every space's (opaque) fill, so a later space's fill can't paint over an earlier space's border
    for (const space of on_screen_spaces) {
      drawRisqSpaceBorder(ctx, this, space, draw_config);
    }
    this.drawUnitOrders(ctx);
    this.drawGatherPointOrder(ctx);
    if (this.dragging_selection) {
      const { min, max } = normalizeRect(this.drag_start, this.drag_current);
      configDraw(
        ctx,
        transform,
        { fill_style: 'rgba(255, 255, 255, 0.15)', stroke_style: 'white', stroke_width: 1, fixed_position: true },
        false,
        false,
        () => drawRect(ctx, min, max.x - min.x, max.y - min.y)
      );
    }
    // draw panels
    for (const component of this.canvas_components) {
      component.draw(ctx, transform, dt);
    }
    // draw red dot
    if (DRAW_CENTER_DOT && DEV) {
      ctx.fillStyle = 'red';
      ctx.strokeStyle = 'transparent';
      const vis_center = multiplyPoint2D(1 / transform.scale, transform.view);
      drawCircle(ctx, vis_center, 6 / transform.scale);
    }
  }

  private drawUnitOrders(ctx: CanvasRenderingContext2D) {
    const player = this.getPlayer();
    if (!player) {
      return;
    }
    const selected = this.selectedUnitIds();
    for (const unit of player.units.values()) {
      this.drawOrdersForUnit(ctx, unit, selected.has(unit.internal_id));
    }
  }

  private selectedUnitIds(): Set<number> {
    const data = this.left_panel.getData();
    switch (data?.data_type) {
      case LeftPanelDataType.UNIT:
        return new Set([data.data.internal_id]);
      case LeftPanelDataType.UNITS_BY_TYPE:
      case LeftPanelDataType.ECONOMIC_UNITS:
      case LeftPanelDataType.MILITARY_UNITS:
        return new Set(data.data.units.flatMap((u) => [...u.units]));
      case LeftPanelDataType.MULTIPLE_PLAYERS_UNITS:
        return new Set(data.data.units_by_player.flatMap(([, units]) => units.flatMap((u) => [...u.units])));
      default:
        return new Set();
    }
  }

  // when the left panel's current selection is exactly one unit/building/resource/foundation, its kind and id; otherwise undefined
  private currentSingleSelection(): { kind: 'unit' | 'building' | 'resource' | 'foundation'; id: number } | undefined {
    const data = this.left_panel.getData();
    switch (data?.data_type) {
      case LeftPanelDataType.UNIT:
        return { kind: 'unit', id: data.data.internal_id };
      case LeftPanelDataType.BUILDING:
        return { kind: 'building', id: data.data.internal_id };
      case LeftPanelDataType.RESOURCE:
        return { kind: 'resource', id: data.data.internal_id };
      case LeftPanelDataType.FOUNDATION:
        return { kind: 'foundation', id: data.data.coordinate_key };
      case LeftPanelDataType.UNITS_BY_TYPE:
      case LeftPanelDataType.ECONOMIC_UNITS:
      case LeftPanelDataType.MILITARY_UNITS: {
        const ids = this.selectedUnitIds();
        return ids.size === 1 ? { kind: 'unit', id: [...ids][0] } : undefined;
      }
      default:
        return undefined;
    }
  }

  private currentUnitTypeGroups(): UnitByTypeData[] {
    const data = this.left_panel.getData();
    switch (data?.data_type) {
      case LeftPanelDataType.UNIT:
        return [
          { player_id: data.data.player_id, unit_id: data.data.unit_id, units: new Set([data.data.internal_id]) },
        ];
      case LeftPanelDataType.UNITS_BY_TYPE:
      case LeftPanelDataType.ECONOMIC_UNITS:
      case LeftPanelDataType.MILITARY_UNITS:
        return data.data.units;
      case LeftPanelDataType.MULTIPLE_PLAYERS_UNITS:
        return data.data.units_by_player.flatMap(([, units]) => units);
      default:
        return [];
    }
  }

  isUnitSelected(internal_id: number): boolean {
    return this.selectedUnitIds().has(internal_id);
  }

  isBuildingSelected(internal_id: number): boolean {
    const data = this.left_panel.getData();
    return data?.data_type === LeftPanelDataType.BUILDING && data.data.internal_id === internal_id;
  }

  isResourceSelected(internal_id: number): boolean {
    const data = this.left_panel.getData();
    return data?.data_type === LeftPanelDataType.RESOURCE && data.data.internal_id === internal_id;
  }

  private drawOrdersForUnit(ctx: CanvasRenderingContext2D, unit: RisqUnit, selected: boolean) {
    if (unit.garrisoned_in !== undefined) {
      return;
    }
    const orders = this.orders_model.effectiveForSubject(unit.internal_id, 'unit');
    if (!orders.length) {
      return;
    }
    const zone_view = this.draw_detail === DrawRisqSpaceDetail.ZONE_DETAILS;
    const unit_offset = this.unitAnchorOffset(unit);
    let from = this.orderPoint(unit.space_coordinate, unit_offset, zone_view);
    ctx.lineWidth = selected ? 2 : 1;
    ctx.globalAlpha = selected ? 1 : 0.35;
    ctx.setLineDash([8, 5]);
    for (const order of orders) {
      const to = this.orderTargetPoint(order, zone_view, from);
      if (!to || equalsPoint2D(from, to)) {
        continue;
      }
      const color = orderArrowColor(order.order_type);
      ctx.strokeStyle = color;
      ctx.fillStyle = color;
      drawArrow(ctx, from, to, selected ? 10 : 6);
      from = to;
    }
    ctx.setLineDash([]);
    ctx.globalAlpha = 1;
  }

  /** Space/zone-view-aware canvas point for an order endpoint; offset is ignored (space-to-space) outside zone view */
  private orderPoint(space: Point2D, offset: Point2D | undefined, zone_view: boolean): Point2D {
    const p = this.coordinateToCanvas(space);
    return zone_view && offset ? addPoint2D(p, offset) : p;
  }

  /** Offset (relative to its space's center) of the specific unit-slot circle a unit currently occupies */
  private unitAnchorOffset(unit: RisqUnit): Point2D | undefined {
    if (!this.game) {
      return undefined;
    }
    const space = getSpace(this.game, coordinateToIndex(this.game.board_size, unit.space_coordinate));
    const zone = getRisqZone(space, unit.zone_coordinate);
    if (!zone) {
      return zoneCenterOffset(unit.zone_coordinate, this.hex_r);
    }
    return (
      unitSlotWorldPosition(zone, unit.zone_coordinate, this.hex_r, this.player_id, unit.internal_id) ??
      zoneCenterOffset(unit.zone_coordinate, this.hex_r)
    );
  }

  private orderTargetPoint(order: RisqFrontendOrder, zone_view: boolean, from: Point2D): Point2D | undefined {
    if (!this.game) {
      return undefined;
    }
    let target_space: Point2D;
    let target_offset: Point2D | undefined;
    switch (order.order_type) {
      case RisqOrderType.OrderType_UnitMoveSpace:
        target_space = invertPair(order.target_id);
        break;
      case RisqOrderType.OrderType_UnitMoveZone:
      case RisqOrderType.OrderType_UnitAttackZone: {
        const decoded = invertZoneKey(order.target_id);
        target_space = decoded.space;
        const space_canvas = this.coordinateToCanvas(target_space);
        target_offset = zoneApproachPoint(decoded.zone, this.hex_r, subtractPoint2D(from, space_canvas));
        break;
      }
      case RisqOrderType.OrderType_UnitGather: {
        const decoded = invertZoneKey(order.target_id);
        target_space = decoded.space;
        target_offset = zoneCenterOffset(decoded.zone, this.hex_r);
        break;
      }
      case RisqOrderType.OrderType_UnitBuild: {
        const decoded = invertBuildKey(order.target_id);
        target_space = decoded.space;
        target_offset = zoneCenterOffset(decoded.zone, this.hex_r);
        break;
      }
      case RisqOrderType.OrderType_UnitRepair:
      case RisqOrderType.OrderType_UnitRenew:
      case RisqOrderType.OrderType_UnitAttackBuilding:
      case RisqOrderType.OrderType_UnitGarrison: {
        const building = this.findBuildingById(order.target_id);
        if (!building) {
          return undefined;
        }
        target_space = building.space_coordinate;
        target_offset = zoneCenterOffset(building.zone_coordinate, this.hex_r);
        break;
      }
      case RisqOrderType.OrderType_UnitAttackUnit: {
        const target_unit = this.findUnitById(order.target_id);
        if (!target_unit) {
          return undefined;
        }
        target_space = target_unit.space_coordinate;
        target_offset = this.unitAnchorOffset(target_unit);
        break;
      }
      default:
        return undefined;
    }
    return this.orderPoint(target_space, target_offset, zone_view);
  }

  /** Reuses orderTargetPoint's exact placement logic by building the equivalent synthetic order */
  private gatherPointTargetPoint(
    gather_point: RisqGatherPoint,
    zone_view: boolean,
    from: Point2D
  ): Point2D | undefined {
    let order_type: RisqOrderType;
    let target_id: number;
    switch (gather_point.object_type) {
      case RisqGatherObjectType.BUILDING:
        order_type = RisqOrderType.OrderType_UnitAttackBuilding;
        target_id = gather_point.object_id;
        break;
      case RisqGatherObjectType.UNIT:
        order_type = RisqOrderType.OrderType_UnitAttackUnit;
        target_id = gather_point.object_id;
        break;
      case RisqGatherObjectType.RESOURCE:
        order_type = RisqOrderType.OrderType_UnitGather;
        target_id = gather_point.location_id;
        break;
      default:
        order_type =
          gather_point.location_kind === RisqGatherPointLocationKind.SPACE
            ? RisqOrderType.OrderType_UnitMoveSpace
            : RisqOrderType.OrderType_UnitMoveZone;
        target_id = gather_point.location_id;
        break;
    }
    return this.orderTargetPoint({ player_id: this.player_id, order_type, target_id, subjects: [] }, zone_view, from);
  }

  private drawGatherPointOrder(ctx: CanvasRenderingContext2D) {
    const data = this.left_panel.getData();
    if (data?.data_type !== LeftPanelDataType.BUILDING) {
      return;
    }
    const building = data.data;
    const gather_point = building.gather_point;
    if (!gather_point) {
      return;
    }
    const zone_view = this.draw_detail === DrawRisqSpaceDetail.ZONE_DETAILS;
    const building_offset = zoneCenterOffset(building.zone_coordinate, this.hex_r);
    if (building_offset.x === 0 && building_offset.y === 0) {
      building_offset.x = 0.001;
    }
    const from = this.orderPoint(building.space_coordinate, building_offset, zone_view);
    const to = this.gatherPointTargetPoint(gather_point, zone_view, from);
    if (!to || equalsPoint2D(from, to)) {
      return;
    }
    ctx.lineWidth = 2;
    ctx.globalAlpha = 1;
    ctx.setLineDash([8, 5]);
    ctx.strokeStyle = 'white';
    ctx.fillStyle = 'white';
    drawArrow(ctx, from, to, 10);
    ctx.setLineDash([]);
    const flag_icon = this.getPlayerColoredIcon('risq/icons/garrison_flag', new ColorRGB(255, 255, 255));
    const flag_size = Math.max(16, 0.15 * this.hex_r);
    ctx.drawImage(flag_icon, to.x, to.y - flag_size, flag_size, flag_size);
  }

  private getDrawDetail(scale: number): DrawRisqSpaceDetail {
    const max_scale = this.board.getMaxScale();
    if (scale > 0.6 * (max_scale - 1) + 1) {
      return DrawRisqSpaceDetail.ZONE_DETAILS;
    } else if (scale < 1 / (0.2 * (max_scale - 1) + 1)) {
      return DrawRisqSpaceDetail.OWNERSHIP;
    }
    return DrawRisqSpaceDetail.SPACE_DETAILS;
  }

  // scroll() signature already used by HTMLElement
  private scrollDwg(dy: number, mode: number): boolean {
    for (const component of this.canvas_components) {
      if (component.isHovering()) {
        component.scroll?.(dy, mode);
        return true;
      }
    }
    return false;
  }

  recalculateHover() {
    if (!this.board || !this.board.isInitialized()) {
      return;
    }
    this.mousemove(this.mouse_canvas, this.mouse_screen, this.last_transform, {
      ctrl: false,
      shift: false,
      alt: false,
    });
  }

  private mousemove(m: Point2D, screen: Point2D, transform: BoardTransformData, modifiers: ModifierKeys) {
    if (!this.game) {
      return;
    }
    this.draw_detail = this.getDrawDetail(transform.scale);
    this.mouse_canvas = m;
    this.mouse_screen = screen;
    if (this.dragging_selection) {
      this.drag_current = screen;
      if (
        Math.hypot(this.drag_current.x - this.drag_start.x, this.drag_current.y - this.drag_start.y) >
        DRAG_SELECT_THRESHOLD
      ) {
        if (this.hovered_space) {
          this.hovered_space.clicked = false;
        }
        if (this.hovered_zone) {
          this.hovered_zone.clicked = false;
        }
      }
      return;
    }
    if (!!this.hovered_space) {
      this.hovered_space.center = this.coordinateToCanvas(this.hovered_space.coordinate);
    }
    const hovered_other_component = this.canvas_components.map((c) => c.mousemove(m, screen, transform)).some(Boolean);
    this.board.setPanSuppressed(false, this.left_panel.isHovering() || this.right_panel.isHovering());
    this.mouse_coordinate = this.canvasToCoordinate(m, this.game.board_size);
    const index = coordinateToIndex(this.game.board_size, roundAxialCoordinate(this.mouse_coordinate));
    const new_hovered_space = getSpace(this.game, index);
    if (hovered_other_component || !new_hovered_space) {
      this.removeHoveredFlags();
      if (!!this.hovered_space) {
        this.hovered_space.clicked = false;
        this.hovered_space = undefined;
        if (!!this.hovered_zone) {
          unhoverRisqZone(this.hovered_zone);
          this.hovered_zone = undefined;
        }
      }
      this.board.setCursor(DEFAULT_CURSOR_IMAGE);
      return;
    }

    const resolve_zones = () => {
      if (this.draw_detail === DrawRisqSpaceDetail.ZONE_DETAILS) {
        const new_hovered_zone = resolveHoveredZones(m, this.hovered_space, this.hex_r);
        if (!!this.hovered_zone && !equalsPoint2D(new_hovered_zone?.coordinate, this.hovered_zone?.coordinate)) {
          unhoverRisqZone(this.hovered_zone);
          this.hovered_zone = undefined;
        }
        this.hovered_zone = new_hovered_zone;
      } else if (!!this.hovered_zone) {
        unhoverRisqZone(this.hovered_zone);
        this.hovered_zone = undefined;
      }
    };

    if (equalsPoint2D(new_hovered_space.coordinate, this.hovered_space?.coordinate)) {
      this.updateHoveredFlags();
      resolve_zones.call(this);
      this.updateCursor(modifiers.ctrl);
      return;
    }
    this.removeHoveredFlags();
    if (!!this.hovered_space) {
      this.hovered_space.clicked = false;
      if (!!this.hovered_zone) {
        unhoverRisqZone(this.hovered_zone);
        this.hovered_zone = undefined;
      }
    }
    this.hovered_space = new_hovered_space;
    resolve_zones.call(this);
    this.updateHoveredFlags();
    this.updateCursor(modifiers.ctrl);
  }

  private draggingCallback() {
    this.removeHoveredFlags();
    if (!!this.hovered_space) {
      this.hovered_space.clicked = false;
    }
  }

  private mouseleave() {
    if (!!this.hovered_zone) {
      unhoverRisqZone(this.hovered_zone);
      this.hovered_zone = undefined;
    }
    if (!!this.hovered_space) {
      this.hovered_space.hovered = false;
      this.hovered_space.clicked = false;
      this.hovered_space = undefined;
    }
  }

  // returns false if mousedown event should initiate dragging
  private mousedown(e: MouseEvent): boolean {
    if (this.canvas_components.map((c) => c.mousedown(e)).some(Boolean)) {
      return true;
    }
    // left click
    if (e.button === 0) {
      if (!!this.hovered_space && this.hovered_space.visibility > 0) {
        this.hovered_space.clicked = true;
        if (this.draw_detail === DrawRisqSpaceDetail.ZONE_DETAILS && !!this.hovered_zone) {
          this.hovered_space.clicked = false;
          this.hovered_zone.clicked = true;
          for (const part of this.hovered_zone.hovered_data) {
            if (part.hovered) {
              part.clicked = true;
              this.hovered_zone.clicked = false;
              return true;
            }
          }
        }
      }
      // right click
    } else if (e.button === 2 && this.left_panel.isOrderable() && this.canGiveOrders()) {
      const left_panel_data = this.left_panel.getData();
      switch (left_panel_data?.data_type) {
        case LeftPanelDataType.UNIT:
          this.unitOrder(left_panel_data, e.ctrlKey);
          break;
        case LeftPanelDataType.UNITS_BY_TYPE:
        case LeftPanelDataType.ECONOMIC_UNITS:
        case LeftPanelDataType.MILITARY_UNITS:
          this.unitGroupOrder(left_panel_data, e.ctrlKey);
          break;
        case LeftPanelDataType.BUILDING:
          this.buildingOrder(left_panel_data);
          break;
        default:
          break;
      }
      return true;
    }
    if (this.draw_detail === DrawRisqSpaceDetail.ZONE_DETAILS && e.button === 0 && !e.shiftKey) {
      this.dragging_selection = true;
      this.drag_additive = e.ctrlKey;
      this.drag_start = { ...this.mouse_screen };
      this.drag_current = this.drag_start;
      return true;
    }
    return e.button !== 0 && e.button !== 2;
  }

  armOrder(order_type: RisqOrderType, on_disarm: () => void, building?: { id: number; display_name: string }) {
    this.armed_button_callback?.();
    this.armed_order = order_type;
    this.armed_building = building;
    this.armed_button_callback = on_disarm;
    this.gather_point_armed = false;
    this.building_attack_armed = false;
    this.updateCursor(false);
  }

  isGatherPointArmed(): boolean {
    return this.gather_point_armed;
  }

  isBuildingAttackArmed(): boolean {
    return this.building_attack_armed;
  }

  private armGatherPoint() {
    this.disarmOrder();
    this.building_attack_armed = false;
    this.gather_point_armed = true;
    this.updateCursor(false);
  }

  private disarmGatherPoint() {
    this.gather_point_armed = false;
    this.updateCursor(false);
  }

  private armBuildingAttack() {
    this.disarmOrder();
    this.gather_point_armed = false;
    this.building_attack_armed = true;
    this.updateCursor(false);
  }

  private disarmBuildingAttack() {
    this.building_attack_armed = false;
    this.updateCursor(false);
  }

  getArmedOrder(): RisqOrderType {
    return this.armed_order;
  }

  getArmedBuildingId(): number {
    return this.armed_building?.id ?? 0;
  }

  getArmedBuilding(): { id: number; display_name: string } | undefined {
    return this.armed_building;
  }

  disarmOrder() {
    this.armed_button_callback?.();
    this.armed_order = RisqOrderType.NONE;
    this.armed_building = undefined;
    this.armed_button_callback = undefined;
    this.updateCursor(false);
  }

  createUnit(building_id: number, unit_id: number) {
    if (!this.canGiveOrders()) {
      return;
    }
    this.orders_model.add({
      player_id: this.player_id,
      order_type: RisqOrderType.OrderType_BuildingCreate,
      subjects: [building_id],
      target_id: unit_id,
      clear_previous_orders: false,
    });
  }

  researchTech(building_id: number, tech_id: number) {
    if (!this.canGiveOrders()) {
      return;
    }
    this.orders_model.add({
      player_id: this.player_id,
      order_type: RisqOrderType.OrderType_BuildingResearch,
      subjects: [building_id],
      target_id: tech_id,
      clear_previous_orders: false,
    });
  }

  confirmDeleteUnit(internal_ids: number[]) {
    const dialog = document.createElement('dwg-confirm-dialog');
    dialog.setData({
      question: `Are you sure you want to delete ${internal_ids.length === 1 ? 'this unit' : 'these units'}?`,
      size: DialogSize.SMALL,
    });
    dialog.addEventListener('confirmed', () => {
      this.deleteUnit(internal_ids);
    });
    this.appendChild(dialog);
  }

  private deleteUnit(internal_ids: number[]) {
    if (!this.canGiveOrders() || internal_ids.length === 0) {
      return;
    }
    this.orders_model.add({
      player_id: this.player_id,
      order_type: RisqOrderType.OrderType_UnitDelete,
      subjects: internal_ids,
      target_id: 0,
      clear_previous_orders: true,
    });
  }

  confirmDeleteBuilding(internal_id: number) {
    const dialog = document.createElement('dwg-confirm-dialog');
    dialog.setData({
      question: 'Are you sure you want to delete this building?',
      size: DialogSize.SMALL,
    });
    dialog.addEventListener('confirmed', () => {
      this.deleteBuilding(internal_id);
    });
    this.appendChild(dialog);
  }

  private deleteBuilding(internal_id: number) {
    if (!this.canGiveOrders()) {
      return;
    }
    this.orders_model.add({
      player_id: this.player_id,
      order_type: RisqOrderType.OrderType_BuildingDelete,
      subjects: [internal_id],
      target_id: 0,
      clear_previous_orders: true,
    });
  }

  attackFromBuilding(_building_id: number) {
    if (this.building_attack_armed) {
      this.disarmBuildingAttack();
    } else {
      this.armBuildingAttack();
    }
  }

  toggleBuildingGatherPoint(_building_id: number) {
    if (this.gather_point_armed) {
      this.disarmGatherPoint();
    } else {
      this.armGatherPoint();
    }
  }

  ungarrisonBuilding(_building_id: number) {
    // TODO: building-initiated ungarrison isn't implemented yet
  }

  stopUnit(internal_ids: number[]) {
    if (!this.canGiveOrders()) {
      return;
    }
    for (const internal_id of internal_ids) {
      this.orders_model.cancelForSubject(internal_id);
    }
  }

  ungarrisonUnits(internal_ids: number[]) {
    if (!this.canGiveOrders() || internal_ids.length === 0) {
      return;
    }
    this.orders_model.add({
      player_id: this.player_id,
      order_type: RisqOrderType.OrderType_UnitUngarrison,
      subjects: internal_ids,
      target_id: 0,
      clear_previous_orders: true,
    });
  }

  setGatherPoint(building_id: number, point: RisqGatherPoint) {
    if (!this.canGiveOrders()) {
      return;
    }
    const game_update = createMessage(
      `player-${this.player_id}`,
      'game-update',
      JSON.stringify({ building_id, clear: false, ...point }),
      'set-gather-point'
    );
    this.dispatchEvent(new CustomEvent('game_update', { detail: game_update, bubbles: true }));
  }

  clearGatherPoint(building_id: number) {
    if (!this.canGiveOrders()) {
      return;
    }
    const game_update = createMessage(
      `player-${this.player_id}`,
      'game-update',
      JSON.stringify({ building_id, clear: true }),
      'set-gather-point'
    );
    this.dispatchEvent(new CustomEvent('game_update', { detail: game_update, bubbles: true }));
  }

  private sendUnitBehavior(internal_ids: number[], fields: Record<string, unknown>) {
    if (!this.canGiveOrders() || internal_ids.length === 0) {
      return;
    }
    const game_update = createMessage(
      `player-${this.player_id}`,
      'game-update',
      JSON.stringify({ internal_ids, ...fields }),
      'set-unit-behavior'
    );
    this.dispatchEvent(new CustomEvent('game_update', { detail: game_update, bubbles: true }));
  }

  setUnitStance(internal_ids: number[], stance: RisqUnitStance) {
    this.sendUnitBehavior(internal_ids, { stance });
  }

  setUnitToggle(internal_ids: number[], field: UnitToggleField, value: boolean) {
    this.sendUnitBehavior(internal_ids, { [field]: value });
  }

  selectOrderSubjects(order: RisqFrontendOrder) {
    const player = this.getPlayer();
    const game = this.game;
    if (!player || !game) {
      return;
    }
    if (isBuildingOrder(order.order_type)) {
      const building = player.buildings.get(order.subjects[0]);
      if (building) {
        this.left_panel.openPanel({ data_type: LeftPanelDataType.BUILDING, data: building }, RisqVisibilityLevel.SPY);
      }
      return;
    }
    if (!isUnitOrder(order.order_type)) {
      return;
    }
    const units_by_type = groupUnitsByType(player.units, order.subjects);
    const first = order.subjects.map((id) => player.units.get(id)).find((u) => !!u);
    if (units_by_type.length === 0 || !first) {
      return;
    }
    const space = getSpace(game, coordinateToIndex(game.board_size, first.space_coordinate));
    if (!space) {
      return;
    }
    this.left_panel.openPanel(
      {
        data_type: LeftPanelDataType.UNITS,
        data: { space, units_by_player: new Map([[player.player.player_id, units_by_type]]) },
      },
      RisqVisibilityLevel.SPY
    );
  }

  showMessage(text: string, color: ColorRGB) {
    this.message_queue.enqueue(text, color);
  }

  selectNextIdleUnit() {
    if (!this.game || this.idle_units.length === 0) {
      return;
    }
    let idx = 0;
    if (this.last_idle_selected !== undefined) {
      const found = this.idle_units.findIndex((u) => u.internal_id > this.last_idle_selected!);
      idx = found === -1 ? 0 : found;
    }
    const unit = this.idle_units[idx];
    this.last_idle_selected = unit.internal_id;
    this.left_panel.openPanel({ data_type: LeftPanelDataType.UNIT, data: unit }, RisqVisibilityLevel.SPY);
    const scale = this.last_transform.scale;
    const view = addPoint2D(
      this.coordinateToCanvas(unit.space_coordinate),
      zoneCenterOffset(unit.zone_coordinate, this.hex_r)
    );
    this.board.setView(multiplyPoint2D(scale, view));
  }

  private hasNegativeResources(): boolean {
    const player = this.getPlayer();
    return !!player && [...player.resources.values()].some((pr) => pr.amount - pr.spending < 0);
  }

  confirmSubmitOrders() {
    const n_units = this.idleUnitCount();
    const n_buildings = this.idleBuildingCount();
    const n = n_units + n_buildings;
    const over_budget = this.hasNegativeResources();
    if (n === 0 && !over_budget) {
      this.toggleSubmitOrdersButton();
      return;
    }
    const warnings: string[] = [];
    if (n > 0) {
      let noun: string;
      if (n_units > 0 && n_buildings > 0) {
        noun = 'units and buildings';
      } else if (n_buildings > 0) {
        noun = n_buildings === 1 ? 'building' : 'buildings';
      } else {
        noun = n_units === 1 ? 'unit' : 'units';
      }
      warnings.push(`You have ${n} idle ${noun}.`);
    }
    if (over_budget) {
      warnings.push('One or more resources will go negative.');
    }
    const dialog = document.createElement('dwg-confirm-dialog');
    dialog.setData({
      question: `Are you sure you want to submit orders? ${warnings.join(' ')}`,
      size: DialogSize.SMALL,
    });
    dialog.addEventListener('confirmed', () => this.toggleSubmitOrdersButton());
    this.appendChild(dialog);
  }

  private assignControlGroup(group: number) {
    if (!this.left_panel.isOrderable()) {
      return;
    }
    const data = this.left_panel.getData();
    switch (data?.data_type) {
      case LeftPanelDataType.UNIT:
        this.control_groups.set(group, { kind: 'unit', ids: [data.data.internal_id] });
        break;
      case LeftPanelDataType.BUILDING:
        this.control_groups.set(group, { kind: 'building', ids: [data.data.internal_id] });
        break;
      case LeftPanelDataType.UNITS_BY_TYPE:
      case LeftPanelDataType.ECONOMIC_UNITS:
      case LeftPanelDataType.MILITARY_UNITS:
        this.control_groups.set(group, { kind: 'unit', ids: data.data.units.flatMap((u) => [...u.units]) });
        break;
      default:
        break;
    }
  }

  private selectControlGroup(group: number) {
    const group_data = this.control_groups.get(group);
    const player = this.getPlayer();
    if (!group_data || !player) {
      return;
    }
    if (group_data.kind === 'building') {
      const building = player.buildings.get(group_data.ids[0]);
      if (!building) {
        this.control_groups.delete(group);
        return;
      }
      this.left_panel.openPanel({ data_type: LeftPanelDataType.BUILDING, data: building }, RisqVisibilityLevel.SPY);
      return;
    }
    const alive_ids = group_data.ids.filter((id) => player.units.has(id));
    if (alive_ids.length === 0) {
      this.control_groups.delete(group);
      return;
    }
    group_data.ids = alive_ids;
    this.left_panel.openPanel(
      { data_type: LeftPanelDataType.UNITS_BY_TYPE, data: { units: groupUnitsByType(player.units, alive_ids) } },
      RisqVisibilityLevel.SPY
    );
  }

  private updateResourceSpending() {
    const player = this.getPlayer();
    if (!player) {
      return;
    }
    for (const pr of player.resources.values()) {
      pr.spending = 0;
    }
    for (const order of this.orders_model.pendingOrders()) {
      let kind: RisqProducibleKind;
      if (order.order_type === RisqOrderType.OrderType_BuildingCreate) {
        kind = RisqProducibleKind.UNIT;
      } else if (order.order_type === RisqOrderType.OrderType_BuildingResearch) {
        kind = RisqProducibleKind.TECH;
      } else {
        continue;
      }
      for (const subject_id of order.subjects) {
        const cost = player.buildings
          .get(subject_id)
          ?.produces.find((p) => p.kind === kind && p.id === order.target_id)?.cost;
        if (!cost) {
          continue;
        }
        player.resources.get(RisqResourceType.FOOD)!.spending += cost.food;
        player.resources.get(RisqResourceType.WOOD)!.spending += cost.wood;
        player.resources.get(RisqResourceType.STONE)!.spending += cost.stone;
        player.resources.get(RisqResourceType.GOLD)!.spending += cost.gold;
      }
    }
  }

  private atSelectedUnitPosition(zone_valid: boolean, ctrl_held: boolean): boolean {
    if (ctrl_held || !this.hovered_space) {
      return false;
    }
    const data = this.left_panel.getData();
    if (data?.data_type !== LeftPanelDataType.UNIT) {
      return false;
    }
    const unit = data.data;
    if (!equalsPoint2D(this.hovered_space.coordinate, unit.space_coordinate)) {
      return false;
    }
    return zone_valid ? equalsPoint2D(this.hovered_zone?.coordinate, unit.zone_coordinate) : true;
  }

  private isZoneValid(): boolean {
    return (
      this.drawDetail() === DrawRisqSpaceDetail.ZONE_DETAILS &&
      !!this.hovered_space &&
      this.hovered_space.visibility >= RisqVisibilityLevel.FOG &&
      !!this.hovered_zone
    );
  }

  hasPlannedFoundation(zone: RisqZone | undefined): boolean {
    if (!zone) {
      return false;
    }
    const has_local = this.local_foundations.has(zone.coordinate_key);
    const has_server = !!this.getPlayer()?.planned_foundations?.has(zone.coordinate_key);
    return has_local || has_server;
  }

  private buildTargetValid(): boolean {
    const ownership = this.hovered_space?.ownership;
    return (
      this.left_panel.hasVillager() &&
      this.isZoneValid() &&
      !this.hovered_zone?.resource &&
      !this.hovered_zone?.building &&
      !this.hasPlannedFoundation(this.hovered_zone) &&
      (ownership === undefined || ownership < 0 || ownership === this.player_id)
    );
  }

  private repairTargetValid(): boolean {
    const b = this.hovered_zone?.building;
    return (
      this.isZoneValid() &&
      !!b &&
      b.player_id === this.player_id &&
      !b.under_construction &&
      b.combat_stats.health < b.combat_stats.max_health
    );
  }

  private renewTargetValid(): boolean {
    const b = this.hovered_zone?.building;
    return (
      this.isZoneValid() &&
      !!b &&
      b.player_id === this.player_id &&
      !b.under_construction &&
      b.gather_capacity !== undefined &&
      (b.resources_left ?? 0) <= 0
    );
  }

  private findBuildingById(internal_id: number): RisqBuilding | undefined {
    for (const player of this.game?.players ?? []) {
      const building = player.buildings.get(internal_id);
      if (building) {
        return building;
      }
    }
    return undefined;
  }

  private findUnitById(internal_id: number): RisqUnit | undefined {
    for (const player of this.game?.players ?? []) {
      const unit = player.units.get(internal_id);
      if (unit) {
        return unit;
      }
    }
    return undefined;
  }

  private unitsInZoneOfType(zone: RisqZone, player_id: number, unit_id: number): number[] {
    return [...zone.units.values()]
      .filter((u) => u.player_id === player_id && u.unit_id === unit_id)
      .map((u) => u.internal_id);
  }

  // space-level bounding-box check first (cheap pre-filter), then whether the unit's own rendered circle overlaps the viewport
  private unitsOnScreenOfType(player_id: number, unit_id: number): number[] {
    const ids: number[] = [];
    for (const row of this.game?.spaces ?? []) {
      for (const space of row) {
        if (!this.isSpaceOnScreen(space)) {
          continue;
        }
        for (const zone_row of space.zones ?? []) {
          for (const zone of zone_row) {
            for (const unit of zone.units.values()) {
              if (unit.player_id !== player_id || unit.unit_id !== unit_id) {
                continue;
              }
              const canvas_pos = this.orderPoint(unit.space_coordinate, this.unitAnchorOffset(unit), true);
              const radius = UNIT_SLOT_CIRCLE_RADIUS_MULTIPLIER * this.hex_r;
              if (this.isCircleOnScreen(canvas_pos, radius)) {
                ids.push(unit.internal_id);
              }
            }
          }
        }
      }
    }
    return ids;
  }

  private visibleCanvasBounds(): { min: Point2D; max: Point2D } {
    const transform = this.last_transform;
    const { width, height } = this.canvas_size;
    const corners = [
      screenToCanvas({ x: 0, y: 0 }, transform),
      screenToCanvas({ x: width, y: 0 }, transform),
      screenToCanvas({ x: 0, y: height }, transform),
      screenToCanvas({ x: width, y: height }, transform),
    ];
    return {
      min: { x: Math.min(...corners.map((c) => c.x)), y: Math.min(...corners.map((c) => c.y)) },
      max: { x: Math.max(...corners.map((c) => c.x)), y: Math.max(...corners.map((c) => c.y)) },
    };
  }

  private isSpaceOnScreen(space: RisqSpace): boolean {
    const { min, max } = this.visibleCanvasBounds();
    return !(
      space.center.x + this.hex_a < min.x ||
      space.center.x - this.hex_a > max.x ||
      space.center.y + this.hex_r < min.y ||
      space.center.y - this.hex_r > max.y
    );
  }

  private isCircleOnScreen(c: Point2D, radius: number): boolean {
    const { min, max } = this.visibleCanvasBounds();
    const closest_x = Math.min(Math.max(c.x, min.x), max.x);
    const closest_y = Math.min(Math.max(c.y, min.y), max.y);
    const dx = c.x - closest_x;
    const dy = c.y - closest_y;
    return dx * dx + dy * dy <= radius * radius;
  }

  private openUnitTypeSelection(
    player_id: number,
    internal_ids: number[],
    space: RisqSpace | undefined,
    visibility: number
  ) {
    if (internal_ids.length === 0) {
      return;
    }
    const units_map =
      this.game?.players.find((p) => p.player.player_id === player_id)?.units ?? new Map<number, RisqUnit>();
    const units = groupUnitsByType(units_map, internal_ids);
    this.left_panel.openPanel({ data_type: LeftPanelDataType.UNITS_BY_TYPE, data: { space, units } }, visibility);
  }

  private resolveActiveOrderType(ctrl_held: boolean): RisqOrderType {
    if (!this.left_panel.getData() || !this.left_panel.isOrderable() || !this.canGiveOrders() || !this.hovered_space) {
      return RisqOrderType.NONE;
    }
    const is_unit = this.left_panel.isUnit();
    const has_villager = this.left_panel.hasVillager();
    const has_military = this.left_panel.hasMilitary();
    const zone_valid = this.isZoneValid();
    const armed_order = this.getArmedOrder();
    if (armed_order !== RisqOrderType.NONE) {
      switch (armed_order) {
        case RisqOrderType.OrderType_UnitMoveSpace:
        case RisqOrderType.OrderType_UnitMoveZone:
          if (is_unit && !this.atSelectedUnitPosition(zone_valid, ctrl_held)) {
            return zone_valid ? RisqOrderType.OrderType_UnitMoveZone : RisqOrderType.OrderType_UnitMoveSpace;
          }
          return RisqOrderType.NONE;
        case RisqOrderType.OrderType_UnitAttackSpace:
        case RisqOrderType.OrderType_UnitAttackZone: {
          if (!is_unit) {
            return RisqOrderType.NONE;
          }
          const hovered = this.hovered_zone && hoveredZoneObject(this.hovered_zone);
          if (hovered?.kind === 'building' && this.hovered_zone!.building!.player_id !== this.player_id) {
            return RisqOrderType.OrderType_UnitAttackBuilding;
          }
          if (hovered?.kind === 'unit' && hovered.groups.some((t) => t.player_id !== this.player_id)) {
            return RisqOrderType.OrderType_UnitAttackUnit;
          }
          return zone_valid ? RisqOrderType.OrderType_UnitAttackZone : RisqOrderType.OrderType_UnitAttackSpace;
        }
        case RisqOrderType.OrderType_UnitGather:
          if (has_villager && zone_valid && !!this.hovered_zone?.resource) {
            return RisqOrderType.OrderType_UnitGather;
          }
          return RisqOrderType.NONE;
        case RisqOrderType.OrderType_UnitBuild:
          if (this.buildTargetValid()) {
            return RisqOrderType.OrderType_UnitBuild;
          }
          return RisqOrderType.NONE;
        case RisqOrderType.OrderType_UnitRepair:
          if (has_villager && this.repairTargetValid()) {
            return RisqOrderType.OrderType_UnitRepair;
          }
          return RisqOrderType.NONE;
        case RisqOrderType.OrderType_UnitRenew:
          if (has_villager && this.renewTargetValid()) {
            return RisqOrderType.OrderType_UnitRenew;
          }
          return RisqOrderType.NONE;
        case RisqOrderType.OrderType_UnitGarrison: {
          const building = this.hovered_zone?.building;
          if (
            zone_valid &&
            building &&
            building.player_id === this.player_id &&
            !building.under_construction &&
            (building.garrisoned_units?.length ?? 0) < building.garrison_capacity
          ) {
            return RisqOrderType.OrderType_UnitGarrison;
          }
          return RisqOrderType.NONE;
        }
        default:
          return RisqOrderType.NONE;
      }
    }
    if (is_unit) {
      if (has_military && zone_valid && this.hovered_zone) {
        const hovered = hoveredZoneObject(this.hovered_zone);
        if (hovered?.kind === 'building' && this.hovered_zone.building!.player_id !== this.player_id) {
          return RisqOrderType.OrderType_UnitAttackBuilding;
        }
        if (hovered?.kind === 'unit' && hovered.groups.some((t) => t.player_id !== this.player_id)) {
          return RisqOrderType.OrderType_UnitAttackUnit;
        }
      }
      if (
        zone_valid &&
        this.hovered_zone?.hovered_data[0]?.hovered &&
        this.hovered_zone.building &&
        this.hovered_zone.building.player_id === this.player_id &&
        !this.hovered_zone.building.under_construction &&
        (this.hovered_zone.building.garrisoned_units?.length ?? 0) < this.hovered_zone.building.garrison_capacity
      ) {
        return RisqOrderType.OrderType_UnitGarrison;
      }
      if (has_villager && this.hovered_zone?.hovered_data[0]?.hovered && this.renewTargetValid()) {
        return RisqOrderType.OrderType_UnitRenew;
      }
      if (has_villager && this.hovered_zone?.hovered_data[0]?.hovered && this.repairTargetValid()) {
        return RisqOrderType.OrderType_UnitRepair;
      }
      if (has_villager && zone_valid && this.hovered_zone?.hovered_data[0]?.hovered) {
        if (!!this.hovered_zone.resource) {
          return RisqOrderType.OrderType_UnitGather;
        }
        if (this.hovered_zone.building?.under_construction || this.hasPlannedFoundation(this.hovered_zone)) {
          return RisqOrderType.OrderType_UnitBuild;
        }
      }
      if (this.atSelectedUnitPosition(zone_valid, ctrl_held)) {
        return RisqOrderType.NONE;
      }
      return zone_valid ? RisqOrderType.OrderType_UnitMoveZone : RisqOrderType.OrderType_UnitMoveSpace;
    }
    return RisqOrderType.NONE;
  }

  private updateCursor(ctrl_held: boolean) {
    if (this.gather_point_armed) {
      this.board.setCursor('gather_point');
      return;
    }
    if (this.getArmedOrder() === RisqOrderType.OrderType_UnitBuild && this.armed_building) {
      const url = resolveBuildCursorUrl(this, this.armed_building, this.buildTargetValid());
      if (url) {
        this.board.setCursorUrl(url);
        return;
      }
    }
    this.board.setCursor(cursorImageForOrderType(this.resolveActiveOrderType(ctrl_held)));
  }

  private canGiveOrders(): boolean {
    const player = this.getPlayer();
    const game = this.getGame();
    if (!player || !game) {
      return false;
    }
    if (player.orders_submitted) {
      return false;
    }
    if (!game.giving_orders) {
      return false;
    }
    return true;
  }

  private addUnitOrder(order_type: RisqOrderType, subjects: number[], target_id: number, ctrl_held: boolean) {
    if (subjects.length === 0) {
      return;
    }
    if (order_type === RisqOrderType.OrderType_UnitRenew) {
      const player = this.getPlayer();
      const cost = this.findBuildingById(target_id)?.renew_cost;
      if (player && cost && !canAffordCost(player, cost)) {
        this.showMessage('Not enough resources', RISQ_MESSAGE_WARNING_COLOR);
        return;
      }
    }
    this.orders_model.add({
      player_id: this.player_id,
      order_type,
      subjects,
      target_id,
      clear_previous_orders: !ctrl_held,
    });
  }

  // internal_id of the enemy unit in whichever unit-slot is currently hovered, if any
  private resolveHoveredEnemyUnitId(): number | undefined {
    const zone = this.hovered_zone;
    for (const [i, slot] of (zone?.unit_slots ?? []).entries()) {
      if (!zone?.hovered_data[i + 1]?.hovered) {
        continue;
      }
      const enemy = slot.find((t) => t.player_id !== this.player_id);
      return enemy ? [...enemy.units][0] : undefined;
    }
    return undefined;
  }

  private buildingOrder(data: BuildingData) {
    const target = this.resolveBuildingAttackTarget(data.data);
    if (target) {
      this.orders_model.add({
        player_id: this.player_id,
        order_type:
          target.kind === 'unit'
            ? RisqOrderType.OrderType_BuildingAttackUnit
            : RisqOrderType.OrderType_BuildingAttackBuilding,
        subjects: [data.data.internal_id],
        target_id: target.internal_id,
        clear_previous_orders: false,
      });
    } else {
      this.buildingGatherPointOrder(data);
    }
    this.disarmGatherPoint();
    this.disarmBuildingAttack();
  }

  private resolveBuildingAttackTarget(
    building: RisqBuilding
  ): { kind: 'unit' | 'building'; internal_id: number } | undefined {
    if (building.combat_stats.attack_type === RisqAttackType.NONE) {
      return undefined;
    }
    if (this.draw_detail === DrawRisqSpaceDetail.ZONE_DETAILS && this.hovered_zone) {
      const zone = this.hovered_zone;
      const hovered = hoveredZoneObject(zone);
      if (hovered?.kind === 'building' && zone.building && zone.building.player_id !== this.player_id) {
        return { kind: 'building', internal_id: zone.building.internal_id };
      }
      if (hovered?.kind === 'unit') {
        const enemy = hovered.groups.find((t) => t.player_id !== this.player_id);
        const unit_id = enemy ? [...enemy.units][0] : undefined;
        if (unit_id !== undefined) {
          return { kind: 'unit', internal_id: unit_id };
        }
      }
      if (!this.building_attack_armed) {
        return undefined;
      }
      if (zone.building && zone.building.player_id !== this.player_id) {
        return { kind: 'building', internal_id: zone.building.internal_id };
      }
      const enemy_unit = [...zone.units.values()].find((u) => u.player_id !== this.player_id);
      if (enemy_unit) {
        return { kind: 'unit', internal_id: enemy_unit.internal_id };
      }
      return undefined;
    }
    if (this.building_attack_armed && this.hovered_space) {
      const enemy_building = [...(this.hovered_space.buildings?.values() ?? [])].find(
        (b) => b.player_id !== this.player_id
      );
      if (enemy_building) {
        return { kind: 'building', internal_id: enemy_building.internal_id };
      }
      const enemy_unit = [...(this.hovered_space.units?.values() ?? [])].find((u) => u.player_id !== this.player_id);
      if (enemy_unit) {
        return { kind: 'unit', internal_id: enemy_unit.internal_id };
      }
    }
    return undefined;
  }

  private buildingGatherPointOrder(data: BuildingData) {
    const building = data.data;
    if (!canHaveGatherPoint(building)) {
      return;
    }
    let point: RisqGatherPoint;
    if (this.draw_detail === DrawRisqSpaceDetail.ZONE_DETAILS && this.hovered_zone) {
      point = {
        location_kind: RisqGatherPointLocationKind.ZONE,
        location_id: this.hovered_zone.coordinate_key,
        object_type: RisqGatherObjectType.NONE,
        object_id: 0,
      };
      const hovered = hoveredZoneObject(this.hovered_zone);
      if (hovered?.kind === 'resource' && this.hovered_zone.resource) {
        point.object_type = RisqGatherObjectType.RESOURCE;
        point.object_id = this.hovered_zone.resource.internal_id;
      } else if (hovered?.kind === 'building' && this.hovered_zone.building) {
        point.object_type = RisqGatherObjectType.BUILDING;
        point.object_id = this.hovered_zone.building.internal_id;
      } else if (hovered?.kind === 'unit') {
        const unit_id = [...hovered.groups[0].units][0];
        if (unit_id !== undefined) {
          point.object_type = RisqGatherObjectType.UNIT;
          point.object_id = unit_id;
        }
      }
    } else if (this.hovered_space) {
      point = {
        location_kind: RisqGatherPointLocationKind.SPACE,
        location_id: this.hovered_space.coordinate_key,
        object_type: RisqGatherObjectType.NONE,
        object_id: 0,
      };
    } else {
      return;
    }
    this.setGatherPoint(building.internal_id, point);
  }

  private unitOrder(data: UnitData, ctrl_held: boolean) {
    if (!this.hovered_space) {
      this.disarmOrder();
      return;
    }
    // TODO: implement attack vs just move
    // TODO: implement if holding the shift key
    switch (this.resolveActiveOrderType(ctrl_held)) {
      case RisqOrderType.OrderType_UnitMoveSpace:
        this.addUnitOrder(
          RisqOrderType.OrderType_UnitMoveSpace,
          [data.data.internal_id],
          this.hovered_space.coordinate_key,
          ctrl_held
        );
        break;
      case RisqOrderType.OrderType_UnitMoveZone:
        if (!this.hovered_zone) {
          return;
        }
        this.addUnitOrder(
          RisqOrderType.OrderType_UnitMoveZone,
          [data.data.internal_id],
          this.hovered_zone.coordinate_key,
          ctrl_held
        );
        break;
      case RisqOrderType.OrderType_UnitAttackBuilding: {
        const target = this.hovered_zone?.building;
        if (!target) {
          return;
        }
        this.addUnitOrder(
          RisqOrderType.OrderType_UnitAttackBuilding,
          [data.data.internal_id],
          target.internal_id,
          ctrl_held
        );
        break;
      }
      case RisqOrderType.OrderType_UnitAttackUnit: {
        const target_id = this.resolveHoveredEnemyUnitId();
        if (target_id === undefined) {
          return;
        }
        this.addUnitOrder(RisqOrderType.OrderType_UnitAttackUnit, [data.data.internal_id], target_id, ctrl_held);
        break;
      }
      case RisqOrderType.OrderType_UnitGarrison: {
        const target = this.hovered_zone?.building;
        if (!target) {
          return;
        }
        this.addUnitOrder(RisqOrderType.OrderType_UnitGarrison, [data.data.internal_id], target.internal_id, ctrl_held);
        break;
      }
      case RisqOrderType.OrderType_UnitAttackSpace:
        this.addUnitOrder(
          RisqOrderType.OrderType_UnitAttackSpace,
          [data.data.internal_id],
          this.hovered_space.coordinate_key,
          ctrl_held
        );
        break;
      case RisqOrderType.OrderType_UnitAttackZone:
        if (!this.hovered_zone) {
          return;
        }
        this.addUnitOrder(
          RisqOrderType.OrderType_UnitAttackZone,
          [data.data.internal_id],
          this.hovered_zone.coordinate_key,
          ctrl_held
        );
        break;
      case RisqOrderType.OrderType_UnitGather:
        if (!this.hovered_zone) {
          return;
        }
        this.addUnitOrder(
          RisqOrderType.OrderType_UnitGather,
          [data.data.internal_id],
          this.hovered_zone.coordinate_key,
          ctrl_held
        );
        break;
      case RisqOrderType.OrderType_UnitBuild: {
        if (!this.hovered_zone) {
          return;
        }
        const local_foundation = this.local_foundations.get(this.hovered_zone.coordinate_key);
        if (local_foundation) {
          if (!local_foundation.order.subjects.includes(data.data.internal_id)) {
            if (!ctrl_held) {
              this.orders_model.cancelForSubject(data.data.internal_id);
            }
            local_foundation.order.subjects.push(data.data.internal_id);
            this.orders_model.triggerChange();
          }
        } else {
          const b = this.hovered_zone.building;
          const server_f = this.getPlayer()?.planned_foundations?.get(this.hovered_zone.coordinate_key);
          const building_id = this.armed_building ? this.armed_building.id : b ? b.building_id : server_f?.building_id;

          if (building_id !== undefined) {
            const order: RisqFrontendOrder = {
              player_id: this.player_id,
              order_type: RisqOrderType.OrderType_UnitBuild,
              subjects: [data.data.internal_id],
              target_id: cantorPair(building_id, this.hovered_zone.coordinate_key),
              clear_previous_orders: !ctrl_held,
            };
            this.orders_model.add(order);

            if (this.armed_building) {
              this.local_foundations.set(this.hovered_zone.coordinate_key, {
                coordinate_key: this.hovered_zone.coordinate_key,
                building_id: this.armed_building.id,
                display_name: this.armed_building.display_name,
                order,
              });
            }
          }
        }
        break;
      }
      case RisqOrderType.OrderType_UnitRepair:
        if (!this.hovered_zone?.building) {
          return;
        }
        this.addUnitOrder(
          RisqOrderType.OrderType_UnitRepair,
          [data.data.internal_id],
          this.hovered_zone.building.internal_id,
          ctrl_held
        );
        break;
      case RisqOrderType.OrderType_UnitRenew:
        if (!this.hovered_zone?.building) {
          return;
        }
        this.addUnitOrder(
          RisqOrderType.OrderType_UnitRenew,
          [data.data.internal_id],
          this.hovered_zone.building.internal_id,
          ctrl_held
        );
        break;
      default:
        this.disarmOrder();
        return;
    }
    this.disarmOrder();
  }

  // Returns whether the given unit is already sitting at the current hover target (never true while ctrl is held)
  private isUnitAtHoverTarget(internal_id: number, zone_valid: boolean, ctrl_held: boolean): boolean {
    if (ctrl_held || !this.hovered_space) {
      return false;
    }
    const unit = this.getPlayer()?.units.get(internal_id);
    if (!unit || !equalsPoint2D(this.hovered_space.coordinate, unit.space_coordinate)) {
      return false;
    }
    return zone_valid ? equalsPoint2D(this.hovered_zone?.coordinate, unit.zone_coordinate) : true;
  }

  private unitGroupOrder(data: UnitsByTypeData | EconomicUnitsData | MilitaryUnitsData, ctrl_held: boolean) {
    if (!this.hovered_space) {
      this.disarmOrder();
      return;
    }
    const units = data.data.units.flatMap((u) =>
      [...u.units].map((internal_id) => ({ unit_id: u.unit_id, internal_id }))
    );
    switch (this.resolveActiveOrderType(ctrl_held)) {
      case RisqOrderType.OrderType_UnitMoveSpace: {
        const subjects = units
          .filter((u) => !this.isUnitAtHoverTarget(u.internal_id, false, ctrl_held))
          .map((u) => u.internal_id);
        this.addUnitOrder(
          RisqOrderType.OrderType_UnitMoveSpace,
          subjects,
          this.hovered_space.coordinate_key,
          ctrl_held
        );
        break;
      }
      case RisqOrderType.OrderType_UnitMoveZone: {
        if (!this.hovered_zone) {
          return;
        }
        const subjects = units
          .filter((u) => !this.isUnitAtHoverTarget(u.internal_id, true, ctrl_held))
          .map((u) => u.internal_id);
        this.addUnitOrder(RisqOrderType.OrderType_UnitMoveZone, subjects, this.hovered_zone.coordinate_key, ctrl_held);
        break;
      }
      case RisqOrderType.OrderType_UnitAttackBuilding: {
        const target = this.hovered_zone?.building;
        if (!target) {
          return;
        }
        const attackers = this.getArmedOrder() === RisqOrderType.NONE ? units.filter((u) => u.unit_id > 10) : units;
        this.addUnitOrder(
          RisqOrderType.OrderType_UnitAttackBuilding,
          attackers.map((u) => u.internal_id),
          target.internal_id,
          ctrl_held
        );
        break;
      }
      case RisqOrderType.OrderType_UnitAttackUnit: {
        const target_id = this.resolveHoveredEnemyUnitId();
        if (target_id === undefined) {
          return;
        }
        const attackers = this.getArmedOrder() === RisqOrderType.NONE ? units.filter((u) => u.unit_id > 10) : units;
        this.addUnitOrder(
          RisqOrderType.OrderType_UnitAttackUnit,
          attackers.map((u) => u.internal_id),
          target_id,
          ctrl_held
        );
        break;
      }
      case RisqOrderType.OrderType_UnitAttackSpace: {
        this.addUnitOrder(
          RisqOrderType.OrderType_UnitAttackSpace,
          units.map((u) => u.internal_id),
          this.hovered_space.coordinate_key,
          ctrl_held
        );
        break;
      }
      case RisqOrderType.OrderType_UnitAttackZone: {
        if (!this.hovered_zone) {
          return;
        }
        this.addUnitOrder(
          RisqOrderType.OrderType_UnitAttackZone,
          units.map((u) => u.internal_id),
          this.hovered_zone.coordinate_key,
          ctrl_held
        );
        break;
      }
      case RisqOrderType.OrderType_UnitGarrison: {
        const target = this.hovered_zone?.building;
        if (!target) {
          return;
        }
        this.addUnitOrder(
          RisqOrderType.OrderType_UnitGarrison,
          units.map((u) => u.internal_id),
          target.internal_id,
          ctrl_held
        );
        break;
      }
      case RisqOrderType.OrderType_UnitGather:
        if (!this.hovered_zone) {
          return;
        }
        this.addUnitOrder(
          RisqOrderType.OrderType_UnitGather,
          units.filter((u) => u.unit_id === 1).map((u) => u.internal_id),
          this.hovered_zone.coordinate_key,
          ctrl_held
        );
        break;
      case RisqOrderType.OrderType_UnitBuild:
        if (!this.hovered_zone || !this.armed_building) {
          return;
        }
        const group_order: RisqFrontendOrder = {
          player_id: this.player_id,
          order_type: RisqOrderType.OrderType_UnitBuild,
          subjects: units.filter((u) => u.unit_id === 1).map((u) => u.internal_id),
          target_id: cantorPair(this.armed_building.id, this.hovered_zone.coordinate_key),
          clear_previous_orders: !ctrl_held,
        };
        this.orders_model.add(group_order);
        this.local_foundations.set(this.hovered_zone.coordinate_key, {
          coordinate_key: this.hovered_zone.coordinate_key,
          building_id: this.armed_building.id,
          display_name: this.armed_building.display_name,
          order: group_order,
        });
        break;
      case RisqOrderType.OrderType_UnitRepair:
        if (!this.hovered_zone?.building) {
          return;
        }
        this.addUnitOrder(
          RisqOrderType.OrderType_UnitRepair,
          units.filter((u) => u.unit_id === 1).map((u) => u.internal_id),
          this.hovered_zone.building.internal_id,
          ctrl_held
        );
        break;
      case RisqOrderType.OrderType_UnitRenew:
        if (!this.hovered_zone?.building) {
          return;
        }
        this.addUnitOrder(
          RisqOrderType.OrderType_UnitRenew,
          units.filter((u) => u.unit_id === 1).map((u) => u.internal_id),
          this.hovered_zone.building.internal_id,
          ctrl_held
        );
        break;
      default:
        this.disarmOrder();
        return;
    }
    this.disarmOrder();
  }

  private mouseup(e: MouseEvent) {
    const armed_before = this.armed_order;
    const armed_callback_before = this.armed_button_callback;
    for (const component of this.canvas_components) {
      component.mouseup(e);
    }
    if (this.dragging_selection) {
      const dragged =
        Math.hypot(this.drag_current.x - this.drag_start.x, this.drag_current.y - this.drag_start.y) >
        DRAG_SELECT_THRESHOLD;
      this.dragging_selection = false;
      if (dragged) {
        this.finalizeSelectionDrag();
        return;
      }
    }
    const space = this.hovered_space;
    const zone = this.hovered_zone;
    if (!!space && space.visibility > 0 && armed_before === RisqOrderType.NONE) {
      if (!!zone) {
        if (space.visibility > 0 && this.draw_detail === DrawRisqSpaceDetail.ZONE_DETAILS) {
          let open_zone = true;
          for (const [i, part] of zone.hovered_data.entries()) {
            if (part.clicked && part.hovered) {
              open_zone = false;
              if (i === 0) {
                const target_id = zone.resource?.internal_id ?? zone.building?.internal_id;
                const current = this.currentSingleSelection();
                if (
                  target_id !== undefined &&
                  current &&
                  (current.kind === 'building' || current.kind === 'resource') &&
                  current.id === target_id
                ) {
                  this.left_panel.close();
                } else if (!!zone.resource) {
                  this.left_panel.openPanel(
                    { data_type: LeftPanelDataType.RESOURCE, data: zone.resource },
                    space.visibility
                  );
                } else if (!!zone.building) {
                  const canonical_building =
                    this.game?.players[zone.building.player_id]?.buildings.get(zone.building.internal_id) ??
                    zone.building;
                  this.left_panel.openPanel(
                    { data_type: LeftPanelDataType.BUILDING, data: canonical_building },
                    space.visibility
                  );
                } else {
                  const local_foundation = this.local_foundations.get(zone.coordinate_key);
                  const server_foundation = this.getPlayer()?.planned_foundations?.get(zone.coordinate_key);
                  if (local_foundation || server_foundation) {
                    if (current && current.kind === 'foundation' && current.id === zone.coordinate_key) {
                      this.left_panel.close();
                    } else {
                      const building_id = local_foundation
                        ? local_foundation.building_id
                        : server_foundation!.building_id;
                      const display_name = local_foundation
                        ? local_foundation.display_name
                        : server_foundation!.display_name;
                      this.left_panel.openPanel(
                        {
                          data_type: LeftPanelDataType.FOUNDATION,
                          data: {
                            coordinate_key: zone.coordinate_key,
                            building_id,
                            player_id: this.player_id,
                            is_local: !!local_foundation,
                            display_name: `Planned ${display_name}`,
                            zone,
                          },
                        },
                        space.visibility
                      );
                    }
                  }
                }
              } else if (e.detail >= 2) {
                const target = zone.unit_slots?.[i - 1]?.[0];
                if (target) {
                  const ids =
                    e.detail >= 3
                      ? this.unitsOnScreenOfType(target.player_id, target.unit_id)
                      : this.unitsInZoneOfType(zone, target.player_id, target.unit_id);
                  this.openUnitTypeSelection(
                    target.player_id,
                    ids,
                    e.detail >= 3 ? undefined : space,
                    space.visibility
                  );
                }
              } else {
                const slot_groups = zone.unit_slots?.[i - 1] ?? [];
                const slot_unit_ids = slot_groups.flatMap((t) => [...t.units]);
                const current = this.currentSingleSelection();
                if (e.ctrlKey) {
                  const selected = this.selectedUnitIds();
                  const deselect = slot_unit_ids.length > 0 && slot_unit_ids.every((id) => selected.has(id));
                  const groups = new Map<string, UnitByTypeData>();
                  for (const g of this.currentUnitTypeGroups()) {
                    groups.set(`${g.player_id}:${g.unit_id}`, { ...g, units: new Set(g.units) });
                  }
                  for (const g of slot_groups) {
                    const key = `${g.player_id}:${g.unit_id}`;
                    const target = groups.get(key) ?? { ...g, units: new Set<number>() };
                    for (const id of g.units) {
                      deselect ? target.units.delete(id) : target.units.add(id);
                    }
                    target.units.size > 0 ? groups.set(key, target) : groups.delete(key);
                  }
                  const units_by_player = new Map<number, UnitByTypeData[]>();
                  for (const g of groups.values()) {
                    if (!units_by_player.has(g.player_id)) {
                      units_by_player.set(g.player_id, []);
                    }
                    units_by_player.get(g.player_id)!.push(g);
                  }
                  this.left_panel.openPanel(
                    { data_type: LeftPanelDataType.UNITS, data: { space, units_by_player } },
                    space.visibility
                  );
                } else if (slot_unit_ids.length === 1 && current?.kind === 'unit' && current.id === slot_unit_ids[0]) {
                  this.left_panel.close();
                } else {
                  const units_by_player = new Map<number, UnitByTypeData[]>();
                  for (const t of slot_groups) {
                    if (!units_by_player.has(t.player_id)) {
                      units_by_player.set(t.player_id, []);
                    }
                    units_by_player.get(t.player_id)!.push(t);
                  }
                  this.left_panel.openPanel(
                    { data_type: LeftPanelDataType.UNITS, data: { space, units_by_player } },
                    space.visibility
                  );
                }
              }
              break;
            }
          }
          if (open_zone && zone.clicked) {
            this.left_panel.openPanel(
              {
                data_type: LeftPanelDataType.ZONE,
                data: {
                  space,
                  zone,
                },
              },
              space.visibility
            );
          }
        }
        zone.clicked = false;
        for (const part of zone.hovered_data) {
          part.clicked = false;
        }
      } else if (space.clicked && space.visibility > 0 && this.draw_detail !== DrawRisqSpaceDetail.ZONE_DETAILS) {
        this.left_panel.openPanel({ data_type: LeftPanelDataType.SPACE, data: space }, space.visibility);
      }
      space.clicked = false;
    } else if (armed_before === RisqOrderType.NONE && !this.left_panel.isHovering() && !this.right_panel.isHovering()) {
      this.left_panel.close();
    }
    if (
      armed_before !== RisqOrderType.NONE &&
      this.armed_order === armed_before &&
      this.armed_button_callback === armed_callback_before
    ) {
      this.disarmOrder();
    }
  }

  private finalizeSelectionDrag() {
    this.dragging_selection = false;
    const player = this.getPlayer();
    if (!player) {
      return;
    }
    const zone_view = this.draw_detail === DrawRisqSpaceDetail.ZONE_DETAILS;
    const found_ids: number[] = [];
    for (const unit of player.units.values()) {
      if (unit.garrisoned_in !== undefined) {
        continue;
      }
      const anchor = canvasToScreen(
        this.orderPoint(unit.space_coordinate, this.unitAnchorOffset(unit), zone_view),
        this.last_transform
      );
      if (pointInRect(anchor, this.drag_start, this.drag_current)) {
        found_ids.push(unit.internal_id);
      }
    }
    if (found_ids.length === 0) {
      if (!this.drag_additive) {
        this.left_panel.close();
      }
      return;
    }
    const selected_ids = this.drag_additive ? new Set([...this.selectedUnitIds(), ...found_ids]) : new Set(found_ids);
    this.left_panel.openPanel(
      {
        data_type: LeftPanelDataType.UNITS_BY_TYPE,
        data: { units: groupUnitsByType(player.units, [...selected_ids]) },
      },
      RisqVisibilityLevel.SPY
    );
  }

  canvasToCoordinate(canvas: Point2D, board_size: number): Point2D {
    const cy = (canvas.y - 0.25 * this.hex_r) / (1.5 * this.hex_r) - board_size - 0.5;
    return {
      x: canvas.x / (1.732 * this.hex_r) - 0.5 * cy - board_size - 0.5,
      y: cy,
    };
  }

  private coordinateToCanvas(coordinate: Point2D): Point2D {
    if (!this.game) {
      return { x: 0, y: 0 };
    }
    return {
      x: 1.732 * (coordinate.x + 0.5 * coordinate.y + this.game.board_size + 0.5) * this.hex_r,
      y: 1.5 * (coordinate.y + this.game.board_size + 0.5) * this.hex_r + 0.25 * this.hex_r,
    };
  }

  private removeHoveredFlags() {
    if (!this.hovered_space) {
      return;
    }
    this.hovered_space.hovered = false;
    for (const neighbor of this.getBoardNeighbors(this.hovered_space)) {
      neighbor.hovered_neighbor = false;
    }
    for (const row of this.getBoardRows(this.hovered_space)) {
      row.hovered_row = false;
    }
  }

  private updateHoveredFlags() {
    if (!this.hovered_space) {
      return;
    }
    this.hovered_space.hovered = true;
    for (const neighbor of this.getBoardNeighbors(this.hovered_space)) {
      neighbor.hovered_neighbor = true;
    }
    for (const row of this.getBoardRows(this.hovered_space)) {
      row.hovered_row = true;
    }
  }

  private getBoardNeighbors(space: RisqSpace): RisqSpace[] {
    if (!this.game) {
      return [];
    }
    const neighbors: RisqSpace[] = [];
    for (const neighbor of hexagonalBoardNeighbors(space.coordinate, this.game.board_size)) {
      const index = coordinateToIndex(this.game.board_size, neighbor);
      const space = getSpace(this.game, index);
      if (!!space) {
        neighbors.push(space);
      }
    }
    return neighbors;
  }

  private getBoardRows(space: RisqSpace): RisqSpace[] {
    if (!this.game) {
      return [];
    }
    const rows: RisqSpace[] = [];
    for (const neighbor of hexagonalBoardRows(space.coordinate, this.game.board_size)) {
      const index = coordinateToIndex(this.game.board_size, neighbor);
      const space = getSpace(this.game, index);
      if (!!space) {
        rows.push(space);
      }
    }
    return rows;
  }

  updateDialogComponent(update: UpdateMessage): HTMLElement {
    const update_el = document.createElement('div');
    update_el.innerText = `ID: ${update.update_id}, Kind: ${update.kind}, data: ${JSON.stringify(update.content)}`;
    return update_el;
  }

  async toggleSubmitOrdersButton() {
    const player = this.getPlayer();
    if (this.toggling_submit_orders_button || !this.givingOrders() || !player) {
      this.toggling_submit_orders_button = false;
      return;
    }
    this.toggling_submit_orders_button = true;
    if (player.orders_submitted) {
      this.right_panel.unsubmittingOrders();
      const game_update = createMessage(`player-${this.player_id}`, 'game-update', '', 'unsubmit-orders');
      this.dispatchEvent(
        new CustomEvent('game_update', {
          detail: game_update,
          bubbles: true,
        })
      );
    } else {
      this.right_panel.submittingOrders();
      const game_update = createMessage(
        `player-${this.player_id}`,
        'game-update',
        JSON.stringify({ orders: this.orders_model.pendingOrders() }),
        'submit-orders'
      );
      this.dispatchEvent(
        new CustomEvent('game_update', {
          detail: game_update,
          bubbles: true,
        })
      );
      this.orders_submitted_times++;
    }
  }

  givingOrders(): boolean {
    return this.game?.giving_orders ?? false;
  }

  ordersSubmittedTimes(): number {
    return this.orders_submitted_times;
  }
}

customElements.define('dwg-risq', DwgRisq);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-risq': DwgRisq;
  }
}
