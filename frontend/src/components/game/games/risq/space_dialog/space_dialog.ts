import {ColorRGB} from '../../../../../scripts/color_rgb';
import {atangent} from '../../../../../scripts/math';
import {DwgDialogBox} from '../../../../dialog_box/dialog_box';
import {drawHexagon} from '../../../util/canvas_util';
import {Point2D, addPoint2D, equalsPoint2D, multiplyPoint2D, pointInHexagon, subtractPoint2D} from '../../../util/objects2d';
import {GameRisq, RisqSpace, RisqZone, coordinateToIndex, getSpace, getSpaceFill, indexToCoordinate} from '../risq_data';

import html from './space_dialog.html';

import './space_dialog.scss';

/** Input data for a space dialog */
export declare interface SpaceDialogData {
  space: RisqSpace;
  game: GameRisq;
}

export class DwgSpaceDialog extends DwgDialogBox<SpaceDialogData> {
  canvas: HTMLCanvasElement;

  private data: SpaceDialogData;
  private size: Point2D;
  private radius: number;
  private ctx: CanvasRenderingContext2D;
  private draw_interval?: NodeJS.Timer = undefined;
  private mouse: Point2D = {x: 0, y: 0};
  private hovered_zone?: RisqZone = undefined;
  private hovered_space?: RisqSpace = undefined;
  private hovered = false;

  constructor() {
    super();
    this.configureElement('canvas');
  }

  override getHTML(): string {
    return html;
  }

  override closeDialog(): void {
    if (!!this.draw_interval) {
      clearInterval(this.draw_interval);
    }
    document.body.removeEventListener('keyup', this.keyup.bind(this));
    super.closeDialog();
  }

  getData(): SpaceDialogData {
    return this.data;
  }

  setData(data: SpaceDialogData, parsed?: boolean) {
    if (!!data) {
      this.data = data;
    }
    if (!parsed && !this.fully_parsed) {
      return;
    }
    if (!this.canvas.getContext) {
      console.error('Browser does not support canvas; cannot draw board');
      return;
    }
    for (const row of this.data.space.zones) {
      for (const zone of row) {
        zone.hovered = false;
        zone.clicked = false;
      }
    }
    const size = Math.min(0.95 * window.innerHeight, 0.95 * window.innerWidth);
    this.size = {x: size, y: 0.5 * 1.732 * size};
    this.radius = 0.38 * size;
    this.ctx = this.canvas.getContext('2d');
    this.canvas.style.setProperty('--w', `${this.size.x.toString()}px`);
    this.canvas.style.setProperty('--h', `${this.size.y.toString()}px`);
    this.canvas.width = this.size.x;
    this.canvas.height = this.size.y;
    this.addEventListener('mousemove', (e: MouseEvent) => {
      this.hovered = true;
      const rect = this.canvas.getBoundingClientRect();
      this.mousemove({x: (e.clientX - rect.left), y: (e.clientY - rect.top)});
    });
    this.addEventListener('mousedown', (e: MouseEvent) => {
      e.stopImmediatePropagation();
      this.mousedown(e);
    });
    this.addEventListener('mouseup', (e: MouseEvent) => {
      e.stopImmediatePropagation();
      this.mouseup(e);
    });
    this.addEventListener('mouseenter', () => {
      this.hovered = true;
    });
    this.addEventListener('mouseleave', () => {
      this.hovered = false;
    });
    this.addEventListener('contextmenu', (e) => {
      e.preventDefault();
    }, false);
    document.body.addEventListener('keyup', this.keyup.bind(this));
    this.draw_interval = setInterval(() => {
      // TODO: ensure last frame is done drawing (need to do tick either way)
      this.ctx.resetTransform();
      this.ctx.fillStyle = "black";
      this.ctx.fillRect(0, 0, this.size.x, this.size.y);
      this.draw();
    }, 20);
  }

