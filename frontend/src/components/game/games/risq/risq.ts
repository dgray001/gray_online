import { DwgElement } from '../../../dwg_element';
import type { UpdateMessage } from '../../data_models';
import { drawArrow, drawCircle } from '../../util/canvas_util';
import type { BoardTransformData, DwgCanvasBoard } from '../../util/canvas_board/canvas_board';
import type { Point2D } from '../../util/objects2d';
import {
  addPoint2D,
  equalsPoint2D,
  hexagonalBoardNeighbors,
  hexagonalBoardRows,
  multiplyPoint2D,
  roundAxialCoordinate,
  subtractPoint2D,
} from '../../util/objects2d';
import type { DwgGame } from '../../game';
import { DEV, createLock, isTypingInInput } from '../../../../scripts/util';
import { ColorRGB } from '../../../../scripts/color_rgb';

import html from './risq.html';
import type { GameRisq, GameRisqFromServer, RisqFrontendOrder, RisqPlayer, RisqSpace, RisqZone } from './risq_data';
import { RisqOrderType, RisqResourceType, RisqVisibilityLevel, serverToGameRisq } from './risq_data';
import { cantorPair, coordinateToIndex, getSpace, invertBuildKey, invertPair, invertZoneKey } from './risq_coordinates';
import type { StartTurnData, SubmittedOrdersData, UnsubmittedOrdersData } from './risq_updates';
import {
  BUILD_CURSOR_SIZE,
  buildOrderCursorKey,
  cursorImageForOrderType,
  DEFAULT_CURSOR_IMAGE,
  drawBuildOrderCursor,
} from './risq_cursor';
import { buildingImage } from './risq_buildings';
import { RisqImageCache } from './risq_image_cache';
import { RisqRightPanel } from './canvas_components/right_panel/right_panel';
import type { DrawRisqSpaceConfig } from './risq_space';
import { DrawRisqSpaceDetail, drawRisqSpace } from './risq_space';
import { RisqLeftPanel } from './canvas_components/left_panel/left_panel';
import { RisqOrdersModel } from './risq_orders';
import { groupUnitsByType, resolveHoveredZones, unhoverRisqZone, zoneCenterOffset } from './risq_zone';
import { RisqViewMode, nextViewMode } from './risq_view_mode';
import type {
  EconomicUnitsData,
  MilitaryUnitsData,
  UnitData,
  UnitsByTypeData,
} from './canvas_components/left_panel/left_panel_data';
import { LeftPanelDataType } from './canvas_components/left_panel/left_panel_data';

import './risq.scss';
import '../../util/canvas_board/canvas_board';
import '../../../dialog_box/confirm_dialog/confirm_dialog';
import { DialogSize } from '../../../dialog_box/dialog_box';
import { createMessage } from '../../../lobby/data_models';

const DEFAULT_HEXAGON_RADIUS = 60;

const DRAW_CENTER_DOT = false;

const DRAW_ORDER_ARROWS = false;

export class DwgRisq extends DwgElement {
  private board!: DwgCanvasBoard;

  private game?: GameRisq;
  private player_id: number = -1;
  private hex_r = DEFAULT_HEXAGON_RADIUS;
  private hex_a = 0.5 * 1.732 * DEFAULT_HEXAGON_RADIUS;
  private canvas_center: Point2D = { x: 0, y: 0 };
  private last_transform: BoardTransformData = {
    view: { x: 0, y: 0 },
    scale: 1,
  };
  private canvas_size: DOMRect = DOMRect.fromRect();
  private mouse_canvas: Point2D = { x: 0, y: 0 };
  private mouse_coordinate: Point2D = { x: 0, y: 0 };
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
  private armed_building_id = 0;
  private armed_button_callback?: () => void;
  private ctrl_held = false;
  private shift_held = false;
  private alt_held = false;
  // control groups 1-10 ('0' is group 10)
  private control_groups = new Map<number, { kind: 'unit' | 'building'; ids: number[] }>();
  private orders_model = new RisqOrdersModel(() => this.ordersChanged());

