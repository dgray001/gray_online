import type { BoardTransformData } from '../../canvas_board/canvas_board';
import type { Point2D } from '../../objects2d';
import type { CanvasComponent } from '../canvas_component';

const CARET_BLINK_MS = 530;

export declare interface InputConfig {
  value?: string;
  max_length?: number;
  allowed_chars?: RegExp;
  start_disabled?: boolean;
}

export abstract class DwgInput implements CanvasComponent {
  private hovering = false;
  private clicking = false;
  private focused = false;
  private disabled: boolean;
  private text: string;
  private caret: number;
  private caret_timer = 0;
  private config: InputConfig;

  constructor(config: InputConfig) {
    this.config = config;
    this.disabled = !!config.start_disabled;
    this.text = this.sanitize(config.value ?? '');
    this.caret = this.text.length;
  }

  value(): string {
    return this.text;
  }

  setValue(value: string): void {
    this.text = this.sanitize(value);
    this.caret = this.text.length;
  }

  protected caretIndex(): number {
    return this.caret;
  }

  protected caretVisible(): boolean {
    return this.focused && this.caret_timer < CARET_BLINK_MS;
  }

  isFocused(): boolean {
    return this.focused;
  }

  focus(): void {
    if (this.focused || this.disabled) {
      return;
    }
    this.focused = true;
    this.caret_timer = 0;
    document.addEventListener('keydown', this.handleKeydown, true);
    document.addEventListener('paste', this.handlePaste, true);
  }

  blur(): void {
    if (!this.focused) {
      return;
    }
    this.focused = false;
    document.removeEventListener('keydown', this.handleKeydown, true);
    document.removeEventListener('paste', this.handlePaste, true);
  }

  isHovering(): boolean {
    return this.hovering;
  }

  setHovering(hovering: boolean): void {
    this.hovering = hovering;
  }

  isClicking(): boolean {
    return this.clicking;
  }

  setClicking(clicking: boolean): void {
    this.clicking = clicking;
  }

  isDisabled(): boolean {
    return this.disabled;
  }

  disable(): void {
    this.disabled = true;
    this.hovering = false;
    this.clicking = false;
    this.blur();
  }

  enable(): void {
    this.disabled = false;
  }

  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void {
    this.caret_timer = (this.caret_timer + dt) % (2 * CARET_BLINK_MS);
    this._draw(ctx, transform, dt);
  }

  mousemove(canvas: Point2D, screen: Point2D, _transform: BoardTransformData): boolean {
    this.hovering = !this.disabled && this.mouseOver(canvas, screen);
    return this.hovering;
  }

  mousedown(e: MouseEvent): boolean {
    if (!this.disabled && e.button === 0 && this.hovering) {
      this.clicking = true;
      this.focus();
      this.moveCaret(this.text.length);
      return true;
    }
    this.blur();
    return false;
  }

  mouseup(_e: MouseEvent): void {
    this.clicking = false;
  }

  private sanitize(s: string): string {
    const allowed = this.config.allowed_chars;
    const filtered = [...s].filter((c) => !allowed || allowed.test(c)).join('');
    return filtered.slice(0, this.config.max_length ?? filtered.length);
  }

  private insert(s: string): void {
    const room = (this.config.max_length ?? Infinity) - this.text.length;
    const added = this.sanitize(s).slice(0, Math.max(0, room));
    if (!added) {
      return;
    }
    this.text = this.text.slice(0, this.caret) + added + this.text.slice(this.caret);
    this.moveCaret(this.caret + added.length);
    this.changed(this.text);
  }

  private removeAt(index: number): void {
    if (index < 0 || index >= this.text.length) {
      return;
    }
    this.text = this.text.slice(0, index) + this.text.slice(index + 1);
    this.moveCaret(Math.min(this.caret, index));
    this.changed(this.text);
  }

  private moveCaret(index: number): void {
    this.caret = Math.min(Math.max(index, 0), this.text.length);
    this.caret_timer = 0;
  }

  private applyKey(key: string): boolean {
    switch (key) {
      case 'Enter':
        this.submitted(this.text);
        this.blur();
        return true;
      case 'Escape':
        this.blur();
        return true;
      case 'Backspace':
        this.removeAt(this.caret - 1);
        return true;
      case 'Delete':
        this.removeAt(this.caret);
        return true;
      case 'ArrowLeft':
        this.moveCaret(this.caret - 1);
        return true;
      case 'ArrowRight':
        this.moveCaret(this.caret + 1);
        return true;
      case 'Home':
        this.moveCaret(0);
        return true;
      case 'End':
        this.moveCaret(this.text.length);
        return true;
      default:
        if (key.length !== 1) {
          return false;
        }
        this.insert(key);
        return true;
    }
  }

  private handleKeydown = (e: KeyboardEvent): void => {
    if (e.ctrlKey || e.metaKey || e.altKey || !this.applyKey(e.key)) {
      return;
    }
    e.preventDefault();
    e.stopImmediatePropagation();
  };

  private handlePaste = (e: ClipboardEvent): void => {
    this.insert(e.clipboardData?.getData('text') ?? '');
    e.preventDefault();
    e.stopImmediatePropagation();
  };

  protected abstract _draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void;
  abstract mouseOver(canvas: Point2D, screen: Point2D): boolean;

  protected abstract changed(value: string): void;
  protected abstract submitted(value: string): void;

  abstract xi(): number;
  abstract xf(): number;
  abstract yi(): number;
  abstract yf(): number;
  abstract w(): number;
  abstract h(): number;
}
