import type { DwgCanvasBoard } from '../../../../util/canvas_board/canvas_board';
import type { Point2D } from '../../../../util/objects2d';
import { rotatePoint } from '../../../../util/objects2d';
import type { ReplayEvent } from '../../model/replay_event';
import type { RisqTickLocation } from '../../model/tick_actions';
import type { RisqViewport } from '../../rendering/board/viewport';
import { zoneCenterOffset } from '../../rendering/zones/geometry';

export class RisqReplayCamera {
  private frame = 0;

  constructor(private board: () => DwgCanvasBoard) {}

  focus(event: ReplayEvent, viewport: RisqViewport): void {
    const points = event.map_position
      ? [viewport.coordinateToCanvas(event.map_position)]
      : event.locations.map((location: RisqTickLocation): Point2D => {
          const center = viewport.coordinateToCanvas(location.space);
          const offset = zoneCenterOffset(location.zone, viewport.hexR());
          return { x: center.x + offset.x, y: center.y + offset.y };
        });
    this.frameEvent(points, viewport);
  }

  move(center: Point2D, target_scale: number): void {
    this.cancel();
    const board = this.board();
    const transform = board.getTransform();
    const initial_scale = transform.scale;
    const initial_center = { x: transform.view.x / initial_scale, y: transform.view.y / initial_scale };
    const started = performance.now();
    const advance = (time: number): void => {
      const progress = Math.min(1, (time - started) / 200);
      const eased = 1 - (1 - progress) ** 3;
      const scale = board.setScale(initial_scale + (target_scale - initial_scale) * eased);
      board.setView({
        x: (initial_center.x + (center.x - initial_center.x) * eased) * scale,
        y: (initial_center.y + (center.y - initial_center.y) * eased) * scale,
      });
      this.frame = progress < 1 ? requestAnimationFrame(advance) : 0;
    };
    this.frame = requestAnimationFrame(advance);
  }

  private frameEvent(points: Point2D[], viewport: RisqViewport): void {
    const center = {
      x: points.reduce((sum: number, point: Point2D): number => sum + point.x, 0) / points.length,
      y: points.reduce((sum: number, point: Point2D): number => sum + point.y, 0) / points.length,
    };
    const rotated = points.map((point: Point2D): Point2D => rotatePoint(point, viewport.lastTransform().rotation));
    const width =
      Math.max(...rotated.map((point: Point2D): number => point.x)) -
      Math.min(...rotated.map((point: Point2D): number => point.x)) +
      2 * viewport.hexR();
    const height =
      Math.max(...rotated.map((point: Point2D): number => point.y)) -
      Math.min(...rotated.map((point: Point2D): number => point.y)) +
      2 * viewport.hexR();
    const size = viewport.canvasSize();
    this.move(
      center,
      Math.min(viewport.zoneDetailScale() * 1.5, (size.width * 0.5) / width, (size.height * 0.5) / height)
    );
  }

  cancel(): void {
    if (this.frame !== 0) {
      cancelAnimationFrame(this.frame);
    }
    this.frame = 0;
  }
}
