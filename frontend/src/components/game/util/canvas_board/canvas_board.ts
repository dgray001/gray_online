import { DwgElement } from '../../../dwg_element';
import { until } from '../../../../scripts/util';
import type { Point2D } from '../objects2d';
import { subtractPoint2D } from '../objects2d';

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
  board_size: Point2D;
  max_scale: number;
  fill_space?: boolean;
  allow_side_move?: boolean;
  draw: (ctx: CanvasRenderingContext2D, transform: BoardTransformData) => void;
  mousemove: (m: Point2D, transform: BoardTransformData) => void;
  mouseleave: () => void;
  mousedown: (e: MouseEvent) => boolean; // returns whether something was clicked
  mouseup: (e: MouseEvent) => void;
  zoom_config: ZoomConfig;
}

/** Data describing the size of a the board */
export declare interface CanvasBoardSize {
  board_size: Point2D;
  el_size: DOMRect;
}

/** Data how the board is transformed */
export declare interface BoardTransformData {
  scale: number;
  view: Point2D;
}

/** Data describing how zoom can operate */
export declare interface ZoomConfig {
  zoom_constant: number;
  max_zoom?: number;
  min_zoom?: number;
}

export class DwgCanvasBoard extends DwgElement {
  private canvas: HTMLCanvasElement;
  private cursor: HTMLImageElement;

  private ctx: CanvasRenderingContext2D;
  private data: CanvasBoardInitializationData;
  private orig_size: Point2D;
  private transform: BoardTransformData = {
    scale: 1,
    view: { x: 0, y: 0 },
  };
  private zoom_config: ZoomConfig;

  private hovered = false;
  private holding_keys: HoldingKeysData = {
    arrow_up: false,
    arrow_down: false,
    arrow_left: false,
    arrow_right: false,
  };
  private cursor_move_threshold = 5;
  private dragging = false;
  private mouse: Point2D = { x: 0, y: 0 };
  private cursor_in_range = false;

  private bounding_rect: DOMRect;
  private resize_observer = new ResizeObserver(async (els) => {
    for (const el of els) {
      await this.updateSize(this.data, el.contentRect);
      this.dispatchEvent(
        new CustomEvent<CanvasBoardSize>('canvas_resize', {
          detail: {
            board_size: this.data.board_size,
            el_size: this.bounding_rect,
          },
          bubbles: true,
        })
      );
    }
  });

  constructor() {
    super();
    this.html_string = html;
    this.configureElement('canvas');
    this.configureElement('cursor');
  }

  getBoundingRect(): DOMRect {
    return this.bounding_rect;
  }

  async initialize(data: CanvasBoardInitializationData): Promise<CanvasBoardSize> {
    data.allow_side_move = data.allow_side_move ?? true;
    this.zoom_config = Object.assign({}, data.zoom_config);
    this.orig_size = {
      x: data.board_size.x,
      y: data.board_size.y,
    };
    const success = await this.updateSize(data);
    if (!success) {
      return undefined;
    }
    this.cursor.src = '/images/cursors/cursor.png';
    this.addEventListeners();
    await until(() => !!this.canvas.getBoundingClientRect()?.width);
    this.resize_observer.observe(this);
    setInterval(() => {
      this.tick();
      // If ever made async everything after tick() needs to run at same time (just skip until last finished)
      this.ctx.resetTransform();
      this.ctx.fillStyle = 'black';
      this.ctx.fillRect(0, 0, this.data.board_size.x, this.data.board_size.y);
      this.ctx.translate(-this.transform.view.x, -this.transform.view.y);
      this.ctx.scale(this.transform.scale, this.transform.scale);
      this.data.draw(this.ctx, this.transform);
    }, 20);
    return {
      board_size: this.data.board_size,
      el_size: this.bounding_rect,
    };
  }

  async updateSize(data: CanvasBoardInitializationData, override_rect?: DOMRect): Promise<boolean> {
    if (!data || this.orig_size.x < 1 || this.orig_size.y < 1) {
      console.error('Size must be at least 1px in each direction');
      return false;
    }
    if (!data.max_scale || data.max_scale < 1) {
      console.error(`Max scale of ${data.max_scale} is invalid`);
      return false;
    }
    this.data = data;
    await until(() => this.fully_parsed);
    if (!this.canvas.getContext) {
      console.error('Browser does not support canvas; cannot draw board');
      return false;
    }
    await this.setSize(override_rect);
    return true;
  }

  private async setSize(rect?: DOMRect) {
    const data = this.data;
    if (!!rect) {
      this.bounding_rect = rect;
    } else {
      await until(() => {
        this.bounding_rect = this.getBoundingClientRect();
        return !!this.bounding_rect?.width;
      });
      rect = this.bounding_rect;
    }
    if (data.fill_space) {
      const aspect_ratio = this.orig_size.x / this.orig_size.y;
      data.board_size.x = Math.max(this.orig_size.x, rect.width);
      data.board_size.y = Math.max(this.orig_size.y, rect.height);
      const new_ratio = data.board_size.x / data.board_size.y;
      if (new_ratio > aspect_ratio) {
        data.board_size.y = data.board_size.x / aspect_ratio;
      } else {
        data.board_size.x = data.board_size.y * aspect_ratio;
      }
    }
    this.ctx = this.canvas.getContext('2d');
    this.canvas.style.setProperty('--w', `${data.board_size.x.toString()}px`);
    this.canvas.style.setProperty('--h', `${data.board_size.y.toString()}px`);
    this.canvas.width = data.board_size.x;
    this.canvas.height = data.board_size.y;
  }

