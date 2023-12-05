import {DwgElement} from '../../../dwg_element';
import {until} from '../../../../scripts/util';
import {Point2D, addPoint2D, subtractPoint2D} from '../objects2d';

import html from './canvas_board.html';

import './canvas_board.scss';

interface HoldingKeysData {
  arrow_up: boolean;
  arrow_down: boolean;
  arrow_left: boolean;
  arrow_right: boolean;
}

/** Data describing how the canvas should be initialized */
export declare interface CanvasBoardInitializationData {
  size: Point2D;
  max_scale: number;
  fill_space?: boolean;
  draw: (ctx: CanvasRenderingContext2D, transform: BoardTransformData) => void;
  mousemove: (m: Point2D, transform: BoardTransformData) => void;
  mouseleave: () => void;
  mousedown: (e: MouseEvent) => boolean; // returns whether something was clicked
  mouseup: (e: MouseEvent) => void;
}

/** Data how the board is transformed */
export declare interface BoardTransformData {
  scale: number;
  view: Point2D;
}

export class DwgCanvasBoard extends DwgElement {
  canvas: HTMLCanvasElement;
  cursor: HTMLImageElement;

  ctx: CanvasRenderingContext2D;
  data: CanvasBoardInitializationData;
  transform: BoardTransformData = {
    scale: 1,
    view: {x: 0, y: 0},
  };