  private draw() {
    this.ctx.strokeStyle = 'rgba(250, 250, 250, 0.9)';
    this.ctx.fillStyle = 'rgb(0, 0, 0, 0)';
    this.ctx.lineWidth = 4;
    const c = multiplyPoint2D(0.5, this.size);
    this.ctx.translate(c.x, c.y);
    const r = this.radius;
    drawHexagon(this.ctx, {x: 0, y: 0}, r);
    this.ctx.fillStyle = 'rgb(10, 120, 10, 0.8)';
    this.ctx.lineWidth = 2;
    let zone = this.data.space.zones[1][1];
    this.setZoneFill(zone);
    drawHexagon(this.ctx, {x: 0, y: 0}, 0.4 * r);
    this.drawZone(zone,
      {x: 0.2 * r * Math.cos(0), y: 0.2 * r * Math.sin(0)},
      {x: 0.2 * r * Math.cos(2 * Math.PI / 3), y: 0.2 * r * Math.sin(2 * Math.PI / 2)},
      {x: 0.2 * r * Math.cos(4 * Math.PI / 3), y: 0.2 * r * Math.sin(4 * Math.PI / 2)},
    );
    const a = Math.PI / 3;
    for (var i = 0; i < 6; i++) {
      let direction_vector: Point2D = {x: 0, y: 0};
      switch(i) {
        case 0:
          direction_vector = {x: 2, y: 1};
          break;
        case 1:
          direction_vector = {x: 2, y: 0};
          break;
        case 2:
          direction_vector = {x: 1, y: 0};
          break;
        case 3:
          direction_vector = {x: 0, y: 0};
          break;
        case 4:
          direction_vector = {x: 0, y: 1};
          break;
        case 5:
          direction_vector = {x: 1, y: 2};
          break;
      }
      zone = this.data.space.zones[direction_vector.x][direction_vector.y];
      this.setZoneFill(zone);
      this.ctx.beginPath();
      this.ctx.lineTo(0.4 * r * Math.cos(a * i + Math.PI / 6), 0.4 * r * Math.sin(a * i + Math.PI / 6));
      this.ctx.lineTo(0.4 * r * Math.cos(a * i + Math.PI / 2), 0.4 * r * Math.sin(a * i + Math.PI / 2));
      this.ctx.lineTo(r * Math.cos(a * i + Math.PI / 2), r * Math.sin(a * i + Math.PI / 2));
      this.ctx.lineTo(r * Math.cos(a * i + Math.PI / 6), r * Math.sin(a * i + Math.PI / 6));
      this.ctx.closePath();
      this.ctx.stroke();
      this.ctx.fill();
      this.drawZone(zone,
        {x: 0.2 * r * Math.cos(0), y: 0.2 * r * Math.sin(0)},
        {x: 0.2 * r * Math.cos(2 * Math.PI / 3), y: 0.2 * r * Math.sin(2 * Math.PI / 2)},
        {x: 0.2 * r * Math.cos(4 * Math.PI / 3), y: 0.2 * r * Math.sin(4 * Math.PI / 2)},
      );
      const axial_vector = indexToCoordinate(1, direction_vector);
      const index = coordinateToIndex(this.data.game.board_size, addPoint2D(this.data.space.coordinate, axial_vector));
      const adjacent_space = getSpace(this.data.game, index);
      this.ctx.strokeStyle = 'rgba(250, 250, 250, 0.8)';
      const fill_color = getSpaceFill(adjacent_space);
      this.ctx.fillStyle = fill_color.dAlpha(-0.2).getString();;
      this.ctx.beginPath();
      this.ctx.lineTo(r * Math.cos(a * i + Math.PI / 6), r * Math.sin(a * i + Math.PI / 6));
      this.ctx.lineTo(3 * r * Math.cos(a * i + Math.PI / 6), 3 * r * Math.sin(a * i + Math.PI / 6));
      this.ctx.lineTo(3 * r * Math.cos(a * i + Math.PI / 2), 3 * r * Math.sin(a * i + Math.PI / 2));
      this.ctx.lineTo(r * Math.cos(a * i + Math.PI / 2), r * Math.sin(a * i + Math.PI / 2));
      this.ctx.closePath();
      this.ctx.stroke();
      this.ctx.fill();
      // TODO: draw space summary
    }
    // Other method from https://rangevoting.org/PuzzGerryOpt.html
    /*drawHexagon(this.ctx, {x: 0, y: 0}, 0.4464 * r, 0);
    for (let i = 0; i < 6; i++) {
      // draw the pentagon above the central hexagon then rotate and repeat
      this.ctx.beginPath();
      this.ctx.lineTo(0.4464 * r * Math.cos(0), 0.4464 * r * Math.sin(0));
      this.ctx.lineTo(0.4464 * r * Math.cos(Math.PI / 3), 0.4464 * r * Math.sin(Math.PI / 3));
      this.ctx.lineTo(0.5 * 1.732 * r * Math.cos(Math.PI / 3), 0.5 * 1.732 * r * Math.sin(Math.PI / 3));
      this.ctx.lineTo(r * Math.cos(Math.PI / 6), r * Math.sin(Math.PI / 6));
      this.ctx.lineTo(0.5 * 1.732 * r * Math.cos(0), 0.5 * 1.732 * r * Math.sin(0));
      this.ctx.closePath();
      this.ctx.stroke();
      this.ctx.fill();
      this.ctx.rotate(Math.PI / 3);
    }*/
  }

