import { DwgElement } from '../../../dwg_element';
import type { UpdateMessage } from '../../data_models';
import { drawArrow, drawCircle, drawLine, drawRect } from '../../util/canvas_util';
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
import { DEV, createLock, isDialogOpen, isTypingInInput } from '../../../../scripts/util';
import { err, log } from '../../../../scripts/log';
import { ColorRGB } from '../../../../scripts/color_rgb';
import { getSettings } from '../../../../scripts/settings_store';

import html from './risq.html';
import type {
  GameRisq,
  GameRisqFromServer,
  RisqBuilding,
  RisqCost,
  RisqFrontendOrder,
  RisqMovePathStep,
  RisqProducible,
  RisqPlayer,
  RisqRegion,
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
  RisqUnitStance,
  RisqUnitType,
  RisqVisibilityLevel,
  canAffordCost,
  canHaveGatherPoint,
  serverToGameRisq,
} from './risq_data';
import type { RisqGatherPoint, RisqResource, RisqTargetCategory } from './risq_data';
import { cantorPair, coordinateToIndex, getSpace, invertBuildKey, invertPair, invertZoneKey } from './risq_coordinates';
import type {
  GatherPointSetData,
  StartTurnData,
  SubmittedOrdersData,
  UnitBehaviorSetData,
  UnsubmittedOrdersData,
} from './risq_updates';
import {
  armedCursorAlpha,
  cursorImageForOrderType,
  DEFAULT_CURSOR_IMAGE,
  resolveBuildCursorUrl,
  resolveMercenaryCursorUrl,
} from './risq_cursor';
import { PLAYER_ICON_SIZE, RisqImageCache } from './risq_image_cache';
import { RisqRightPanel } from './canvas_components/right_panel/right_panel';
import type { DrawRisqSpaceConfig } from './risq_space';
import { DrawRisqSpaceDetail, drawRisqSpace, drawRisqSpaceBorder } from './risq_space';
import { buildRegionLookup, drawRisqRegionBorders, drawRisqRegionLabels } from './risq_region';
import { RisqLeftPanel } from './canvas_components/left_panel/left_panel';
import type { UnitToggleField } from './canvas_components/left_panel/action_button/unit_toggle_button';
import { RisqMinimap } from './canvas_components/minimap/risq_minimap';
import { RisqBottomPanel } from './canvas_components/bottom_panel/bottom_panel';
import { RisqSummaryReportButton } from './canvas_components/bottom_panel/summary_report_button';
import { RisqViewModeButton } from './canvas_components/bottom_panel/view_mode_button';
import { RisqTechTreeButton } from './canvas_components/bottom_panel/tech_tree_button';
import { RisqMercenaryGrid } from './canvas_components/bottom_panel/mercenary_grid';
import type { RisqTurnReport } from './risq_turn_report';
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
  zoneMercenarySlotOffsets,
} from './risq_zone';
import { RisqViewMode, nextViewMode } from './risq_terrain';
import { drawMercenaryGhosts, mercenaryGhostSlots } from './risq_mercenary';
import type {
  BuildingData,
  EconomicUnitsData,
  MilitaryUnitsData,
  UnitData,
  UnitsByTypeData,
} from './canvas_components/left_panel/left_panel_data';
import { LeftPanelDataType } from './canvas_components/left_panel/left_panel_data';
import type { RisqHotkeyLookupEntry } from './risq_hotkeys';
import { RisqHotkeyAction, buildHotkeyLookup, comboFromKeyboardEvent, comboToString } from './risq_hotkeys';

import './risq.scss';
import '../../util/canvas_board/canvas_board';
import '../../../dialog_box/confirm_dialog/confirm_dialog';
import './turn_report_dialog/turn_report_dialog';
import './tech_tree_dialog/tech_tree_dialog';
import { DialogSize } from '../../../dialog_box/dialog_box';
import { createMessage } from '../../../lobby/data_models';

const DEFAULT_HEXAGON_RADIUS = 60;

const DRAW_CENTER_DOT = false;

const DRAG_SELECT_THRESHOLD = 3;

declare interface CanvasBounds {
  min: Point2D;
  max: Point2D;
}

export declare interface LocalRisqFoundation {
  coordinate_key: number;
  building_id: number;
  display_name: string;
  order: RisqFrontendOrder;
}

export class DwgRisq extends DwgElement {
  private board!: DwgCanvasBoard;

  private game?: GameRisq;
  private region_by_space = new Map<number, RisqRegion>();
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
  private armed_mercenary?: RisqProducible;
  private armed_button_callback?: () => void;
  private gather_point_armed = false;
  private building_attack_armed = false;
  private local_foundations = new Map<number, LocalRisqFoundation>();
  // control groups 1-10 ('0' is group 10)
  private control_groups = new Map<number, { kind: 'unit' | 'building'; ids: number[] }>();
  private orders_model = new RisqOrdersModel(() => this.ordersChanged());
  private idle_units: RisqUnit[] = [];
  private last_idle_selected?: number;
  private last_turn_report?: RisqTurnReport;

