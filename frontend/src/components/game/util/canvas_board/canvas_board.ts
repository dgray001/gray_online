import { DwgElement } from '../../../dwg_element';
import { isTypingInInput, until } from '../../../../scripts/util';
import type { Point2D } from '../objects2d';
import { rotatePoint, subtractPoint2D } from '../objects2d';

import html from './canvas_board.html';

import './canvas_board.scss';

interface HoldingKeysData {
  arrow_up: boolean;
  arrow_down: boolean;
  arrow_left: boolean;
  arrow_right: boolean;
}

/** Modifier key state */
export declare interface ModifierKeys {
  ctrl: boolean;
  shift: boolean;
  alt: boolean;
}

function modifiersFrom(e: MouseEvent): ModifierKeys {
  return { ctrl: e.ctrlKey, shift: e.shiftKey, alt: e.altKey };
}

/** Data describing how the canvas should be initialized */
export declare interface CanvasBoardInitializationData {
  board_size: Point2D;
  max_scale: number;
  fill_space?: boolean;
  allow_side_move?: boolean;
  draw: (ctx: CanvasRenderingContext2D, transform: BoardTransformData) => void;
  // returns whether something was scrolled
  scroll?: (dy: number, mode: number, dx?: number) => boolean;
  mousemove: (canvas: Point2D, screen: Point2D, transform: BoardTransformData, modifiers: ModifierKeys) => void;
  draggingCallback?: () => void;
  mouseleave: () => void;
  // returns whether something was clicked
  mousedown: (e: MouseEvent) => boolean;
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
  offset: Point2D;
  rotation: number;
}

/** Returns a default board transform data object */
export function defaultTransform(): BoardTransformData {
  return {
    scale: 1,
    view: { x: 0, y: 0 },
    offset: { x: 0, y: 0 },
    rotation: 0,
  };
}

export function canvasToScreen(canvas: Point2D, transform: BoardTransformData): Point2D {
  const scaled = rotatePoint(
    {
      x: canvas.x * transform.scale - transform.view.x,
      y: canvas.y * transform.scale - transform.view.y,
    },
    transform.rotation
  );
  return {
    x: scaled.x + transform.offset.x,
    y: scaled.y + transform.offset.y,
  };
}

export function screenToCanvas(screen: Point2D, transform: BoardTransformData): Point2D {
  const unrotated = rotatePoint(
    {
      x: screen.x - transform.offset.x,
      y: screen.y - transform.offset.y,
    },
    -transform.rotation
  );
  return {
    x: (unrotated.x + transform.view.x) / transform.scale,
    y: (unrotated.y + transform.view.y) / transform.scale,
  };
}

/** Data describing how zoom can operate */
export declare interface ZoomConfig {
  zoom_constant: number;
  max_zoom?: number;
  min_zoom?: number;
}

export class DwgCanvasBoard extends DwgElement {
  private canvas!: HTMLCanvasElement;

  private initialized_successfully = false;
  private ctx!: CanvasRenderingContext2D;
  private data!: CanvasBoardInitializationData;
  private orig_size!: Point2D;
  private transform: BoardTransformData = defaultTransform();
  private zoom_config!: ZoomConfig;

  private hovered = false;
  private holding_keys: HoldingKeysData = {
    arrow_up: false,
    arrow_down: false,
    arrow_left: false,
    arrow_right: false,
  };
  private cursor_move_threshold = 5;
  private draw_interval?: ReturnType<typeof setInterval>;
  private dragging = false;
  private drag_button = 0;
  private dragged = false;
  private mouse: Point2D = { x: 0, y: 0 };
  private cursor_in_range = false;