  private mousemove(p: Point2D) {
    this.mouse = subtractPoint2D({x: p.x, y: p.y}, multiplyPoint2D(0.5, this.size));
    let new_hovered_zone: RisqZone|undefined = undefined;
    let new_hovered_space: RisqSpace|undefined = undefined;
    if (pointInHexagon(this.mouse, 0.4 * this.radius)) {
      new_hovered_zone = this.data.space.zones[1][1];
    } else {
      const angle = atangent(this.mouse.y, this.mouse.x);
      const index = Math.floor((angle + Math.PI / 6) / (Math.PI / 3));
      let direction_vector: Point2D = {x: 0, y: 0};
      switch(index) {
        case 0:
        case 6:
          direction_vector = {x: 1, y: 2};
          break;
        case 1:
          direction_vector = {x: 0, y: 1};
          break;
        case 2:
          direction_vector = {x: 0, y: 0};
          break;
        case 3:
          direction_vector = {x: 1, y: 0};
          break;
        case 4:
          direction_vector = {x: 2, y: 0};
          break;
        case 5:
          direction_vector = {x: 2, y: 1};
          break;
        default:
          console.error('Unknown zone hovered', angle);
          return;
      }
      if (pointInHexagon(this.mouse, this.radius)) {
        new_hovered_zone = this.data.space.zones[direction_vector.x][direction_vector.y];
      } else {
        const axial_vector = indexToCoordinate(1, direction_vector);
        const index = coordinateToIndex(this.data.game.board_size, addPoint2D(this.data.space.coordinate, axial_vector));
        new_hovered_space = getSpace(this.data.game, index);
      }
    }
    if (!equalsPoint2D(this.hovered_zone?.coordinate, new_hovered_zone?.coordinate)) {
      if (!!this.hovered_zone) {
        this.hovered_zone.clicked = false;
        this.hovered_zone.hovered = false;
      }
      this.hovered_zone = new_hovered_zone;
      if (!!this.hovered_zone) {
        this.hovered_zone.hovered = true;
      }
    }
    if (!equalsPoint2D(this.hovered_space?.coordinate, new_hovered_space?.coordinate)) {
      if (!!this.hovered_space) {
        this.hovered_space.clicked = false;
        this.hovered_space.hovered = false;
      }
      this.hovered_space = new_hovered_space;
      if (!!this.hovered_space) {
        this.hovered_space.hovered = true;
      }
    }
  }

  private mousedown(_e: MouseEvent) {
    if (!!this.hovered_zone) {
      this.hovered_zone.clicked = true;
    }
  }

  private mouseup(_e: MouseEvent) {
    if (!!this.hovered_zone) {
      this.hovered_zone.clicked = false;
    }
  }

  private keyup(e: KeyboardEvent) {
    switch(e.key) {
      case 'Escape':
        this.closeDialog();
        break;
      default:
        return;
    }
  }

  private setZoneFill(zone: RisqZone) {
    this.ctx.strokeStyle = 'rgba(250, 250, 250, 0.9)';
    const color = new ColorRGB(10, 120, 10, 0.8);
    if (zone.hovered) {
      if (zone.clicked) {
        color.addColor(210, 210, 210, 0.4);
      } else {
        color.addColor(190, 190, 190, 0.2);
      }
    }
    this.ctx.fillStyle = color.getString();
  }

  private setSpaceFill(space: RisqSpace) {
    this.ctx.strokeStyle = 'rgba(250, 250, 250, 0.8)';
    const color = new ColorRGB(0, 0, 0, 0);
    if (!!space) {
      if (space.hovered) {
        if (space.clicked) {
          color.addColor(210, 210, 210, 0.4);
        } else {
          color.addColor(190, 190, 190, 0.2);
        }
      } else {
        color.setColor(5, 90, 5, 0.7);
      }
    }
    this.ctx.fillStyle = color.getString();
  }

  private drawZone(zone: RisqZone, p1: Point2D, p2: Point2D, p3: Point2D) {
    //
  }
}

customElements.define('dwg-space-dialog', DwgSpaceDialog);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-space-dialog': DwgSpaceDialog;
  }
}