  private addEventListeners() {
    this.addEventListener('wheel', (e: WheelEvent) => {
      let zoom = 1 + e.deltaY / this.zoom_config.zoom_constant;
      if (!!this.zoom_config.max_zoom && this.zoom_config.max_zoom < zoom) {
        zoom = this.zoom_config.max_zoom;
      }
      if (!!this.zoom_config.min_zoom && this.zoom_config.min_zoom > zoom) {
        zoom = this.zoom_config.min_zoom;
      }
      this.setScale(this.transform.scale / zoom);
      this.data.mousemove(
        {
          x: (this.mouse.x + this.transform.view.x) / this.transform.scale,
          y: (this.mouse.y + this.transform.view.y) / this.transform.scale,
        },
        this.transform
      );
    });
    this.addEventListener('mousemove', (e: MouseEvent) => {
      this.hovered = true;
      this.cursor.style.setProperty('--x', `${e.clientX.toString()}px`);
      this.cursor.style.setProperty('--y', `${e.clientY.toString()}px`);
      const rect = this.canvas.getBoundingClientRect();
      const new_mouse = {
        x: e.clientX - rect.left,
        y: e.clientY - rect.top,
      };
      const dif_mouse = subtractPoint2D(new_mouse, this.mouse);
      this.mouse = new_mouse;
      if (this.dragging) {
        this.setView(subtractPoint2D(this.transform.view, dif_mouse));
      } else {
        this.data.mousemove(
          {
            x: (this.mouse.x + this.transform.view.x) / this.transform.scale,
            y: (this.mouse.y + this.transform.view.y) / this.transform.scale,
          },
          this.transform
        );
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
    this.addEventListener(
      'contextmenu',
      (e) => {
        e.preventDefault();
      },
      false
    );
    document.body.addEventListener('keydown', (e) => {
      if (!this.hovered) {
        return;
      }
      switch (e.key) {
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
      switch (e.key) {
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
  }

  private tick() {
    const d_view = { x: 0, y: 0 };
    const arrow_key_speed = 20 * this.transform.scale;
    let moved = false;
    const rect = this.canvas.getBoundingClientRect();
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
    if (!this.dragging && this.data.allow_side_move) {
      const maybe_cursor_in_range = !this.cursor_in_range && !moved;
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
      if (maybe_cursor_in_range) {
        if (moved) {
          moved = false;
        } else {
          this.cursor_in_range = true;
        }
      }
    }
    if (moved) {
      this.setView({
        x: this.transform.view.x + d_view.x,
        y: this.transform.view.y + d_view.y,
      });
      this.data.mousemove(
        {
          x: (this.mouse.x + this.transform.view.x) / this.transform.scale,
          y: (this.mouse.y + this.transform.view.y) / this.transform.scale,
        },
        this.transform
      );
    }
  }

  scaleView(scale: number) {
    this.setView({
      x: this.transform.view.x * scale,
      y: this.transform.view.y * scale,
    });
  }

  setView(view: Point2D) {
    if (isNaN(view.x) || isNaN(view.y)) {
      return;
    }
    if (view.x < 0) {
      view.x = 0;
    } else if (view.x > this.data.board_size.x * this.transform.scale) {
      view.x = this.data.board_size.x * this.transform.scale;
    }
    if (view.y < 0) {
      view.y = 0;
    } else if (view.y > this.data.board_size.y * this.transform.scale) {
      view.y = this.data.board_size.y * this.transform.scale;
    }
    this.transform.view = view;
  }

  getMaxScale(): number {
    return this.data.max_scale;
  }

  setMaxScale(max_scale: number) {
    if (!max_scale || max_scale < 1) {
      return;
    }
    const scale_ratio = max_scale / this.data.max_scale;
    this.data.max_scale = max_scale;
    if (!!scale_ratio) {
      if (this.transform.scale > 1) {
        this.setScale(this.transform.scale * scale_ratio);
      } else if (this.transform.scale < 1) {
        this.setScale(this.transform.scale / scale_ratio);
      }
    } else {
      this.setScale(this.transform.scale);
    }
  }

  setScale(scale: number): number {
    if (!scale) {
      return this.transform.scale;
    }
    if (scale < 1 / this.data.max_scale) {
      scale = 1 / this.data.max_scale;
    } else if (scale > this.data.max_scale) {
      scale = this.data.max_scale;
    }
    this.transform.view.x *= scale / this.transform.scale;
    this.transform.view.y *= scale / this.transform.scale;
    this.transform.scale = scale;
    return scale;
  }
}

customElements.define('dwg-canvas-board', DwgCanvasBoard);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-canvas-board': DwgCanvasBoard;
  }
}