  private handleKeydown = (e: KeyboardEvent) => {
    if (isTypingInInput()) {
      return;
    }
    this.ctrl_held = e.ctrlKey;
    this.shift_held = e.shiftKey;
    this.alt_held = e.altKey;
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

  private handleKeyup = (e: KeyboardEvent) => {
    this.ctrl_held = e.ctrlKey;
    this.shift_held = e.shiftKey;
    this.alt_held = e.altKey;
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

  constructor() {
    super();
    this.html_string = html;
    this.configureElement('board');
  }

  override disconnectedCallback(): void {
    super.disconnectedCallback();
    document.body.removeEventListener('keydown', this.handleKeydown);
    document.body.removeEventListener('keyup', this.handleKeyup);
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
    // TODO: ability to get image variations (player colors on image??)
    if (!icon) {
      return this.createIcon(name);
    }
    return icon;
  }

  async initialize(abstract_game: DwgGame, game: GameRisqFromServer): Promise<void> {
    this.player_id = abstract_game.isPlayer() ? abstract_game.playerId() : -1;
    abstract_game.setPadding('0px');
    this.setNewGameData(game);
    document.body.addEventListener('keydown', this.handleKeydown);
    document.body.addEventListener('keyup', this.handleKeyup);
    const board_size: Point2D = {
      x: 1.732 * this.hex_r * (2 * game.board_size + 1),
      y: 1.5 * this.hex_r * (2 * game.board_size + 1) + 0.5 * this.hex_r,
    };
    this.board
      .initialize({
        board_size,
        max_scale: 1,
        fill_space: true,
        allow_side_move: false,
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

  getOrdersModel(): RisqOrdersModel {
    return this.orders_model;
  }

  private ordersChanged() {
    this.right_panel.dataRefreshed();
    this.left_panel.dataRefreshed();
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
      const view = this.coordinateToCanvas(building.space_coordinate, this.last_transform.scale ?? 1);
      this.board.setView(subtractPoint2D(view, this.canvas_center));
      return;
    }
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
      this.toggleRightPanel(this.right_panel.isOpen());
    });
  }

  canvasSize(): DOMRect {
    return this.canvas_size;
  }

  drawDetail(): DrawRisqSpaceDetail {
    return this.draw_detail;
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
    this.orders_model.setSubmitted(this.getPlayer()?.active_orders ?? []);
    this.right_panel.dataRefreshed();
    this.left_panel.dataRefreshed();
  }

  private async applyStartTurn(data: StartTurnData) {
    if (this.player_id > -1) {
      this.orders_submitted_times = 0;
    }
    this.clearSelection();
    this.setNewGameData(data.game);
  }

  private clearSelection() {
    this.left_panel.close();
    this.disarmOrder();
  }

  private async applySubmittedOrders(data: SubmittedOrdersData) {
    if (data.player_id === this.player_id) {
      this.toggling_submit_orders_button = false;
    }
    this.setNewGameData(data.game);
  }

  private async applyUnsubmittedOrders(data: UnsubmittedOrdersData) {
    if (data.player_id === this.player_id) {
      this.toggling_submit_orders_button = false;
    }
    this.setNewGameData(data.game);
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
    const inset_offset = 0.25; // this determines how the inset rect (for summaries) is constructed
    const inset_w = 2 * this.hex_a * (1 - inset_offset);
    const inset_h = this.hex_r * (1 + inset_offset);
    const inset_row = inset_h / 4 - 4;
    this.draw_detail = this.getDrawDetail(transform.scale);
    const draw_config: DrawRisqSpaceConfig = {
      hex_r: this.hex_r,
      inset_w,
      inset_h,
      inset_row,
      draw_detail: this.draw_detail,
      view_mode: this.view_mode,
    };
    // draw spaces
    for (const row of this.game.spaces) {
      for (const space of row) {
        space.center = this.coordinateToCanvas(space.coordinate, transform.scale);
        if (
          space.center.x + this.hex_a < transform.view.x / transform.scale ||
          space.center.x - this.hex_a > (transform.view.x + this.canvas_size.width) / transform.scale ||
          space.center.y + this.hex_r < transform.view.y / transform.scale ||
          space.center.y - this.hex_r > (transform.view.y + this.canvas_size.height) / transform.scale
        ) {
          continue;
        }
        drawRisqSpace(ctx, this, space, draw_config);
      }
    }
    this.drawSelectedUnitOrders(ctx);
    // draw panels
    this.right_panel.draw(ctx, transform, dt);
    this.left_panel.draw(ctx, transform, dt);
    // draw red dot
    if (DRAW_CENTER_DOT && DEV) {
      ctx.fillStyle = 'red';
      ctx.strokeStyle = 'transparent';
      const vis_center = multiplyPoint2D(1 / transform.scale, addPoint2D(this.canvas_center, transform.view));
      drawCircle(ctx, vis_center, 6 / transform.scale);
    }
  }

  private drawSelectedUnitOrders(ctx: CanvasRenderingContext2D) {
    if (!DRAW_ORDER_ARROWS) {
      return; // TODO: order arrows are positioned wrong; reimplement next commit
    }
    const data = this.left_panel.getData();
    if (data?.data_type !== LeftPanelDataType.UNIT) {
      return;
    }
    const unit = data.data;
    const orders = this.orders_model.effectiveForSubject(unit.internal_id);
    if (!orders.length) {
      return;
    }
    let from = addPoint2D(
      this.coordinateToCanvas(unit.space_coordinate, this.last_transform.scale),
      zoneCenterOffset(unit.zone_coordinate, this.hex_r)
    );
    ctx.strokeStyle = 'rgba(255, 225, 0, 0.9)';
    ctx.fillStyle = 'rgba(255, 225, 0, 0.9)';
    ctx.lineWidth = 2;
    ctx.setLineDash([8, 5]);
    for (const order of orders) {
      const to = this.orderTargetPoint(order);
      if (!to) {
        continue;
      }
      drawArrow(ctx, from, to, 10);
      from = to;
    }
    ctx.setLineDash([]);
  }

  private orderTargetPoint(order: RisqFrontendOrder): Point2D | undefined {
    if (!this.game) {
      return undefined;
    }
    let target_space: Point2D;
    let target_zone: Point2D | undefined;
    switch (order.order_type) {
      case RisqOrderType.OrderType_UnitMoveSpace:
        target_space = invertPair(order.target_id);
        break;
      case RisqOrderType.OrderType_UnitMoveZone:
      case RisqOrderType.OrderType_UnitGather: {
        const decoded = invertZoneKey(order.target_id);
        target_space = decoded.space;
        target_zone = decoded.zone;
        break;
      }
      case RisqOrderType.OrderType_UnitBuild: {
        const decoded = invertBuildKey(order.target_id);
        target_space = decoded.space;
        target_zone = decoded.zone;
        break;
      }
      default:
        return undefined;
    }
    const p = this.coordinateToCanvas(target_space, this.last_transform.scale);
    return target_zone ? addPoint2D(p, zoneCenterOffset(target_zone, this.hex_r)) : p;
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
    if (this.left_panel.isHovering()) {
      this.left_panel.scroll(dy, mode);
      return true;
    }
    if (this.right_panel.isHovering()) {
      this.right_panel.scroll(dy, mode);
      return true;
    }
    return false;
  }

  private mousemove(m: Point2D, transform: BoardTransformData) {
    if (!this.game) {
      return;
    }
    this.draw_detail = this.getDrawDetail(transform.scale);
    this.mouse_canvas = m;
    if (!!this.hovered_space) {
      this.hovered_space.center = this.coordinateToCanvas(this.hovered_space.coordinate, transform.scale);
    }
    const hovered_other_component = [
      this.right_panel.mousemove(m, transform),
      this.left_panel.mousemove(m, transform),
    ].some((b) => !!b);
    this.mouse_coordinate = this.canvasToCoordinate(m, transform.scale, this.game.board_size);
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
      this.updateCursor();
      return;
    }
    this.removeHoveredFlags();
    if (!!this.hovered_space) {
      this.hovered_space.clicked = false;
      if (!!this.hovered_zone) {
        unhoverRisqZone(this.hovered_zone);
        this.hovered_zone = undefined;
      }
      resolve_zones.call(this);
    }
    this.hovered_space = new_hovered_space;
    this.updateHoveredFlags();
    this.updateCursor();
  }

  private draggingCallback() {
    this.removeHoveredFlags();
    if (!!this.hovered_space) {
      this.hovered_space.clicked = false;
    }
  }

  private mouseleave() {
    if (!!this.hovered_space) {
      this.hovered_space.hovered = false;
      this.hovered_space.clicked = false;
      this.hovered_space = undefined;
    }
  }

  // returns false if mousedown event should initiate dragging
  private mousedown(e: MouseEvent): boolean {
    if ([this.right_panel.mousedown(e), this.left_panel.mousedown(e)].some((b) => !!b)) {
      return true;
    }
    // left click
    if (e.button === 0) {
      if (!!this.hovered_space && this.hovered_space.visibility > 0) {
        this.hovered_space.clicked = true;
        if (this.draw_detail !== DrawRisqSpaceDetail.ZONE_DETAILS) {
          return false;
        }
        if (!!this.hovered_zone) {
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
          this.unitOrder(left_panel_data);
          break;
        case LeftPanelDataType.UNITS_BY_TYPE:
        case LeftPanelDataType.ECONOMIC_UNITS:
        case LeftPanelDataType.MILITARY_UNITS:
          this.unitGroupOrder(left_panel_data);
          break;
        default:
          break;
      }
    }
    // only drag on left click
    // TODO: implement rotate on right click
    return e.button !== 0;
  }

  armOrder(order_type: RisqOrderType, on_disarm: () => void, building_id = 0) {
    this.armed_button_callback?.();
    this.armed_order = order_type;
    this.armed_building_id = building_id;
    this.armed_button_callback = on_disarm;
    this.updateCursor();
  }

  getArmedOrder(): RisqOrderType {
    return this.armed_order;
  }

  disarmOrder() {
    this.armed_button_callback?.();
    this.armed_order = RisqOrderType.NONE;
    this.armed_building_id = 0;
    this.armed_button_callback = undefined;
    this.updateCursor();
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
    this.updateResourceSpending();
    this.left_panel.dataRefreshed();
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
    this.updateResourceSpending();
    this.left_panel.dataRefreshed();
  }

  confirmDeleteUnit(internal_id: number) {
    const dialog = document.createElement('dwg-confirm-dialog');
    dialog.setData({ question: 'Are you sure you want to delete this unit?', size: DialogSize.SMALL });
    dialog.addEventListener('confirmed', () => {
      this.deleteUnit(internal_id);
    });
    this.appendChild(dialog);
  }

  private deleteUnit(internal_id: number) {
    if (!this.canGiveOrders()) {
      return;
    }
    this.orders_model.add({
      player_id: this.player_id,
      order_type: RisqOrderType.OrderType_UnitDelete,
      subjects: [internal_id],
      target_id: 0,
      clear_previous_orders: true,
    });
  }

  stopUnit(internal_id: number) {
    if (!this.canGiveOrders()) {
      return;
    }
    this.orders_model.cancelForSubject(internal_id);
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
      if (
        order.order_type !== RisqOrderType.OrderType_BuildingCreate &&
        order.order_type !== RisqOrderType.OrderType_BuildingResearch
      ) {
        continue;
      }
      for (const subject_id of order.subjects) {
        const cost = player.buildings.get(subject_id)?.produces.find((p) => p.id === order.target_id)?.cost;
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

  private atSelectedUnitPosition(zone_valid: boolean): boolean {
    if (this.ctrl_held || !this.hovered_space) {
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

  private buildTargetValid(): boolean {
    return (
      this.left_panel.isVillager() && this.isZoneValid() && !this.hovered_zone?.resource && !this.hovered_zone?.building
    );
  }

  private resolveActiveOrderType(): RisqOrderType {
    if (!this.left_panel.getData() || !this.left_panel.isOrderable() || !this.canGiveOrders() || !this.hovered_space) {
      return RisqOrderType.NONE;
    }
    const is_unit = this.left_panel.isUnit();
    const is_villager = this.left_panel.isVillager();
    const zone_valid = this.isZoneValid();
    switch (this.getArmedOrder()) {
      case RisqOrderType.OrderType_UnitMoveSpace:
      case RisqOrderType.OrderType_UnitMoveZone:
        if (is_unit && !this.atSelectedUnitPosition(zone_valid)) {
          return zone_valid ? RisqOrderType.OrderType_UnitMoveZone : RisqOrderType.OrderType_UnitMoveSpace;
        }
        break;
      case RisqOrderType.OrderType_UnitGather:
        if (is_villager && zone_valid && !!this.hovered_zone?.resource) {
          return RisqOrderType.OrderType_UnitGather;
        }
        break;
      case RisqOrderType.OrderType_UnitBuild:
        if (is_villager && zone_valid && !this.hovered_zone?.resource && !this.hovered_zone?.building) {
          return RisqOrderType.OrderType_UnitBuild;
        }
        break;
      default:
        break;
    }
    if (is_unit) {
      if (is_villager && zone_valid && !!this.hovered_zone?.resource && this.hovered_zone.hovered_data[0]?.hovered) {
        return RisqOrderType.OrderType_UnitGather;
      }
      if (this.atSelectedUnitPosition(zone_valid)) {
        return RisqOrderType.NONE;
      }
      return zone_valid ? RisqOrderType.OrderType_UnitMoveZone : RisqOrderType.OrderType_UnitMoveSpace;
    }
    return RisqOrderType.NONE;
  }

  private updateCursor() {
    if (this.getArmedOrder() === RisqOrderType.OrderType_UnitBuild && this.armed_building_id) {
      const building_icon = this.getIcon(buildingImage(this.armed_building_id));
      const build_icon = this.getIcon(cursorImageForOrderType(RisqOrderType.OrderType_UnitBuild));
      const valid = this.buildTargetValid();
      const url = this.image_cache.getCursorUrl(
        buildOrderCursorKey(this.armed_building_id, valid),
        BUILD_CURSOR_SIZE,
        [building_icon, build_icon],
        (ctx) => drawBuildOrderCursor(ctx, build_icon, building_icon, valid)
      );
      if (url) {
        this.board.setCursorUrl(url);
        return;
      }
    }
    const active_order = this.resolveActiveOrderType();
    this.board.setCursor(cursorImageForOrderType(active_order));
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

  private unitOrder(data: UnitData) {
    if (!this.hovered_space) {
      return;
    }
    // TODO: implement attack vs just move
    // TODO: implement if holding the shift key
    switch (this.resolveActiveOrderType()) {
      case RisqOrderType.OrderType_UnitMoveSpace:
        this.orders_model.add({
          player_id: this.player_id,
          order_type: RisqOrderType.OrderType_UnitMoveSpace,
          subjects: [data.data.internal_id],
          target_id: this.hovered_space.coordinate_key,
          clear_previous_orders: !this.ctrl_held,
        });
        break;
      case RisqOrderType.OrderType_UnitMoveZone:
        if (!this.hovered_zone) {
          return;
        }
        this.orders_model.add({
          player_id: this.player_id,
          order_type: RisqOrderType.OrderType_UnitMoveZone,
          subjects: [data.data.internal_id],
          target_id: this.hovered_zone.coordinate_key,
          clear_previous_orders: !this.ctrl_held,
        });
        break;
      case RisqOrderType.OrderType_UnitGather:
        if (!this.hovered_zone) {
          return;
        }
        this.orders_model.add({
          player_id: this.player_id,
          order_type: RisqOrderType.OrderType_UnitGather,
          subjects: [data.data.internal_id],
          target_id: this.hovered_zone.coordinate_key,
          clear_previous_orders: !this.ctrl_held,
        });
        break;
      case RisqOrderType.OrderType_UnitBuild:
        if (!this.hovered_zone) {
          return;
        }
        this.orders_model.add({
          player_id: this.player_id,
          order_type: RisqOrderType.OrderType_UnitBuild,
          subjects: [data.data.internal_id],
          target_id: cantorPair(this.armed_building_id, this.hovered_zone.coordinate_key),
          clear_previous_orders: !this.ctrl_held,
        });
        break;
      default:
        return;
    }
    this.disarmOrder();
  }

  // Returns whether the given unit is already sitting at the current hover target (never true while ctrl is held)
  private isUnitAtHoverTarget(internal_id: number, zone_valid: boolean): boolean {
    if (this.ctrl_held || !this.hovered_space) {
      return false;
    }
    const unit = this.getPlayer()?.units.get(internal_id);
    if (!unit || !equalsPoint2D(this.hovered_space.coordinate, unit.space_coordinate)) {
      return false;
    }
    return zone_valid ? equalsPoint2D(this.hovered_zone?.coordinate, unit.zone_coordinate) : true;
  }

  private unitGroupOrder(data: UnitsByTypeData | EconomicUnitsData | MilitaryUnitsData) {
    if (!this.hovered_space) {
      return;
    }
    const units = data.data.units.flatMap((u) =>
      [...u.units].map((internal_id) => ({ unit_id: u.unit_id, internal_id }))
    );
    // TODO: implement attack vs just move
    // TODO: implement if holding the shift key
    switch (this.resolveActiveOrderType()) {
      case RisqOrderType.OrderType_UnitMoveSpace: {
        const subjects = units.filter((u) => !this.isUnitAtHoverTarget(u.internal_id, false)).map((u) => u.internal_id);
        if (subjects.length === 0) {
          break;
        }
        this.orders_model.add({
          player_id: this.player_id,
          order_type: RisqOrderType.OrderType_UnitMoveSpace,
          subjects,
          target_id: this.hovered_space.coordinate_key,
          clear_previous_orders: !this.ctrl_held,
        });
        break;
      }
      case RisqOrderType.OrderType_UnitMoveZone: {
        if (!this.hovered_zone) {
          return;
        }
        const subjects = units.filter((u) => !this.isUnitAtHoverTarget(u.internal_id, true)).map((u) => u.internal_id);
        if (subjects.length === 0) {
          break;
        }
        this.orders_model.add({
          player_id: this.player_id,
          order_type: RisqOrderType.OrderType_UnitMoveZone,
          subjects,
          target_id: this.hovered_zone.coordinate_key,
          clear_previous_orders: !this.ctrl_held,
        });
        break;
      }
      case RisqOrderType.OrderType_UnitGather:
        if (!this.hovered_zone) {
          return;
        }
        this.orders_model.add({
          player_id: this.player_id,
          order_type: RisqOrderType.OrderType_UnitGather,
          subjects: units.filter((u) => u.unit_id === 1).map((u) => u.internal_id),
          target_id: this.hovered_zone.coordinate_key,
          clear_previous_orders: !this.ctrl_held,
        });
        break;
      case RisqOrderType.OrderType_UnitBuild:
        if (!this.hovered_zone) {
          return;
        }
        this.orders_model.add({
          player_id: this.player_id,
          order_type: RisqOrderType.OrderType_UnitBuild,
          subjects: units.filter((u) => u.unit_id === 1).map((u) => u.internal_id),
          target_id: cantorPair(this.armed_building_id, this.hovered_zone.coordinate_key),
          clear_previous_orders: !this.ctrl_held,
        });
        break;
      default:
        return;
    }
    this.disarmOrder();
  }

  private mouseup(e: MouseEvent) {
    const armed_before = this.armed_order;
    this.right_panel.mouseup(e);
    this.left_panel.mouseup(e);
    if (!!this.hovered_space) {
      if (!!this.hovered_zone) {
        if (this.hovered_space.visibility > 0 && this.draw_detail === DrawRisqSpaceDetail.ZONE_DETAILS) {
          let open_zone = true;
          for (const [i, part] of this.hovered_zone.hovered_data.entries()) {
            if (part.clicked && part.hovered) {
              open_zone = false;
              switch (i) {
                case 0: // building / resource
                  if (!!this.hovered_zone.resource) {
                    this.left_panel.openPanel(
                      { data_type: LeftPanelDataType.RESOURCE, data: this.hovered_zone.resource },
                      this.hovered_space.visibility
                    );
                  } else if (!!this.hovered_zone.building) {
                    this.left_panel.openPanel(
                      { data_type: LeftPanelDataType.BUILDING, data: this.hovered_zone.building },
                      this.hovered_space.visibility
                    );
                  }
                  break;
                case 1: // economic units
                  this.left_panel.openPanel(
                    {
                      data_type: LeftPanelDataType.UNITS,
                      data: {
                        space: this.hovered_space,
                        units_by_player: this.hovered_zone.economic_units_by_type,
                      },
                    },
                    this.hovered_space.visibility
                  );
                  break;
                case 2: // military units
                  this.left_panel.openPanel(
                    {
                      data_type: LeftPanelDataType.UNITS,
                      data: {
                        space: this.hovered_space,
                        units_by_player: this.hovered_zone.military_units_by_type,
                      },
                    },
                    this.hovered_space.visibility
                  );
                  break;
                default:
                  break;
              }
              break;
            }
          }
          if (open_zone && this.hovered_zone.clicked) {
            this.left_panel.openPanel(
              {
                data_type: LeftPanelDataType.ZONE,
                data: {
                  space: this.hovered_space,
                  zone: this.hovered_zone,
                },
              },
              this.hovered_space.visibility
            );
          }
        }
        this.hovered_zone.clicked = false;
        for (const part of this.hovered_zone.hovered_data) {
          part.clicked = false;
        }
      } else if (
        this.hovered_space.clicked &&
        this.hovered_space.visibility > 0 &&
        this.draw_detail !== DrawRisqSpaceDetail.ZONE_DETAILS
      ) {
        this.left_panel.openPanel(
          { data_type: LeftPanelDataType.SPACE, data: this.hovered_space },
          this.hovered_space.visibility
        );
      }
      this.hovered_space.clicked = false;
    }
    if (armed_before !== RisqOrderType.NONE && this.armed_order === armed_before) {
      this.disarmOrder();
    }
  }

  private canvasToCoordinate(canvas: Point2D, scale: number, board_size: number): Point2D {
    const cy = (canvas.y - 0.25 * this.hex_r - this.canvas_center.y / scale) / (1.5 * this.hex_r) - board_size - 0.5;
    return {
      x: (canvas.x - this.canvas_center.x / scale) / (1.732 * this.hex_r) - 0.5 * cy - board_size - 0.5,
      y: cy,
    };
  }

  private coordinateToCanvas(coordinate: Point2D, scale: number): Point2D {
    if (!this.game) {
      return { x: 0, y: 0 };
    }
    return {
      x:
        1.732 * (coordinate.x + 0.5 * coordinate.y + this.game.board_size + 0.5) * this.hex_r +
        this.canvas_center.x / scale,
      y:
        1.5 * (coordinate.y + this.game.board_size + 0.5) * this.hex_r +
        0.25 * this.hex_r +
        this.canvas_center.y / scale,
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
      this.orders_model.clearPending();
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
