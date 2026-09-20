import type { ColorRGB } from '../../../../scripts/color_rgb';

/** RGB value reserved in unit/building art to mark pixels swapped for the owning player's color */
export const PLAYER_COLOR_KEY = { r: 254, g: 12, b: 203 };

/** Standard native resolution for unit/building art */
export const PLAYER_ICON_SIZE = 256;

export class RisqImageCache {
  private canvases = new Map<string, HTMLCanvasElement>();
  private cursor_urls = new Map<string, string>();

  private buildImage(
    w: number,
    h: number,
    images: HTMLImageElement[],
    draw: (ctx: CanvasRenderingContext2D) => void
  ): HTMLCanvasElement | undefined {
    if (!images.every((img) => img.complete)) {
      return undefined;
    }
    const canvas = document.createElement('canvas');
    canvas.width = w;
    canvas.height = h;
    const ctx = canvas.getContext('2d');
    if (ctx) {
      draw(ctx);
    }
    return canvas;
  }

  getImage(
    key: string,
    size: number,
    images: HTMLImageElement[],
    draw: (ctx: CanvasRenderingContext2D) => void
  ): HTMLCanvasElement | undefined {
    return this.getRectImage(key, size, size, images, draw);
  }

  /** Same as getImage but for a non-square canvas, sized to the aspect ratio the draw callback actually needs */
  getRectImage(
    key: string,
    w: number,
    h: number,
    images: HTMLImageElement[],
    draw: (ctx: CanvasRenderingContext2D) => void
  ): HTMLCanvasElement | undefined {
    const cached = this.canvases.get(key);
    if (cached) {
      return cached;
    }
    const canvas = this.buildImage(w, h, images, draw);
    if (canvas) {
      this.canvases.set(key, canvas);
    }
    return canvas;
  }

  /** Returns img with every PLAYER_COLOR_KEY pixel swapped for color, cached per (key, color) */
  getPlayerColoredIcon(
    key: string,
    img: HTMLImageElement,
    size: number,
    color: ColorRGB
  ): HTMLCanvasElement | undefined {
    return this.getImage(`${key}|${color.getR()},${color.getG()},${color.getB()}`, size, [img], (ctx) => {
      ctx.drawImage(img, 0, 0, size, size);
      const image_data = ctx.getImageData(0, 0, size, size);
      const data = image_data.data;
      const { r: kr, g: kg, b: kb } = PLAYER_COLOR_KEY;
      const TOLERANCE = 1;
      for (let i = 0; i < data.length; i += 4) {
        const r = data[i];
        const g = data[i + 1];
        const b = data[i + 2];
        // matches the key color scaled toward black, so shaded/shadowed marker pixels recolor too
        const scale = r / kr;
        if (scale > 1 || Math.abs(g - kg * scale) > TOLERANCE || Math.abs(b - kb * scale) > TOLERANCE) {
          continue;
        }
        data[i] = color.getR() * scale;
        data[i + 1] = color.getG() * scale;
        data[i + 2] = color.getB() * scale;
      }
      ctx.putImageData(image_data, 0, 0);
    });
  }

  getCursorUrl(
    key: string,
    size: number,
    images: HTMLImageElement[],
    draw: (ctx: CanvasRenderingContext2D) => void
  ): string | undefined {
    const cached = this.cursor_urls.get(key);
    if (cached) {
      return cached;
    }
    const canvas = this.buildImage(size, size, images, draw);
    if (!canvas) {
      return undefined;
    }
    const url = canvas.toDataURL();
    this.cursor_urls.set(key, url);
    return url;
  }
}
