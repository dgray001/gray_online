import { DwgElement } from '../../../dwg_element';
import { isDialogOpen, isTypingInInput, until } from '../../../../scripts/util';
import { createImage, isImageReady } from '../../../../scripts/image';
import type { Point2D } from '../objects2d';
import { rotatePoint, subtractPoint2D } from '../objects2d';
import { configDraw } from '../canvas_components/canvas_component';
import { flushTooltipQueue, setTooltipCursor } from '../canvas_components/tooltip';

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

const EXTRA_MOUSE_BUTTON_EVENTS = ['mousedown', 'mouseup', 'auxclick'] as const;

function modifiersFrom(e: MouseEvent | KeyboardEvent): ModifierKeys {
  return { ctrl: e.ctrlKey, shift: e.shiftKey, alt: e.altKey };
}

/** Data describing how the canvas should be initialized */
export declare interface CanvasBoardInitializationData {
  board_size: Point2D;
  max_scale: number;
  min_scale?: number;
  fill_space?: boolean;
  allow_side_move?: boolean;
  draw: (ctx: CanvasRenderingContext2D, transform: BoardTransformData) => void;
  // returns whether something was scrolled
  scroll?: (dy: number, mode: number, dx?: number) => boolean;
  mousemove: (canvas: Point2D, screen: Point2D, transform: BoardTransformData, modifiers: ModifierKeys) => void;
  draggingCallback?: () => void;
  mouseleave: () => void;
  cancelInput?: () => void;
  // returns whether something was clicked
  mousedown: (e: MouseEvent) => boolean;
  mouseup: (e: MouseEvent) => void;
  // press/release of a mouse button past left/right/middle (browser back/forward), anywhere on the page
  extraMouseButton?: (e: MouseEvent) => void;
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
  private initialization_controller = new AbortController();
  private ctx!: CanvasRenderingContext2D;
  private data!: CanvasBoardInitializationData;
  private orig_size!: Point2D;
  private transform: BoardTransformData = defaultTransform();
  private zoom_config!: ZoomConfig;

  private hovered = false;
  private modifiers: ModifierKeys = { ctrl: false, shift: false, alt: false };
  private holding_keys: HoldingKeysData = {
    arrow_up: false,
    arrow_down: false,
    arrow_left: false,
    arrow_right: false,
  };
  private cursor_move_threshold = 5;
  // TODO: replace edge_arm_threshold/sticky_pan with document-level mouse tracking, so edge-pan is a
  // speed-independent viewport-boundary check instead of a threshold the cursor can jump past
  private edge_arm_threshold = 100;
  private sticky_pan = { up: false, down: false, left: false, right: false };
  private pan_suppressed = { x: false, y: false };
  private draw_interval?: ReturnType<typeof setInterval>;
  private dragging = false;
  private drag_button = 0;
  private dragged = false;
  private mouse: Point2D = { x: 0, y: 0 };
  private cursor_images = new Map<string, HTMLImageElement>();
  private cursor_image?: HTMLImageElement;
  private cursor_alpha = 1;

