import type { Point2D } from '../../../../util/objects2d';
import type { WallDrawItem } from './wall_draw';

const MAX_SPRITE_PIXELS = 4096;
const MIN_SCALE = 1 / 64;

declare interface Entry {
  canvas: HTMLCanvasElement;
  scale: number;
  bytes: number;
  x: number;
  y: number;
}

function scaleBucket(scale: number): number {
  return 2 ** Math.ceil(Math.log2(Math.max(scale, MIN_SCALE)));
}

export class WallSpriteCache {
  private entries = new Map<string, Entry>();
  private total_bytes = 0;

  constructor(private budget_bytes: number) {}

  clear(): void {
    this.entries.clear();
    this.total_bytes = 0;
  }

  size(): number {
    return this.entries.size;
  }

  draw(ctx: CanvasRenderingContext2D, item: WallDrawItem, scale: number): boolean {
    const bucket = scaleBucket(scale);
    const key = `${item.key}|${bucket}`;
    let entry = this.entries.get(key);
    if (entry) {
      this.entries.delete(key);
    } else {
      entry = this.render(item, bucket);
      if (!entry) {
        return false;
      }
      this.total_bytes += entry.bytes;
    }
    this.entries.set(key, entry);
    this.evict();
    const at: Point2D = { x: item.anchor.x + entry.x, y: item.anchor.y + entry.y };
    ctx.drawImage(entry.canvas, at.x, at.y, entry.canvas.width / entry.scale, entry.canvas.height / entry.scale);
    return true;
  }

  private render(item: WallDrawItem, scale: number): Entry | undefined {
    const width = Math.ceil(item.bounds.w * scale);
    const height = Math.ceil(item.bounds.h * scale);
    if (width <= 0 || height <= 0 || width > MAX_SPRITE_PIXELS || height > MAX_SPRITE_PIXELS) {
      return undefined;
    }
    const canvas = document.createElement('canvas');
    canvas.width = width;
    canvas.height = height;
    const ctx = canvas.getContext('2d');
    if (!ctx) {
      return undefined;
    }
    ctx.scale(scale, scale);
    ctx.translate(-item.bounds.x, -item.bounds.y);
    item.draw(ctx);
    return { canvas, scale, bytes: width * height * 4, x: item.bounds.x, y: item.bounds.y };
  }

  private evict(): void {
    for (const [key, entry] of this.entries) {
      if (this.total_bytes <= this.budget_bytes) {
        return;
      }
      this.entries.delete(key);
      this.total_bytes -= entry.bytes;
    }
  }
}