  private bounding_rect!: DOMRect;
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
  }

  getBoundingRect(): DOMRect {
    return this.bounding_rect;
  }

  isInitialized(): boolean {
    return this.initialized_successfully;
  }

  async initialize(data: CanvasBoardInitializationData): Promise<CanvasBoardSize | undefined> {
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
    this.setCursor('cursor');
    this.addEventListeners();
    await until(() => !!this.canvas.getBoundingClientRect()?.width);
    this.resize_observer.observe(this);
    this.draw_interval = setInterval(() => {
      this.tick();
      // If ever made async everything after tick() needs to run at same time (just skip until last finished)
      this.ctx.resetTransform();
      this.ctx.fillStyle = 'black';
      this.ctx.fillRect(0, 0, this.data.board_size.x, this.data.board_size.y);
      this.ctx.translate(this.transform.offset.x, this.transform.offset.y);
      this.ctx.rotate(this.transform.rotation);
      this.ctx.translate(-this.transform.view.x, -this.transform.view.y);
      this.ctx.scale(this.transform.scale, this.transform.scale);
      this.data.draw(this.ctx, this.transform);
    }, 20);
    this.initialized_successfully = true;
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
    if (!this.ctx) {
      return false;
    }
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
    this.ctx = this.canvas.getContext('2d')!;
    this.canvas.style.setProperty('--w', `${data.board_size.x.toString()}px`);
    this.canvas.style.setProperty('--h', `${data.board_size.y.toString()}px`);
    this.canvas.width = data.board_size.x;
    this.canvas.height = data.board_size.y;
  }

  private mouseCanvasPoint(): Point2D {
    return screenToCanvas(this.mouse, this.transform);
  }

  private addEventListeners() {
    this.addEventListener('wheel', (e: WheelEvent) => {
      if (this.data.scroll) {
        if (this.data.scroll(e.deltaY, e.deltaMode, e.deltaX)) {
          return;
        }
      }
      let zoom = 1 + e.deltaY / this.zoom_config.zoom_constant;
      if (!!this.zoom_config.max_zoom && this.zoom_config.max_zoom < zoom) {
        zoom = this.zoom_config.max_zoom;
      }
      if (!!this.zoom_config.min_zoom && this.zoom_config.min_zoom > zoom) {
        zoom = this.zoom_config.min_zoom;
      }
      this.zoomTowardPoint(this.transform.scale / zoom, this.mouse);
      this.data.mousemove(this.mouseCanvasPoint(), this.mouse, this.transform, modifiersFrom(e));
    });
    this.addEventListener('mousemove', (e: MouseEvent) => {
      this.hovered = true;
      const rect = this.canvas.getBoundingClientRect();
      const new_mouse = {
        x: e.clientX - rect.left,
        y: e.clientY - rect.top,
      };
      const old_mouse = this.mouse;
      const dif_mouse = subtractPoint2D(new_mouse, old_mouse);
      this.mouse = new_mouse;
      if (this.dragging) {
        if (this.drag_button === 2) {
          const pivot = this.transform.offset;
          const prev_angle = Math.atan2(old_mouse.y - pivot.y, old_mouse.x - pivot.x);
          const new_angle = Math.atan2(new_mouse.y - pivot.y, new_mouse.x - pivot.x);
          this.setRotation(this.transform.rotation + (new_angle - prev_angle));
        } else {
          this.setView(subtractPoint2D(this.transform.view, rotatePoint(dif_mouse, -this.transform.rotation)));
        }
        if (this.data.draggingCallback) {
          this.data.draggingCallback();
        }
        this.dragged = true;
      } else {
        this.data.mousemove(this.mouseCanvasPoint(), this.mouse, this.transform, modifiersFrom(e));
      }
    });
    this.addEventListener('mousedown', (e: MouseEvent) => {
      e.stopImmediatePropagation();
      if (e.button === 2 && e.detail >= 2) {
        this.setRotation(0);
        return;
      }
      if (!this.data.mousedown(e)) {
        this.dragging = true;
        this.drag_button = e.button;
      }
    });
    this.addEventListener('mouseup', (e: MouseEvent) => {
      e.stopImmediatePropagation();
      this.dragging = false;
      if (this.dragged) {
        this.data.mousemove(this.mouseCanvasPoint(), this.mouse, this.transform, modifiersFrom(e));
        this.dragged = false;
      } else {
        this.data.mouseup(e);
      }
    });
    this.addEventListener('mouseenter', () => {
      this.hovered = true;
    });
    this.addEventListener('mouseleave', () => {
      this.hovered = false;
      this.dragging = false;
      this.dragged = false;
      this.data.mouseleave();
    });
    this.addEventListener(
      'contextmenu',
      (e) => {
        e.preventDefault();
      },
      false
    );
    document.body.addEventListener('keydown', this.handleKeydown);
    document.body.addEventListener('keyup', this.handleKeyup);
  }

  private handleKeydown = (e: KeyboardEvent) => {
    if (!this.hovered || isTypingInInput()) {
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
  };

  private handleKeyup = (e: KeyboardEvent) => {
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
  };

  override disconnectedCallback(): void {
    super.disconnectedCallback();
    clearInterval(this.draw_interval);
    this.resize_observer.disconnect();
    document.body.removeEventListener('keydown', this.handleKeydown);
    document.body.removeEventListener('keyup', this.handleKeyup);
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
      const rotated = rotatePoint(d_view, -this.transform.rotation);
      this.setView({
        x: this.transform.view.x + rotated.x,
        y: this.transform.view.y + rotated.y,
      });
      this.data.mousemove(this.mouseCanvasPoint(), this.mouse, this.transform, {
        ctrl: false,
        shift: false,
        alt: false,
      });
    }
  }

  setCursor(image_path: string) {
    this.setCursorUrl(`/images/cursors/${image_path}.png`);
  }

  setCursorUrl(url: string) {
    this.canvas.style.cursor = `url("${url}") 0 0, auto`;
  }

  scaleView(scale: number) {
    this.setView({
      x: this.transform.view.x * scale,
      y: this.transform.view.y * scale,
    });
  }

  setOffset(offset: Point2D) {
    this.transform.offset = offset;
  }

  setRotation(rotation: number) {
    this.transform.rotation = rotation;
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
    scale = this.clampScale(scale);
    if (!scale) {
      return this.transform.scale;
    }
    this.transform.view.x *= scale / this.transform.scale;
    this.transform.view.y *= scale / this.transform.scale;
    this.transform.scale = scale;
    return scale;
  }

  zoomTowardPoint(scale: number, anchor: Point2D): number {
    scale = this.clampScale(scale);
    if (!scale) {
      return this.transform.scale;
    }
    const factor = scale / this.transform.scale;
    this.transform.scale = scale;
    const a = rotatePoint(
      { x: anchor.x - this.transform.offset.x, y: anchor.y - this.transform.offset.y },
      -this.transform.rotation
    );
    this.setView({
      x: factor * this.transform.view.x + (factor - 1) * a.x,
      y: factor * this.transform.view.y + (factor - 1) * a.y,
    });
    return scale;
  }

  private clampScale(scale: number): number {
    if (!scale) {
      return 0;
    }
    if (scale < 1 / this.data.max_scale) {
      return 1 / this.data.max_scale;
    } else if (scale > this.data.max_scale) {
      return this.data.max_scale;
    }
    return scale;
  }
}

customElements.define('dwg-canvas-board', DwgCanvasBoard);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-canvas-board': DwgCanvasBoard;
  }
}