  private bounding_rect!: DOMRect;
  private resize_observer = new ResizeObserver(async (els) => {
    const signal = this.initialization_controller.signal;
    for (const el of els) {
      if (!(await this.updateSize(this.data, el.contentRect, signal)) || signal.aborted) {
        return;
      }
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

  getModifiers(): ModifierKeys {
    return { ...this.modifiers };
  }

  async initialize(data: CanvasBoardInitializationData): Promise<CanvasBoardSize | undefined> {
    this.initialization_controller.abort();
    this.initialization_controller = new AbortController();
    const signal = this.initialization_controller.signal;
    data.allow_side_move = data.allow_side_move ?? true;
    this.zoom_config = Object.assign({}, data.zoom_config);
    this.orig_size = {
      x: data.board_size.x,
      y: data.board_size.y,
    };
    const success = await this.updateSize(data, undefined, signal);
    if (!success || signal.aborted) {
      return undefined;
    }
    await until(() => !!this.canvas.getBoundingClientRect()?.width, 50, signal);
    if (signal.aborted) {
      return undefined;
    }
    this.setCursor('cursor');
    this.addEventListeners();
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
      setTooltipCursor(this.mouse);
      this.data.draw(this.ctx, this.transform);
      flushTooltipQueue();
      this.drawCursor();
    }, 20);
    this.initialized_successfully = true;
    return {
      board_size: this.data.board_size,
      el_size: this.bounding_rect,
    };
  }

  async updateSize(
    data: CanvasBoardInitializationData,
    override_rect?: DOMRect,
    signal: AbortSignal = this.initialization_controller.signal
  ): Promise<boolean> {
    if (!data || this.orig_size.x < 1 || this.orig_size.y < 1) {
      console.error('Size must be at least 1px in each direction');
      return false;
    }
    if (!data.max_scale || data.max_scale < 1) {
      console.error(`Max scale of ${data.max_scale} is invalid`);
      return false;
    }
    this.data = data;
    await until(() => this.fully_parsed, 50, signal);
    if (signal.aborted) {
      return false;
    }
    if (!this.canvas.getContext) {
      console.error('Browser does not support canvas; cannot draw board');
      return false;
    }
    await this.setSize(override_rect, signal);
    if (signal.aborted || !this.ctx) {
      return false;
    }
    return true;
  }

  private async setSize(rect?: DOMRect, signal: AbortSignal = this.initialization_controller.signal): Promise<void> {
    const data = this.data;
    if (!!rect) {
      this.bounding_rect = rect;
    } else {
      await until(
        () => {
          this.bounding_rect = this.getBoundingClientRect();
          return !!this.bounding_rect?.width;
        },
        50,
        signal
      );
      rect = this.bounding_rect;
    }
    if (signal.aborted) {
      return;
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
      this.modifiers = modifiersFrom(e);
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
      this.modifiers = modifiersFrom(e);
      this.hovered = true;
      const rect = this.canvas.getBoundingClientRect();
      const new_mouse = {
        x: e.clientX - rect.left,
        y: e.clientY - rect.top,
      };
      const old_mouse = this.mouse;
      const dif_mouse = subtractPoint2D(new_mouse, old_mouse);
      this.mouse = new_mouse;
      if (this.dragging && isDialogOpen()) {
        this.dragging = false;
      } else if (this.dragging) {
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
      this.modifiers = modifiersFrom(e);
      e.stopImmediatePropagation();
      if (isDialogOpen() || e.button > 2) {
        return;
      }
      if (this.data.mousedown(e)) {
        return;
      }
      if (e.button === 2 && e.detail >= 2) {
        this.setRotation(0);
        return;
      }
      this.dragging = true;
      this.drag_button = e.button;
    });
    this.addEventListener('mouseup', (e: MouseEvent) => {
      this.modifiers = modifiersFrom(e);
      e.stopImmediatePropagation();
      if (e.button > 2) {
        return;
      }
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
      this.sticky_pan = { up: false, down: false, left: false, right: false };
    });
    this.addEventListener('mouseleave', () => {
      this.hovered = false;
      if (this.data.allow_side_move && !this.dragging) {
        this.sticky_pan = {
          up: this.mouse.y < this.edge_arm_threshold,
          down: this.mouse.y > this.bounding_rect.height - this.edge_arm_threshold,
          left: this.mouse.x < this.edge_arm_threshold,
          right: this.mouse.x > this.bounding_rect.width - this.edge_arm_threshold,
        };
      }
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
    document.addEventListener('mousemove', this.handleDocumentMouseMove);
    document.addEventListener('mouseup', this.handleDocumentMouseUp);
    window.addEventListener('blur', this.handleBlur);
    for (const type of EXTRA_MOUSE_BUTTON_EVENTS) {
      window.addEventListener(type, this.handleExtraMouseButton, true);
    }
  }

  // swallowed while the board is mounted so they never trigger browser back/forward navigation
  private handleExtraMouseButton = (e: MouseEvent) => {
    this.modifiers = modifiersFrom(e);
    if (e.button <= 2) {
      return;
    }
    e.preventDefault();
    if (e.type !== 'auxclick') {
      this.data.extraMouseButton?.(e);
    }
  };

  private handleDocumentMouseMove = (e: MouseEvent) => {
    this.modifiers = modifiersFrom(e);
    const rect = this.canvas.getBoundingClientRect();
    this.mouse = { x: e.clientX - rect.left, y: e.clientY - rect.top };
  };

  private handleDocumentMouseUp = (e: MouseEvent): void => {
    if (e.button === this.drag_button) {
      this.dragging = false;
      this.dragged = false;
    }
  };

  private handleBlur = (): void => {
    this.modifiers = { ctrl: false, shift: false, alt: false };
    this.sticky_pan = { up: false, down: false, left: false, right: false };
    this.holding_keys = { arrow_up: false, arrow_down: false, arrow_left: false, arrow_right: false };
    this.dragging = false;
    this.dragged = false;
    this.hovered = false;
    this.data?.cancelInput?.();
  };

  private handleKeydown = (e: KeyboardEvent) => {
    this.modifiers = modifiersFrom(e);
    if (!this.hovered || isTypingInInput() || isDialogOpen()) {
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
    this.modifiers = modifiersFrom(e);
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
    this.initialization_controller.abort();
    this.initialized_successfully = false;
    super.disconnectedCallback();
    clearInterval(this.draw_interval);
    this.handleBlur();
    this.resize_observer.disconnect();
    document.body.removeEventListener('keydown', this.handleKeydown);
    document.body.removeEventListener('keyup', this.handleKeyup);
    document.removeEventListener('mousemove', this.handleDocumentMouseMove);
    document.removeEventListener('mouseup', this.handleDocumentMouseUp);
    window.removeEventListener('blur', this.handleBlur);
    for (const type of EXTRA_MOUSE_BUTTON_EVENTS) {
      window.removeEventListener(type, this.handleExtraMouseButton, true);
    }
  }

  private tick() {
    if (isDialogOpen()) {
      return;
    }
    const d_view = { x: 0, y: 0 };
    const arrow_key_speed = 20;
    let moved = false;
    const rect = this.bounding_rect;
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
    if (!this.dragging && this.data.allow_side_move && this.hovered && document.hasFocus()) {
      if (!this.pan_suppressed.y && this.mouse.y < this.cursor_move_threshold) {
        d_view.y -= arrow_key_speed;
        moved = true;
      }
      if (!this.pan_suppressed.y && this.mouse.y > rect.height - this.cursor_move_threshold) {
        d_view.y += arrow_key_speed;
        moved = true;
      }
      if (!this.pan_suppressed.x && this.mouse.x < this.cursor_move_threshold) {
        d_view.x -= arrow_key_speed;
        moved = true;
      }
      if (!this.pan_suppressed.x && this.mouse.x > rect.width - this.cursor_move_threshold) {
        d_view.x += arrow_key_speed;
        moved = true;
      }
    } else if (!this.dragging && this.data.allow_side_move && !this.hovered && document.hasFocus()) {
      if (this.sticky_pan.up) {
        d_view.y -= arrow_key_speed;
        moved = true;
      }
      if (this.sticky_pan.down) {
        d_view.y += arrow_key_speed;
        moved = true;
      }
      if (this.sticky_pan.left) {
        d_view.x -= arrow_key_speed;
        moved = true;
      }
      if (this.sticky_pan.right) {
        d_view.x += arrow_key_speed;
        moved = true;
      }
    }
    if (moved) {
      const rotated = rotatePoint(d_view, -this.transform.rotation);
      this.setView({
        x: this.transform.view.x + rotated.x,
        y: this.transform.view.y + rotated.y,
      });
      this.data.mousemove(this.mouseCanvasPoint(), this.mouse, this.transform, this.getModifiers());
    }
  }

  setCursor(image_path: string, alpha = 1) {
    this.setCursorUrl(`/images/cursors/${image_path}.png`, alpha);
  }

  setCursorUrl(url: string, alpha = 1) {
    this.cursor_alpha = alpha;
    let img = this.cursor_images.get(url);
    if (!img) {
      img = createImage(url);
      this.cursor_images.set(url, img);
    }
    this.cursor_image = img;
  }

  private drawCursor() {
    if (this.hovered && !isDialogOpen() && this.cursor_image && isImageReady(this.cursor_image)) {
      this.canvas.style.cursor = 'none';
      configDraw(
        this.ctx,
        this.transform,
        { fill_style: 'transparent', stroke_width: 0, fixed_position: true },
        false,
        false,
        () => {
          const img = this.cursor_image!;
          this.ctx.globalAlpha = this.cursor_alpha;
          this.ctx.drawImage(img, this.mouse.x, this.mouse.y, img.naturalWidth, img.naturalHeight);
          this.ctx.globalAlpha = 1;
        }
      );
    } else {
      this.canvas.style.cursor = 'auto';
    }
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

  /** Suppresses edge-of-screen panning along the given axes */
  setPanSuppressed(x: boolean, y: boolean) {
    this.pan_suppressed = { x, y };
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

  setMaxScale(max_scale: number, min_scale?: number): void {
    if (!max_scale || max_scale < 1) {
      return;
    }
    const scale_ratio = max_scale / this.data.max_scale;
    const preserve_scale = min_scale !== undefined && this.data.min_scale !== undefined;
    this.data.max_scale = max_scale;
    this.data.min_scale = min_scale;
    const scale = this.transform.scale;
    const next_scale = preserve_scale || scale > 1 ? scale * scale_ratio : scale < 1 ? scale / scale_ratio : scale;
    this.setScale(next_scale);
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
    const min_scale = this.data.min_scale ?? 1 / this.data.max_scale;
    if (scale < min_scale) {
      return min_scale;
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