  private handleKeydown = (e: KeyboardEvent) => {
    if (isTypingInInput() || isDialogOpen()) {
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
    const entry = this.resolveHotkeyEntry(e);
    if (entry?.kind === 'cycle_building') {
      this.cycleBuildingSelection(entry.id);
    }
  };

  private handleKeyup = (e: KeyboardEvent) => {
    if (isTypingInInput() || isDialogOpen()) {
      return;
    }
    const entry = this.resolveHotkeyEntry(e);
    if (!entry) {
      return;
    }
    if (entry.kind === 'action') {
      this.handleHotkeyAction(entry.action);
    } else if (entry.kind === 'create_unit' || entry.kind === 'research_tech' || entry.kind === 'build_building') {
      this.handleHotkeyProducible(entry);
    } else if (entry.kind === 'hire_mercenary') {
      const mercenary = this.getPlayer()?.available_mercenaries.find((m) => m.id === entry.id);
      if (mercenary) {
        this.toggleMercenary(mercenary);
      }
    }
  };

  private resolveHotkeyEntry(e: KeyboardEvent): RisqHotkeyLookupEntry | undefined {
    const lookup = buildHotkeyLookup(getSettings().risq_hotkeys);
    return lookup.get(comboToString(comboFromKeyboardEvent(e)));
  }

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
    background: 'black',
  });
  private bottom_panel = new RisqBottomPanel(
    this,
    { background: 'rgb(222, 184, 135)', left_panel_w: 300, right_panel_w: 300 },
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
          err('Not able to initialize game board');
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

  getLastTurnReport(): RisqTurnReport | undefined {
    return this.last_turn_report;
  }

  reopenLastTurnReport() {
    const player = this.getPlayer();
    if (player && this.last_turn_report) {
      this.openTurnReportDialog(player, this.last_turn_report);
    }
  }

  private openTurnReportDialog(player: RisqPlayer, report: RisqTurnReport) {
    const dialog = document.createElement('dwg-risq-turn-report-dialog');
    dialog.setData({ risq: this, player, report });
    this.appendChild(dialog);
  }

  cycleViewMode() {
    this.view_mode = nextViewMode(this.view_mode);
  }

  private setViewMode(mode: RisqViewMode) {
    this.view_mode = mode;
  }

  private hotkeySubjectUnitIds(): number[] {
    const data = this.left_panel.getData();
    switch (data?.data_type) {
      case LeftPanelDataType.UNIT:
        return [data.data.internal_id];
      case LeftPanelDataType.UNITS_BY_TYPE:
      case LeftPanelDataType.ECONOMIC_UNITS:
      case LeftPanelDataType.MILITARY_UNITS:
        return data.data.units.flatMap((u) => [...u.units]);
      default:
        return [];
    }
  }

  private toggleArmedOrder(order_type: RisqOrderType) {
    if (this.getArmedOrder() === order_type) {
      this.disarmOrder();
    } else {
      this.armOrder(order_type, () => {});
    }
  }

  private toggleUnitFlag(ids: number[], field: UnitToggleField) {
    const player = this.getPlayer();
    const values = ids.map((id) => player?.units.get(id)?.[field]);
    const active = values.length > 0 && values.every((v) => v === true);
    this.setUnitToggle(ids, field, !active);
  }

  private handleHotkeyAction(action: RisqHotkeyAction) {
    const data = this.left_panel.getData();
    const building = data?.data_type === LeftPanelDataType.BUILDING ? data.data : undefined;
    const subject_ids = this.hotkeySubjectUnitIds();
    const zone_view = this.drawDetail() === DrawRisqSpaceDetail.ZONE_DETAILS;
    switch (action) {
      case RisqHotkeyAction.CYCLE_VIEW_MODE:
        this.cycleViewMode();
        break;
      case RisqHotkeyAction.VIEW_MODE_ALL:
        this.setViewMode(RisqViewMode.ALL);
        break;
      case RisqHotkeyAction.VIEW_MODE_RESOURCE:
        this.setViewMode(RisqViewMode.RESOURCE);
        break;
      case RisqHotkeyAction.VIEW_MODE_MILITARY:
        this.setViewMode(RisqViewMode.MILITARY);
        break;
      case RisqHotkeyAction.VIEW_MODE_OWNERSHIP:
        this.setViewMode(RisqViewMode.OWNERSHIP);
        break;
      case RisqHotkeyAction.VIEW_MODE_REGION:
        this.setViewMode(RisqViewMode.REGION);
        break;
      case RisqHotkeyAction.SUMMARY_REPORT:
        if (!this.getLastTurnReport()) {
          this.showMessage('No previous turn report to review', RISQ_MESSAGE_WARNING_COLOR);
        } else {
          this.reopenLastTurnReport();
        }
        break;
      case RisqHotkeyAction.TECH_TREE: {
        const dialog = document.createElement('dwg-risq-tech-tree-dialog');
        dialog.setData({ risq: this });
        this.appendChild(dialog);
        break;
      }
      case RisqHotkeyAction.NEXT_IDLE:
        this.selectNextIdleUnit();
        break;
      case RisqHotkeyAction.SUBMIT_ORDERS:
        this.confirmSubmitOrders();
        break;
      case RisqHotkeyAction.MOVE:
        this.toggleArmedOrder(zone_view ? RisqOrderType.OrderType_UnitMoveZone : RisqOrderType.OrderType_UnitMoveSpace);
        break;
      case RisqHotkeyAction.ATTACK:
        this.toggleArmedOrder(
          zone_view ? RisqOrderType.OrderType_UnitAttackZone : RisqOrderType.OrderType_UnitAttackSpace
        );
        break;
      case RisqHotkeyAction.STOP:
        if (subject_ids.length) {
          this.stopUnit(subject_ids);
        }
        break;
      case RisqHotkeyAction.DELETE:
        if (building) {
          this.confirmDeleteBuilding(building.internal_id);
        } else if (subject_ids.length) {
          this.confirmDeleteUnit(subject_ids);
        }
        break;
      case RisqHotkeyAction.GARRISON:
        if (this.left_panel.isOnlyGarrisoned()) {
          this.ungarrisonUnits(subject_ids);
        } else {
          this.toggleArmedOrder(RisqOrderType.OrderType_UnitGarrison);
        }
        break;
      case RisqHotkeyAction.UNGARRISON:
        if (subject_ids.length) {
          this.ungarrisonUnits(subject_ids);
        }
        break;
      case RisqHotkeyAction.GATHER:
        if (this.left_panel.isOnlyVillagers()) {
          this.toggleArmedOrder(RisqOrderType.OrderType_UnitGather);
        }
        break;
      case RisqHotkeyAction.REPAIR:
        if (this.left_panel.isOnlyVillagers()) {
          this.toggleArmedOrder(RisqOrderType.OrderType_UnitRepair);
        }
        break;
      case RisqHotkeyAction.RENEW:
        if (this.left_panel.isOnlyVillagers()) {
          this.toggleArmedOrder(RisqOrderType.OrderType_UnitRenew);
        }
        break;
      case RisqHotkeyAction.STANCE_AGGRESSIVE:
        if (this.left_panel.isOnlyMilitary()) {
          this.setUnitStance(subject_ids, RisqUnitStance.AGGRESSIVE);
        }
        break;
      case RisqHotkeyAction.STANCE_DEFENSIVE:
        if (this.left_panel.isOnlyMilitary()) {
          this.setUnitStance(subject_ids, RisqUnitStance.DEFENSIVE);
        }
        break;
      case RisqHotkeyAction.STANCE_STAND_GROUND:
        if (this.left_panel.isOnlyMilitary()) {
          this.setUnitStance(subject_ids, RisqUnitStance.STAND_GROUND);
        }
        break;
      case RisqHotkeyAction.STANCE_PASSIVE:
        if (this.left_panel.isOnlyMilitary()) {
          this.setUnitStance(subject_ids, RisqUnitStance.PASSIVE);
        }
        break;
      case RisqHotkeyAction.TOGGLE_INTERRUPT_CURRENT:
        if (this.left_panel.isOnlyMilitary()) {
          this.toggleUnitFlag(subject_ids, 'interrupt_current');
        }
        break;
      case RisqHotkeyAction.TOGGLE_ATTACK_BACK:
        if (this.left_panel.isOnlyMilitary()) {
          this.toggleUnitFlag(subject_ids, 'attack_back');
        }
        break;
      case RisqHotkeyAction.BUILDING_ATTACK:
        if (building) {
          this.attackFromBuilding(building.internal_id);
        }
        break;
      case RisqHotkeyAction.BUILDING_GATHER_POINT:
        if (building) {
          this.toggleBuildingGatherPoint(building.internal_id);
        }
        break;
      case RisqHotkeyAction.BUILDING_DELETE:
        if (building) {
          this.confirmDeleteBuilding(building.internal_id);
        }
        break;
      default:
        break;
    }
  }

  private handleHotkeyProducible(
    entry: Extract<RisqHotkeyLookupEntry, { kind: 'create_unit' | 'research_tech' | 'build_building' }>
  ) {
    const data = this.left_panel.getData();
    const building = data?.data_type === LeftPanelDataType.BUILDING ? data.data : undefined;
    const player = this.getPlayer();
    if (!player) {
      return;
    }
    if (entry.kind === 'create_unit') {
      const producible = building?.produces.find((p) => p.kind === RisqProducibleKind.UNIT && p.id === entry.id);
      if (!building || !producible) {
        return;
      }
      if (!canAffordCost(player, producible.cost)) {
        this.showMessage('Not enough resources', RISQ_MESSAGE_WARNING_COLOR);
      } else {
        this.createUnit(building.internal_id, entry.id);
      }
    } else if (entry.kind === 'research_tech') {
      const producible = building?.produces.find((p) => p.kind === RisqProducibleKind.TECH && p.id === entry.id);
      if (!building || !producible) {
        return;
      }
      const already_queued = this.orders_model
        .all()
        .some((o) => o.order_type === RisqOrderType.OrderType_BuildingResearch && o.target_id === entry.id);
      if (already_queued) {
        this.showMessage('Already queued for research', RISQ_MESSAGE_WARNING_COLOR);
      } else if (!canAffordCost(player, producible.cost)) {
        this.showMessage('Not enough resources', RISQ_MESSAGE_WARNING_COLOR);
      } else {
        this.researchTech(building.internal_id, entry.id);
      }
    } else {
      const producible = this.hotkeySubjectUnitIds()
        .map((id) => player.units.get(id)?.builds.find((p) => p.id === entry.id))
        .find((p) => !!p);
      if (!producible) {
        return;
      }
      if (this.getArmedOrder() === RisqOrderType.OrderType_UnitBuild && this.getArmedBuildingId() === entry.id) {
        this.disarmOrder();
      } else {
        this.armOrder(RisqOrderType.OrderType_UnitBuild, () => {}, {
          id: entry.id,
          display_name: producible.display_name,
        });
      }
    }
  }

  private ownedBuildingsOfType(building_id: number): RisqBuilding[] {
    const player = this.getPlayer();
    if (!player) {
      return [];
    }
    return [...player.buildings.values()]
      .filter((b) => b.building_id === building_id && !b.under_construction)
      .sort((a, b) => a.internal_id - b.internal_id);
  }

  private cycleBuildingSelection(building_id: number) {
    const buildings = this.ownedBuildingsOfType(building_id);
    if (buildings.length === 0) {
      return;
    }
    const current = this.left_panel.getData();
    const current_id = current?.data_type === LeftPanelDataType.BUILDING ? current.data.internal_id : undefined;
    const current_index = buildings.findIndex((b) => b.internal_id === current_id);
    const next = buildings[(current_index + 1) % buildings.length];
    this.left_panel.openPanel({ data_type: LeftPanelDataType.BUILDING, data: next }, RisqVisibilityLevel.SPY);
    this.goToCoordinate(next.space_coordinate);
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
      for (const row of this.game?.spaces ?? []) {
        for (const space of row) {
          for (const zone_row of space?.zones ?? []) {
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

  getRegionForSpace(coordinate_key: number): RisqRegion | undefined {
    return this.region_by_space.get(coordinate_key);
  }

  getRegionLookup(): Map<number, RisqRegion> {
    return this.region_by_space;
  }

  // regions are hoverable/selectable in the dedicated REGION view mode (any zoom), or fully zoomed out in any other non-military view mode
  regionInteractionEnabled(): boolean {
    return (
      this.view_mode === RisqViewMode.REGION ||
      (this.draw_detail === DrawRisqSpaceDetail.OWNERSHIP && this.view_mode !== RisqViewMode.MILITARY)
    );
  }

  hoveredRegion(): RisqRegion | undefined {
    if (!this.regionInteractionEnabled() || !this.hovered_space) {
      return undefined;
    }
    return this.getRegionForSpace(this.hovered_space.coordinate_key);
  }

  isRegionSelected(region: RisqRegion): boolean {
    const data = this.left_panel.getData();
    return data?.data_type === LeftPanelDataType.REGION && data.data.name === region.name;
  }

  /** Canvas position of a region's label: the centroid of its (explored) member spaces */
  regionLabelPosition(region: RisqRegion): Point2D {
    let sx = 0;
    let sy = 0;
    for (const key of region.spaces) {
      const c = invertPair(key);
      sx += c.x;
      sy += c.y;
    }
    const n = region.spaces.length || 1;
    return this.coordinateToCanvas({ x: sx / n, y: sy / n });
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
          log(`Unknown game update type ${update.kind}`);
          break;
      }
    } catch (e) {
      err(`Error during game update ${JSON.stringify(update)}: ${e}`);
    }
  }

  private setNewGameData(new_game: GameRisqFromServer) {
    this.game = serverToGameRisq(new_game);
    this.region_by_space = buildRegionLookup(this.game?.regions ?? []);
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
    this.last_turn_report = player?.turn_report;
    if (this.player_id > -1 && player?.turn_report) {
      this.openTurnReportDialog(player, player.turn_report);
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
    const bounds = this.visibleCanvasBounds();
    const on_screen_spaces: RisqSpace[] = [];
    for (const row of this.game.spaces) {
      for (const space of row) {
        if (!space) {
          continue;
        }
        space.center = this.coordinateToCanvas(space.coordinate);
        if (!this.isSpaceOnScreen(space, bounds)) {
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
    drawRisqRegionBorders(ctx, this, on_screen_spaces, this.hex_r);
    if (this.view_mode === RisqViewMode.REGION) {
      drawRisqRegionLabels(ctx, this);
    }
    this.drawUnitOrders(ctx);
    this.drawBuildingOrders(ctx);
    this.drawGatherPointOrder(ctx);
    this.drawPendingMercenaries(ctx);
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
    for (const component of this.canvas_components) {
      component.draw(ctx, transform, dt);
    }
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

  private drawBuildingOrders(ctx: CanvasRenderingContext2D) {
    const player = this.getPlayer();
    if (!player) {
      return;
    }
    ctx.setLineDash([8, 5]);
    for (const building of player.buildings.values()) {
      this.drawOrdersForBuilding(ctx, building, this.isBuildingSelected(building.internal_id));
    }
    ctx.setLineDash([]);
    ctx.globalAlpha = 1;
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
          {
            player_id: data.data.player_id,
            unit_id: data.data.unit_id,
            unit_type: data.data.unit_type,
            units: new Set([data.data.internal_id]),
          },
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

  private drawOrdersForBuilding(ctx: CanvasRenderingContext2D, building: RisqBuilding, selected: boolean) {
    const zone_view = this.draw_detail === DrawRisqSpaceDetail.ZONE_DETAILS;
    const building_offset = zoneCenterOffset(building.zone_coordinate, this.hex_r);
    const from = this.orderPoint(building.space_coordinate, building_offset, zone_view);
    ctx.lineWidth = selected ? 2 : 1;
    ctx.globalAlpha = selected ? 1 : 0.35;
    for (const order of this.drawnOrders(building, 'building')) {
      const to = this.orderTargetPoint(order, zone_view, from);
      if (!to || equalsPoint2D(from, to)) {
        continue;
      }
      const color = orderArrowColor(order.order_type);
      ctx.strokeStyle = color;
      ctx.fillStyle = color;
      drawArrow(ctx, from, to, selected ? 10 : 6);
    }
  }

  private drawnOrders(subject: RisqUnit | RisqBuilding, kind: 'unit' | 'building'): RisqFrontendOrder[] {
    const effective = this.orders_model.effectiveForSubject(subject.internal_id, kind);
    const queued_ids = new Set(subject.active_orders.map((o) => o.internal_id));
    const last_clear = [...effective].reverse().find((o) => o.clear_previous_orders);
    if (last_clear && !queued_ids.has(last_clear.internal_id ?? -1)) {
      return effective;
    }
    const queued = subject.active_orders.filter((o) => !this.orders_model.isCancelling(o.internal_id));
    const unqueued = effective.filter((o) => o.internal_id === undefined || !queued_ids.has(o.internal_id));
    return [...queued, ...unqueued];
  }

  private remainingMovePath(unit: RisqUnit): RisqMovePathStep[] {
    const path = unit.move_path ?? [];
    const location = this.unitLocation(unit);
    const current_index = path.findIndex(
      (step) =>
        equalsPoint2D(step.space, location?.space_coordinate) && equalsPoint2D(step.zone, location?.zone_coordinate)
    );
    return path.slice(current_index + 1);
  }

  private drawOrdersForUnit(ctx: CanvasRenderingContext2D, unit: RisqUnit, selected: boolean) {
    const location = this.unitLocation(unit);
    const orders = this.drawnOrders(unit, 'unit');
    if (!location || !orders.length) {
      return;
    }
    const zone_view = this.draw_detail === DrawRisqSpaceDetail.ZONE_DETAILS;
    const unit_offset = this.unitAnchorOffset(unit);
    let from = this.orderPoint(location.space_coordinate, unit_offset, zone_view);
    ctx.lineWidth = selected ? 2 : 1;
    ctx.globalAlpha = selected ? 1 : 0.35;
    ctx.setLineDash([8, 5]);
    for (const [i, order] of orders.entries()) {
      if (i === 0 && order === unit.active_orders[0] && unit.move_path?.length) {
        from = this.drawMovePath(ctx, this.remainingMovePath(unit), from, zone_view, selected ? 10 : 6);
        continue;
      }
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

  private drawMovePath(
    ctx: CanvasRenderingContext2D,
    path: RisqMovePathStep[],
    from: Point2D,
    zone_view: boolean,
    arrow_size: number
  ): Point2D {
    const color = orderArrowColor(RisqOrderType.OrderType_UnitMoveZone);
    ctx.strokeStyle = color;
    ctx.fillStyle = color;
    let current = from;
    for (const [i, step] of path.entries()) {
      const to = this.orderPoint(step.space, zoneCenterOffset(step.zone, this.hex_r), zone_view);
      if (equalsPoint2D(current, to)) {
        continue;
      }
      if (i === path.length - 1) {
        drawArrow(ctx, current, to, arrow_size);
      } else {
        drawLine(ctx, current, to);
      }
      current = to;
    }
    return current;
  }

  /** Space/zone-view-aware canvas point for an order endpoint; offset is ignored (space-to-space) outside zone view */
  private orderPoint(space: Point2D, offset: Point2D | undefined, zone_view: boolean): Point2D {
    const p = this.coordinateToCanvas(space);
    return zone_view && offset ? addPoint2D(p, offset) : p;
  }

  /** A garrisoned unit has no coordinates of its own, so it's located at its building */
  unitLocation(unit: RisqUnit): { space_coordinate: Point2D; zone_coordinate: Point2D } | undefined {
    return unit.garrisoned_in === undefined ? unit : this.findBuildingById(unit.garrisoned_in);
  }

  /** Offset (relative to its space's center) of the specific unit-slot circle a unit currently occupies */
  private unitAnchorOffset(unit: RisqUnit): Point2D | undefined {
    if (!this.game) {
      return undefined;
    }
    if (unit.garrisoned_in !== undefined) {
      const building = this.findBuildingById(unit.garrisoned_in);
      return building ? zoneCenterOffset(building.zone_coordinate, this.hex_r) : undefined;
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
      case RisqOrderType.OrderType_UnitAutoAttackBuilding:
      case RisqOrderType.OrderType_BuildingAttackBuilding:
      case RisqOrderType.OrderType_BuildingAutoAttackBuilding:
      case RisqOrderType.OrderType_UnitGarrison: {
        const building = this.findBuildingById(order.target_id);
        if (!building) {
          return undefined;
        }
        target_space = building.space_coordinate;
        target_offset = zoneCenterOffset(building.zone_coordinate, this.hex_r);
        break;
      }
      case RisqOrderType.OrderType_UnitAttackUnit:
      case RisqOrderType.OrderType_UnitAutoAttackUnit:
      case RisqOrderType.OrderType_BuildingAttackUnit:
      case RisqOrderType.OrderType_BuildingAutoAttackUnit: {
        const target_unit = this.findUnitById(order.target_id);
        const target_location = target_unit ? this.unitLocation(target_unit) : undefined;
        if (!target_unit || !target_location) {
          return undefined;
        }
        target_space = target_location.space_coordinate;
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

  private drawPendingMercenaries(ctx: CanvasRenderingContext2D) {
    const zone_view = this.draw_detail === DrawRisqSpaceDetail.ZONE_DETAILS;
    const groups = new Map<number, { positions: Point2D[]; unit_ids: number[] }>();
    for (const order of this.pendingMercenaryOrders()) {
      const { x: unit_id, y: zone_key } = invertPair(order.target_id);
      const { space, zone } = invertZoneKey(zone_key);
      const key = zone_view ? zone_key : cantorPair(space.x, space.y);
      const space_center = this.coordinateToCanvas(space);
      const group = groups.get(key) ?? {
        positions: zoneMercenarySlotOffsets(zone_view ? zone : { x: 0, y: 0 }, this.hex_r).map((offset) =>
          addPoint2D(space_center, offset)
        ),
        unit_ids: [],
      };
      group.unit_ids.push(unit_id);
      groups.set(key, group);
    }
    for (const { positions, unit_ids } of groups.values()) {
      const slots = mercenaryGhostSlots(this.player_id, unit_ids, positions.length);
      drawMercenaryGhosts(ctx, this, slots, positions, this.hex_r, this.last_transform.rotation);
    }
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

  private panelsHovered(): boolean {
    return this.left_panel.isHovering() || this.right_panel.isHovering() || this.bottom_panel.isHovering();
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
    this.board.setPanSuppressed(false, this.panelsHovered());
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
      if (this.draw_detail === DrawRisqSpaceDetail.ZONE_DETAILS && this.view_mode !== RisqViewMode.REGION) {
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
      resolve_zones();
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
    resolve_zones();
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
    } else if (e.button === 2 && this.armed_order === RisqOrderType.OrderType_BuyMercenary) {
      this.placeMercenary(e.ctrlKey);
      return true;
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
          this.buildingOrder(left_panel_data, e.ctrlKey);
          break;
        default:
          break;
      }
      return true;
    }
    const armed = this.anythingArmed();
    if (this.draw_detail === DrawRisqSpaceDetail.ZONE_DETAILS && e.button === 0 && !e.shiftKey && !armed) {
      this.dragging_selection = true;
      this.drag_additive = e.ctrlKey;
      this.drag_start = { ...this.mouse_screen };
      this.drag_current = this.drag_start;
      return true;
    }
    return (e.button === 2 && armed) || (e.button !== 0 && e.button !== 2);
  }

  armOrder(order_type: RisqOrderType, on_disarm: () => void, building?: { id: number; display_name: string }) {
    this.armed_button_callback?.();
    this.armed_order = order_type;
    this.armed_building = building;
    this.armed_mercenary = undefined;
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

  private anythingArmed(): boolean {
    return this.armed_order !== RisqOrderType.NONE || this.gather_point_armed || this.building_attack_armed;
  }

  disarmAll() {
    this.gather_point_armed = false;
    this.building_attack_armed = false;
    this.disarmOrder();
  }

  getArmedMercenaryId(): number | undefined {
    return this.armed_mercenary?.id;
  }

  toggleMercenary(mercenary: RisqProducible) {
    if (this.armed_mercenary?.id === mercenary.id) {
      this.disarmOrder();
      return;
    }
    const unavailable_reason = this.mercenaryUnavailableReason(mercenary);
    if (unavailable_reason) {
      this.showMessage(unavailable_reason, RISQ_MESSAGE_WARNING_COLOR);
      return;
    }
    this.armOrder(RisqOrderType.OrderType_BuyMercenary, () => {});
    this.armed_mercenary = mercenary;
    this.updateCursor(false);
  }

  disarmOrder() {
    this.armed_button_callback?.();
    this.armed_order = RisqOrderType.NONE;
    this.armed_building = undefined;
    this.armed_mercenary = undefined;
    this.armed_button_callback = undefined;
    this.updateCursor(false);
  }

  mercenaryUnavailableReason(mercenary: RisqProducible): string | undefined {
    const player = this.getPlayer();
    if (!player || !this.canGiveOrders()) {
      return 'Cannot give orders now';
    }
    if (!canAffordCost(player, mercenary.cost)) {
      return 'Not enough resources';
    }
    if (player.units.size + this.pendingMercenaryCount() >= player.population_limit) {
      return 'Population capped';
    }
    return undefined;
  }

  private mercenaryTargetZone(): RisqZone | undefined {
    return this.hovered_zone ?? this.hovered_space?.zones?.[1][1];
  }

  private mercenaryPlacementInvalidReason(zone: RisqZone | undefined): string | undefined {
    if (!this.armed_mercenary || !this.hovered_space || !zone) {
      return 'No valid location';
    }
    if (this.hovered_space.ownership !== this.player_id || zone.ownership !== this.player_id) {
      return 'Must hire in owned territory';
    }
    const region = this.getRegionForSpace(this.hovered_space.coordinate_key);
    if (region && region.owner !== this.player_id) {
      return 'Region not fully owned';
    }
    return this.mercenaryUnavailableReason(this.armed_mercenary);
  }

  private placeMercenary(keep_armed: boolean) {
    const zone = this.mercenaryTargetZone();
    const invalid_reason = this.mercenaryPlacementInvalidReason(zone);
    if (invalid_reason || !zone || !this.armed_mercenary) {
      this.showMessage(invalid_reason ?? 'No valid location', RISQ_MESSAGE_WARNING_COLOR);
      return;
    }
    this.orders_model.add({
      player_id: this.player_id,
      order_type: RisqOrderType.OrderType_BuyMercenary,
      subjects: [],
      target_id: cantorPair(this.armed_mercenary.id, zone.coordinate_key),
      clear_previous_orders: false,
    });
    if (!keep_armed) {
      this.disarmOrder();
    }
  }

  pendingMercenaryCount(): number {
    return this.pendingMercenaryOrders().length;
  }

  private pendingMercenaryOrders(): RisqFrontendOrder[] {
    return this.orders_model.all().filter((o) => o.order_type === RisqOrderType.OrderType_BuyMercenary);
  }

  createUnit(building_id: number, unit_id: number, ctrl_held = false) {
    if (!this.canGiveOrders()) {
      return;
    }
    this.orders_model.add({
      player_id: this.player_id,
      order_type: RisqOrderType.OrderType_BuildingCreate,
      subjects: [building_id],
      target_id: unit_id,
      clear_previous_orders: !ctrl_held,
    });
  }

  researchTech(building_id: number, tech_id: number, ctrl_held = false) {
    if (!this.canGiveOrders()) {
      return;
    }
    this.orders_model.add({
      player_id: this.player_id,
      order_type: RisqOrderType.OrderType_BuildingResearch,
      subjects: [building_id],
      target_id: tech_id,
      clear_previous_orders: !ctrl_held,
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

  setUnitTargetPriority(internal_ids: number[], target_priority: RisqTargetCategory[]) {
    this.sendUnitBehavior(internal_ids, { target_priority });
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
    const location = first ? this.unitLocation(first) : undefined;
    if (units_by_type.length === 0 || !location) {
      return;
    }
    const space = getSpace(game, coordinateToIndex(game.board_size, location.space_coordinate));
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

  predictedGathererCount(zone_key: number): number {
    const player = this.getPlayer();
    if (!player) {
      return 0;
    }
    let count = 0;
    for (const unit of player.units.values()) {
      const effective = this.orders_model.effectiveForSubject(unit.internal_id, 'unit');
      if (effective.some((o) => o.order_type === RisqOrderType.OrderType_UnitGather && o.target_id === zone_key)) {
        count++;
      }
    }
    return count;
  }

  private warnIfGatherOverCapacity(target: RisqBuilding | RisqResource | undefined) {
    if (!target || target.gather_capacity === undefined) {
      return;
    }
    const zone_key = cantorPair(
      cantorPair(target.space_coordinate.x, target.space_coordinate.y),
      cantorPair(target.zone_coordinate.x, target.zone_coordinate.y)
    );
    if (this.predictedGathererCount(zone_key) > target.gather_capacity) {
      this.showMessage(`${target.display_name} is over its worker capacity`, RISQ_MESSAGE_WARNING_COLOR);
    }
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
    const location = this.unitLocation(unit);
    if (!location) {
      return;
    }
    const scale = this.last_transform.scale;
    const view = addPoint2D(
      this.coordinateToCanvas(location.space_coordinate),
      zoneCenterOffset(location.zone_coordinate, this.hex_r)
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
        this.addSpending(
          player,
          player.buildings.get(subject_id)?.produces.find((p) => p.kind === kind && p.id === order.target_id)?.cost
        );
      }
    }
    for (const foundation of this.local_foundations.values()) {
      this.addSpending(player, this.plannedFoundationCost(player, foundation));
    }
    for (const order of this.orders_model.pendingOrders()) {
      if (order.order_type === RisqOrderType.OrderType_BuyMercenary) {
        const unit_id = invertPair(order.target_id).x;
        this.addSpending(player, player.available_mercenaries.find((m) => m.id === unit_id)?.cost);
      } else if (order.order_type === RisqOrderType.OrderType_CancelOrder) {
        this.addSpending(player, this.paidProductionCost(player, order.target_id), -1);
      }
    }
    const renew_targets = new Set(
      this.orders_model
        .pendingOrders()
        .filter((o) => o.order_type === RisqOrderType.OrderType_UnitRenew)
        .map((o) => o.target_id)
    );
    for (const target_id of renew_targets) {
      this.addSpending(player, this.findBuildingById(target_id)?.renew_cost);
    }
  }

  /** Cost the server already charged for a production order still in a building's queue, refunded if it's cancelled */
  private paidProductionCost(player: RisqPlayer, order_internal_id: number): RisqCost | undefined {
    for (const building of player.buildings.values()) {
      const item = building.production_queue.find((i) => i.order_internal_id === order_internal_id);
      if (item) {
        return building.produces.find((p) => p.kind === item.kind && p.id === item.item_id)?.cost;
      }
    }
    return undefined;
  }

  private plannedFoundationCost(player: RisqPlayer, foundation: LocalRisqFoundation): RisqCost | undefined {
    for (const subject_id of foundation.order.subjects) {
      const cost = player.units.get(subject_id)?.builds.find((p) => p.id === foundation.building_id)?.cost;
      if (cost) {
        return cost;
      }
    }
    return undefined;
  }

  private addSpending(player: RisqPlayer, cost: RisqCost | undefined, multiplier = 1) {
    if (!cost) {
      return;
    }
    player.resources.get(RisqResourceType.FOOD)!.spending += multiplier * cost.food;
    player.resources.get(RisqResourceType.WOOD)!.spending += multiplier * cost.wood;
    player.resources.get(RisqResourceType.STONE)!.spending += multiplier * cost.stone;
    player.resources.get(RisqResourceType.GOLD)!.spending += multiplier * cost.gold;
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

  private gatherTargetValid(): boolean {
    if (!this.isZoneValid()) {
      return false;
    }
    if (!!this.hovered_zone?.resource) {
      return true;
    }
    const b = this.hovered_zone?.building;
    return (
      !!b &&
      b.player_id === this.player_id &&
      !b.under_construction &&
      b.gather_capacity !== undefined &&
      (b.resources_left ?? 0) > 0
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
    const bounds = this.visibleCanvasBounds();
    const radius = UNIT_SLOT_CIRCLE_RADIUS_MULTIPLIER * this.hex_r;
    for (const row of this.game?.spaces ?? []) {
      for (const space of row) {
        if (!space || !this.isSpaceOnScreen(space, bounds)) {
          continue;
        }
        for (const zone_row of space.zones ?? []) {
          for (const zone of zone_row) {
            for (const unit of zone.units.values()) {
              if (unit.player_id !== player_id || unit.unit_id !== unit_id) {
                continue;
              }
              const canvas_pos = this.orderPoint(unit.space_coordinate, this.unitAnchorOffset(unit), true);
              if (this.isCircleOnScreen(canvas_pos, radius, bounds)) {
                ids.push(unit.internal_id);
              }
            }
          }
        }
      }
    }
    return ids;
  }

  private visibleCanvasBounds(): CanvasBounds {
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

  private isSpaceOnScreen(space: RisqSpace, bounds: CanvasBounds): boolean {
    const { min, max } = bounds;
    return !(
      space.center.x + this.hex_a < min.x ||
      space.center.x - this.hex_a > max.x ||
      space.center.y + this.hex_r < min.y ||
      space.center.y - this.hex_r > max.y
    );
  }

  private isCircleOnScreen(c: Point2D, radius: number, bounds: CanvasBounds): boolean {
    const { min, max } = bounds;
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
          if (has_villager && this.gatherTargetValid()) {
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
      if (has_villager && this.hovered_zone?.hovered_data[0]?.hovered && this.gatherTargetValid()) {
        return RisqOrderType.OrderType_UnitGather;
      }
      if (has_villager && this.hovered_zone?.hovered_data[0]?.hovered && this.repairTargetValid()) {
        return RisqOrderType.OrderType_UnitRepair;
      }
      if (has_villager && zone_valid && this.hovered_zone?.hovered_data[0]?.hovered) {
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
      const valid = this.buildTargetValid();
      const url = resolveBuildCursorUrl(this, this.armed_building, valid);
      if (url) {
        this.board.setCursorUrl(url, armedCursorAlpha(valid));
        return;
      }
    }
    if (this.armed_mercenary) {
      const valid = !this.mercenaryPlacementInvalidReason(this.mercenaryTargetZone());
      const url = resolveMercenaryCursorUrl(this, this.armed_mercenary.id, valid);
      if (url) {
        this.board.setCursorUrl(url, armedCursorAlpha(valid));
        return;
      }
    }
    if (this.building_attack_armed) {
      const data = this.left_panel.getData();
      const valid = data?.data_type === LeftPanelDataType.BUILDING && !!this.resolveBuildingAttackTarget(data.data);
      this.board.setCursor(
        cursorImageForOrderType(RisqOrderType.OrderType_BuildingAttackUnit),
        armedCursorAlpha(valid)
      );
      return;
    }
    const active_order = this.resolveActiveOrderType(ctrl_held);
    if (active_order === RisqOrderType.NONE && this.armed_order !== RisqOrderType.NONE) {
      this.board.setCursor(cursorImageForOrderType(this.armed_order), armedCursorAlpha(false));
      return;
    }
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

  private buildingOrder(data: BuildingData, ctrl_held: boolean) {
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
        clear_previous_orders: !ctrl_held,
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
        this.warnIfGatherOverCapacity(this.hovered_zone.building ?? this.hovered_zone.resource);
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
      [...u.units].map((internal_id) => ({ unit_type: u.unit_type, internal_id }))
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
        const attackers =
          this.getArmedOrder() === RisqOrderType.NONE
            ? units.filter((u) => u.unit_type !== RisqUnitType.ECONOMIC)
            : units;
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
        const attackers =
          this.getArmedOrder() === RisqOrderType.NONE
            ? units.filter((u) => u.unit_type !== RisqUnitType.ECONOMIC)
            : units;
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
          units.filter((u) => u.unit_type === RisqUnitType.ECONOMIC).map((u) => u.internal_id),
          this.hovered_zone.coordinate_key,
          ctrl_held
        );
        this.warnIfGatherOverCapacity(this.hovered_zone.building ?? this.hovered_zone.resource);
        break;
      case RisqOrderType.OrderType_UnitBuild:
        if (!this.hovered_zone || !this.armed_building) {
          return;
        }
        const group_order: RisqFrontendOrder = {
          player_id: this.player_id,
          order_type: RisqOrderType.OrderType_UnitBuild,
          subjects: units.filter((u) => u.unit_type === RisqUnitType.ECONOMIC).map((u) => u.internal_id),
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
          units.filter((u) => u.unit_type === RisqUnitType.ECONOMIC).map((u) => u.internal_id),
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
          units.filter((u) => u.unit_type === RisqUnitType.ECONOMIC).map((u) => u.internal_id),
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
                  this.left_panel.openPanel(
                    { data_type: LeftPanelDataType.BUILDING, data: zone.building },
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
                      if (deselect) {
                        target.units.delete(id);
                      } else {
                        target.units.add(id);
                      }
                    }
                    if (target.units.size > 0) {
                      groups.set(key, target);
                    } else {
                      groups.delete(key);
                    }
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
      } else if (
        space.clicked &&
        space.visibility > 0 &&
        (this.draw_detail !== DrawRisqSpaceDetail.ZONE_DETAILS || this.view_mode === RisqViewMode.REGION)
      ) {
        const region = this.regionInteractionEnabled() ? this.getRegionForSpace(space.coordinate_key) : undefined;
        if (region) {
          this.left_panel.openPanel({ data_type: LeftPanelDataType.REGION, data: region }, space.visibility);
        } else {
          this.left_panel.openPanel({ data_type: LeftPanelDataType.SPACE, data: space }, space.visibility);
        }
      }
      space.clicked = false;
    } else if (e.button === 0 && armed_before === RisqOrderType.NONE && !this.panelsHovered()) {
      this.left_panel.close();
    }
    const keep_mercenary_armed = e.ctrlKey && armed_before === RisqOrderType.OrderType_BuyMercenary;
    if (
      armed_before !== RisqOrderType.NONE &&
      this.armed_order === armed_before &&
      this.armed_button_callback === armed_callback_before &&
      !keep_mercenary_armed
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