  hovered = false;
  holding_keys: HoldingKeysData = {
    arrow_up: false,
    arrow_down: false,
    arrow_left: false,
    arrow_right: false,
  };
  cursor_move_threshold = 5;
  dragging = false;
  mouse: Point2D = {x: 0, y: 0};

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('canvas');
    this.configureElement('cursor');
  }

  async initialize(data: CanvasBoardInitializationData) {
    if (data.size.x < 1 || data.size.y < 1) {
      console.error('Size must be at least 1px in each direction');
      return;
    }
    this.data = data;
    await until(() => this.fully_parsed);
    if (!this.canvas.getContext) {
      console.error('Browser does not support canvas; cannot draw board');
      return;
    }
    if (data.fill_space) {
      await until(() => !!this.getBoundingClientRect().width);
      const rect = this.getBoundingClientRect();
      const aspect_ratio = data.size.x / data.size.y;
      data.size.x = Math.max(data.size.x, rect.width);
      data.size.y = Math.max(data.size.y, rect.height);
      const new_ratio = data.size.x / data.size.y;
      if (new_ratio > aspect_ratio) {
        data.size.y = data.size.x / aspect_ratio;
      } else {
        data.size.x = data.size.y * aspect_ratio;
      }
    }
    this.ctx = this.canvas.getContext('2d');
    this.canvas.style.setProperty('--w', `${data.size.x.toString()}px`);
    this.canvas.style.setProperty('--h', `${data.size.y.toString()}px`);
    this.canvas.width = data.size.x;
    this.canvas.height = data.size.y;
    this.cursor.src = '/images/cursors/cursor.png';
    this.addEventListener('wheel', (e: WheelEvent) => {
      const zoom = 1 + e.deltaY / 250;
      this.setScale(this.transform.scale / zoom);
      this.data.mousemove({
        x: (this.mouse.x + this.transform.view.x) / this.transform.scale,
        y: (this.mouse.y + this.transform.view.y) / this.transform.scale,
      }, this.transform);
    });
    this.addEventListener('mousemove', (e: MouseEvent) => {
      this.hovered = true;
      this.cursor.style.setProperty('--x', `${e.clientX.toString()}px`);
      this.cursor.style.setProperty('--y', `${e.clientY.toString()}px`);
      const rect = this.canvas.getBoundingClientRect();
      const new_mouse = {
        x: (e.clientX - rect.left),
        y: (e.clientY - rect.top),
      };
      const dif_mouse = subtractPoint2D(new_mouse, this.mouse);
      this.mouse = new_mouse;
      if (this.dragging) {
        this.setView(subtractPoint2D(this.transform.view, dif_mouse));
      } else {
        this.data.mousemove({
          x: (this.mouse.x + this.transform.view.x) / this.transform.scale,
          y: (this.mouse.y + this.transform.view.y) / this.transform.scale,
        }, this.transform);
      }
    });
    this.addEventListener('mousedown', (e: MouseEvent) => {
      e.stopImmediatePropagation();
      if (!this.data.mousedown(e)) {
        this.dragging = true;
      }
    });
    this.addEventListener('mouseup', (e: MouseEvent) => {
      e.stopImmediatePropagation();
      this.dragging = false;
      this.data.mouseup(e);
    });
    this.addEventListener('mouseenter', () => {
      this.hovered = true;
      this.cursor.classList.remove('hide');
    });
    this.addEventListener('mouseleave', () => {
      this.hovered = false;
      this.dragging = false;
      this.cursor.classList.add('hide');
      this.data.mouseleave();
    });
    this.addEventListener('contextmenu', (e) => {
      e.preventDefault();
    }, false);
    document.body.addEventListener('keydown', (e) => {
      if (!this.hovered) {
        return;
      }
      switch(e.key) {
        case 'ArrowUp':
          this.holding_keys.arrow_up = true;
          break;
        case 'ArrowDown':
          this.holding_keys.arrow_down = true;
          break;
        case 'ArrowLeft':
          this.holding_keys.arrow_left = true;
          break;
        case 'ArrowRight':
          this.holding_keys.arrow_right = true;
          break;
        default:
          break;
      }
    });
    document.body.addEventListener('keyup', (e) => {
      if (!this.hovered) {
        return;
      }
      switch(e.key) {
        case 'ArrowUp':
          this.holding_keys.arrow_up = false;
          break;
        case 'ArrowDown':
          this.holding_keys.arrow_down = false;
          break;
        case 'ArrowLeft':
          this.holding_keys.arrow_left = false;
          break;
        case 'ArrowRight':
          this.holding_keys.arrow_right = false;
          break;
        default:
          break;
      }
    });
    await until(() => !!this.canvas.getBoundingClientRect().width);
    setInterval(() => {
      this.tick();
      // TODO: ensure last frame is done drawing (need to do tick either way)
      this.ctx.resetTransform();
      this.ctx.fillStyle = "black";
      this.ctx.fillRect(0, 0, this.data.size.x, this.data.size.y);
      this.ctx.translate(-this.transform.view.x, -this.transform.view.y);
      this.ctx.scale(this.transform.scale, this.transform.scale);
      this.data.draw(this.ctx, this.transform);
    }, 20);
  }

  private tick() {
    const d_view = {x: 0, y: 0};
    const arrow_key_speed = 20 * this.transform.scale;
    let moved = false;
    const rect = this.getBoundingClientRect();
    if (this.holding_keys.arrow_up) {
      d_view.y -= arrow_key_speed;
      moved = true;
    }
    if (this.holding_keys.arrow_down) {
      d_view.y += arrow_key_speed;
      moved = true;
    }
    if (this.holding_keys.arrow_left) {
      d_view.x -= arrow_key_speed;
      moved = true;
    }
    if (this.holding_keys.arrow_right) {
      d_view.x += arrow_key_speed;
      moved = true;
    }
    if (!this.dragging) {
      if (this.mouse.y < this.cursor_move_threshold) {
        d_view.y -= arrow_key_speed;
        moved = true;
      }
      if (this.mouse.y > rect.height - this.cursor_move_threshold) {
        d_view.y += arrow_key_speed;
        moved = true;
      }
      if (this.mouse.x < this.cursor_move_threshold) {
        d_view.x -= arrow_key_speed;
        moved = true;
      }
      if (this.mouse.x > rect.width - this.cursor_move_threshold) {
        d_view.x += arrow_key_speed;
        moved = true;
      }
    }
    if (moved) {
      this.setView({x: this.transform.view.x + d_view.x, y: this.transform.view.y + d_view.y});
      this.data.mousemove({
        x: (this.mouse.x + this.transform.view.x) / this.transform.scale,
        y: (this.mouse.y + this.transform.view.y) / this.transform.scale,
      }, this.transform);
    }
  }

  private setView(view: Point2D) {
    if (view.x < 0) {
      view.x = 0;
    } else if (view.x > this.data.size.x * this.transform.scale) {
      view.x = this.data.size.x * this.transform.scale;
    }
    if (view.y < 0) {
      view.y = 0;
    } else if (view.y > this.data.size.y * this.transform.scale) {
      view.y = this.data.size.y * this.transform.scale;
    }
    this.transform.view = view;
  }

  private setScale(scale: number) {
    if (scale < 1 / this.data.max_scale) {
      scale = 1 / this.data.max_scale;
    } else if (scale > this.data.max_scale) {
      scale = this.data.max_scale;
    }
    this.transform.view.x *= scale / this.transform.scale;
    this.transform.view.y *= scale / this.transform.scale;
    this.transform.scale = scale;
  }
}

customElements.define('dwg-canvas-board', DwgCanvasBoard);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-canvas-board': DwgCanvasBoard;
  }
}
