import { DwgElement } from '../../../dwg_element';
import type { UpdateMessage } from '../../data_models';
import { drawCircle } from '../../util/canvas_util';
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
import { DEV, createLock } from '../../../../scripts/util';
import { ColorRGB } from '../../../../scripts/color_rgb';

import html from './risq.html';
import type { GameRisq, GameRisqFromServer, RisqPlayer, RisqSpace, RisqZone, StartTurnData } from './risq_data';
import { coordinateToIndex, getSpace, RisqOrderType, serverToGameRisq } from './risq_data';
import { RisqRightPanel } from './canvas_components/right_panel/right_panel';
import type { DrawRisqSpaceConfig } from './risq_space';
import { DrawRisqSpaceDetail, drawRisqSpace } from './risq_space';
import { RisqLeftPanel } from './canvas_components/left_panel/left_panel';
import { resolveHoveredZones, unhoverRisqZone } from './risq_zone';
import type { UnitData } from './canvas_components/left_panel/left_panel_data';
import { LeftPanelDataType } from './canvas_components/left_panel/left_panel_data';

import './risq.scss';
import '../../util/canvas_board/canvas_board';
import { createMessage } from '../../../lobby/data_models';

const DEFAULT_HEXAGON_RADIUS = 60;

const DRAW_CENTER_DOT = false;

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
  private last_time = Date.now();
  private draw_detail: DrawRisqSpaceDetail = DrawRisqSpaceDetail.SPACE_DETAILS;
  private toggling_submit_orders_button = false;
  private giving_orders = false;
  private orders_submitted = false;
  private orders_submitted_times = 0;

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
    this.right_panel.dataRefreshed();
  }

  private async applyStartTurn(data: StartTurnData) {
    if (this.player_id > -1) {
      this.giving_orders = true;
      this.orders_submitted = false;
      this.orders_submitted_times = 0;
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
    const inset_row = inset_h / 3 - 4;
    this.draw_detail = this.getDrawDetail(transform.scale);
    const draw_config: DrawRisqSpaceConfig = {
      hex_r: this.hex_r,
      inset_w,
      inset_h,
      inset_row,
      draw_detail: this.draw_detail,
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
    } else if (e.button === 2 && this.left_panel.isOrderable()) {
      const left_panel_data = this.left_panel.getData();
      switch (left_panel_data?.data_type) {
        case LeftPanelDataType.UNIT:
          this.unitOrder(left_panel_data);
          break;
        default:
          break;
      }
    }
    // only drag on left click
    // TODO: implement rotate on right click
    return e.button !== 0;
  }

  private unitOrder(data: UnitData) {
    if (!this.hovered_space) {
      return;
    }
    // TODO: implement attack vs just move
    // TODO: implement if holding the shift key
    if (
      this.hovered_space.visibility < 3 ||
      this.draw_detail !== DrawRisqSpaceDetail.ZONE_DETAILS ||
      !this.hovered_zone
    ) {
      // TODO: if able to attack *not* holding ctrl, then add move and then attack commands (or just attack if in range)
      // move to hovered space
      this.right_panel.addOrder({
        player_id: this.player_id,
        order_type: RisqOrderType.OrderType_UnitMoveSpace,
        subjects: [data.data.internal_id],
        target_id: this.hovered_space.coordinate_key,
      });
    } else {
      this.right_panel.addOrder({
        player_id: this.player_id,
        order_type: RisqOrderType.OrderType_UnitMoveZone,
        subjects: [data.data.internal_id],
        target_id: this.hovered_zone.coordinate_key,
      });
    }
  }

  private mouseup(e: MouseEvent) {
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
    if (this.toggling_submit_orders_button || !this.giving_orders) {
      this.toggling_submit_orders_button = false;
      return;
    }
    this.toggling_submit_orders_button = true;
    if (this.orders_submitted) {
      this.right_panel.unsubmittingOrders();
      const game_update = createMessage(
        `player-${this.player_id}`,
        'game-update',
        JSON.stringify(this.right_panel.getOrders()),
        'unsubmit-orders'
      );
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
        JSON.stringify(this.right_panel.getOrders()),
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
    return this.giving_orders;
  }

  ordersSubmitted(): boolean {
    return this.orders_submitted;
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
