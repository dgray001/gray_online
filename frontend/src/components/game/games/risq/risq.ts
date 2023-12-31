import {DwgElement} from '../../../dwg_element';
import {UpdateMessage} from '../../data_models';
import {drawHexagon} from '../../util/canvas_util';
import {BoardTransformData, DwgCanvasBoard} from '../../util/canvas_board/canvas_board';
import {Point2D, equalsPoint2D, hexagonalBoardNeighbors, hexagonalBoardRows, roundAxialCoordinate} from '../../util/objects2d';
import {DwgGame} from '../../game';
import {createLock, until} from '../../../../scripts/util';

import html from './risq.html';
import {GameRisq, GameRisqFromServer, RisqSpace, coordinateToIndex, getSpace, getSpaceFill, serverToGameRisq} from './risq_data';

import './risq.scss';
import '../../util/canvas_board/canvas_board';
import './space_dialog/space_dialog';

const DEFAULT_HEXAGON_RADIUS = 60;

export class DwgRisq extends DwgElement {
  private board: DwgCanvasBoard;

  private game: GameRisq;
  private hex_r = DEFAULT_HEXAGON_RADIUS;
  private hex_a = 0.5 * 1.732 * DEFAULT_HEXAGON_RADIUS;
  private canvas_center: Point2D = {x: 0, y: 0};
  private mouse_canvas: Point2D = {x: 0, y: 0};
  private mouse_coordinate: Point2D = {x: 0, y: 0};
  private hovered_space?: RisqSpace;
  private icons = new Map<string, HTMLImageElement>();

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('board');
    for (const icon of ['unit', 'building', 'unit_white', 'building_white']) {
      this.createIcon(icon);
    }
  }

  private createIcon(name: string) {
    const el = document.createElement('img');
    el.src = `/images/icons/${name}64.png`;
    el.draggable = false;
    el.alt = name;
    this.icons.set(name, el);
  }

  async initialize(abstract_game: DwgGame, game: GameRisqFromServer): Promise<void> {
    abstract_game.setPadding('0px');
    this.game = serverToGameRisq(game);
    const canvas_size = {
      x: 1.732 * this.hex_r * (2 * game.board_size + 1),
      y: 1.5 * this.hex_r * (2 * game.board_size + 1) + 0.5 * this.hex_r,
    };
    this.board.initialize({
      size: canvas_size,
      max_scale: 1.8,
      fill_space: true,
      draw: this.draw.bind(this),
      mousemove: this.mousemove.bind(this),
      mouseleave: this.mouseleave.bind(this),
      mousedown: this.mousedown.bind(this),
      mouseup: this.mouseup.bind(this),
    }).then(() => {
      const canvas_rect = this.board.getBoundingClientRect();
      this.canvas_center = {
        x: 0.5 * Math.min(canvas_size.x, canvas_rect.width),
        y: 0.5 * Math.min(canvas_size.y, canvas_rect.height),
      };
      // canvas board component should respect scale
      this.hex_r = (2 * this.canvas_center.x) / (1.732 * (2 * game.board_size + 1));
      this.hex_a = 0.5 * 1.732 * this.hex_r;
    });
  }

  async gameUpdate(update: UpdateMessage): Promise<void> {
  }

  private draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData) {
    ctx.strokeStyle = 'rgba(250, 250, 250, 0.9)';
    ctx.textAlign = 'left';
    ctx.lineWidth = 2;
    const inset_offset = 0.25; // this determines how the inset rect (for summaries) is constructed
    const inset_w = 2 * this.hex_a * (1 - inset_offset);
    const inset_h = this.hex_r * (1 + inset_offset);
    const inset_row = inset_h / 3 - 4;
    ctx.font = `bold ${inset_row}px serif`;
    for (const row of this.game.spaces) {
      for (const space of row) {
        const fill = getSpaceFill(space);
        ctx.fillStyle = fill.getString();
        space.center = this.coordinateToCanvas(space.coordinate, transform.scale);
        drawHexagon(ctx, space.center, this.hex_r);
        if (space.visibility > 0) {
          let building_img = this.icons.get('building');
          let unit_img = this.icons.get('unit');
          if (fill.getBrightness() > 0.5) {
            ctx.fillStyle = 'black';
          } else {
            ctx.fillStyle = 'white';
            building_img = this.icons.get('building_white');
            unit_img = this.icons.get('unit_white');
          }
          ctx.textBaseline = 'top';
          const xs = space.center.x - 0.5 * inset_w;
          const y1 = space.center.y - 0.5 * inset_h;
          ctx.drawImage(building_img, xs, y1, inset_row, inset_row);
          ctx.fillText(`: ${space.buildings.size.toString()}`, xs + inset_row + 2, y1, inset_w - inset_row - 2);
          const y2 = space.center.y - 0.5 * inset_h + inset_row + 2;
          ctx.drawImage(unit_img, xs, y2, inset_row, inset_row);
          ctx.fillText(`: ${space.units.size.toString()}`, xs + inset_row + 2, y2, inset_w - inset_row - 2);
        }
      }
    }
  }

  private mousemove(m: Point2D, transform: BoardTransformData) {
    this.mouse_canvas = m;
    this.mouse_coordinate = this.canvasToCoordinate(m, transform.scale);
    const index = coordinateToIndex(this.game.board_size, roundAxialCoordinate(this.mouse_coordinate));
    const new_hovered_space = getSpace(this.game, index);
    if (!new_hovered_space) {
      this.removeHoveredFlags();
      if (!!this.hovered_space) {
        this.hovered_space.clicked = false;
        this.hovered_space = undefined;
      }
      return;
    }
    if (equalsPoint2D(new_hovered_space.coordinate, this.hovered_space?.coordinate)) {
      this.updateHoveredFlags();
      return;
    }
    this.removeHoveredFlags();
    if (!!this.hovered_space) {
      this.hovered_space.clicked = false;
    }
    this.hovered_space = new_hovered_space;
    this.updateHoveredFlags();
  }

  private mouseleave() {
    if (!!this.hovered_space) {
      this.hovered_space.hovered = false;
      this.hovered_space.clicked = false;
      this.hovered_space = undefined;
    }
  }

  private mousedown(e: MouseEvent): boolean {
    if (e.button !== 0) {
      return false;
    }
    if (!!this.hovered_space && this.hovered_space.visibility > 0) {
      this.hovered_space.clicked = true;
      return true;
    }
    return false;
  }

  private mouseup(_e: MouseEvent) {
    if (!!this.hovered_space) {
      if (this.hovered_space.clicked && this.hovered_space.visibility > 0) {
        this.openSpaceDialog(this.hovered_space);
      }
      this.hovered_space.clicked = false;
    }
  }

  private _space_dialog_lock = createLock(true);
  private async openSpaceDialog(space: RisqSpace) {
    if (!space) {
      return;
    }
    this._space_dialog_lock(async () => {
      const space_dialog = document.createElement('dwg-space-dialog');
      space_dialog.setData({space, game: this.game});
      this.appendChild(space_dialog);
      await until(() => !document.body.contains(space_dialog));
    });
  }

  private canvasToCoordinate(canvas: Point2D, scale: number): Point2D {
    const cy = (((canvas.y - 0.25 * this.hex_r - this.canvas_center.y / scale) / (1.5 * this.hex_r)) - this.game.board_size - 0.5);
    return {
      x: ((canvas.x - this.canvas_center.x / scale) / (1.732 * this.hex_r)) - 0.5 * cy - this.game.board_size - 0.5,
      y: cy,
    };
  }

  private coordinateToCanvas(coordinate: Point2D, scale: number): Point2D {
    return {
      x: 1.732 * (coordinate.x + 0.5 * coordinate.y + this.game.board_size + 0.5) * this.hex_r + this.canvas_center.x / scale,
      y: 1.5 * (coordinate.y + this.game.board_size + 0.5) * this.hex_r + 0.25 * this.hex_r + this.canvas_center.y / scale,
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
}

customElements.define('dwg-risq', DwgRisq);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-risq': DwgRisq;
  }
}
