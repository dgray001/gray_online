export class RisqImageCache {
  private canvases = new Map<string, HTMLCanvasElement>();
  private cursor_urls = new Map<string, string>();

  private buildImage(
    size: number,
    images: HTMLImageElement[],
    draw: (ctx: CanvasRenderingContext2D) => void
  ): HTMLCanvasElement | undefined {
    if (!images.every((img) => img.complete)) {
      return undefined;
    }
    const canvas = document.createElement('canvas');
    canvas.width = size;
    canvas.height = size;
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
    const cached = this.canvases.get(key);
    if (cached) {
      return cached;
    }
    const canvas = this.buildImage(size, images, draw);
    if (canvas) {
      this.canvases.set(key, canvas);
    }
    return canvas;
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
    const canvas = this.buildImage(size, images, draw);
    if (!canvas) {
      return undefined;
    }
    const url = canvas.toDataURL();
    this.cursor_urls.set(key, url);
    return url;
  }
}
