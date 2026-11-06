import type { BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import { canvasToScreen } from '../../../../util/canvas_board/canvas_board';
import type { Point2D } from '../../../../util/objects2d';
import type { CanvasComponent } from '../../../../util/canvas_components/canvas_component';
import type { ReplayEvent } from '../../model/replay_event';
import type { RisqTickLocation } from '../../model/tick_actions';
import type { RisqViewport } from '../../rendering/board/viewport';
import { hexagonVertices, zoneVertices } from '../../rendering/zones/geometry';

export type EventRect = { x: number; y: number; w: number; h: number };

function centerFacingPoint(points: Point2D[], _canvas: { width: number; height: number }): Point2D {
  const center = {
    x: points.reduce((sum: number, point: Point2D): number => sum + point.x, 0) / points.length,
    y: points.reduce((sum: number, point: Point2D): number => sum + point.y, 0) / points.length,
  };
  const distance = Math.hypot(center.x, center.y);
  const direction = distance > 0.001 ? { x: center.x / distance, y: center.y / distance } : { x: 0, y: -1 };
  const boundary = Math.max(
    ...points.map((point: Point2D): number => (point.x - center.x) * direction.x + (point.y - center.y) * direction.y)
  );
  const clearance = 32 * (Math.abs(direction.x) + Math.abs(direction.y)) + 8;
  return { x: center.x + direction.x * (boundary + clearance), y: center.y + direction.y * (boundary + clearance) };
}
export function overlaps(a: EventRect, b: EventRect): boolean {
  return a.x < b.x + b.w && a.x + a.w > b.x && a.y < b.y + b.h && a.y + a.h > b.y;
}
export function eventPoint(
  event: ReplayEvent,
  viewport: RisqViewport,
  transform: BoardTransformData,
  scale: number,
  canvas: { width: number; height: number }
): Point2D {
  if (event.map_position) {
    const screen = canvasToScreen(viewport.coordinateToCanvas(event.map_position), transform);
    return { x: screen.x / scale, y: screen.y / scale };
  }
  const points = event.locations.flatMap((location: RisqTickLocation): Point2D[] => {
    const center = viewport.coordinateToCanvas(location.space);
    const vertices =
      event.kind === 'continent' ? hexagonVertices(viewport.hexR()) : zoneVertices(location.zone, viewport.hexR());
    return vertices.map((vertex: Point2D): Point2D => ({ x: center.x + vertex.x, y: center.y + vertex.y }));
  });
  const board_point = centerFacingPoint(points, canvas);
  const screen = canvasToScreen(board_point, transform);
  return { x: screen.x / scale, y: screen.y / scale };
}

export function panelPosition(
  point: Point2D,
  size: Point2D,
  canvas: { width: number; height: number },
  components: CanvasComponent[]
): Point2D | undefined {
  const obstacles = components
    .map((c: CanvasComponent): EventRect => ({ x: c.xi(), y: c.yi(), w: c.xf() - c.xi(), h: c.yf() - c.yi() }))
    .filter((r: EventRect): boolean => r.w > 0 && r.h > 0);
  const xs = [
    point.x + 24,
    point.x - size.x - 24,
    4,
    canvas.width - size.x - 4,
    ...obstacles.flatMap((r: EventRect): number[] => [r.x - size.x - 4, r.x + r.w + 4]),
  ];
  const ys = [
    point.y + 24,
    point.y - size.y - 24,
    4,
    canvas.height - size.y - 4,
    ...obstacles.flatMap((r: EventRect): number[] => [r.y - size.y - 4, r.y + r.h + 4]),
  ];
  const candidates = xs
    .flatMap((x: number): Point2D[] => ys.map((y: number): Point2D => ({ x, y })))
    .filter(
      (p: Point2D): boolean =>
        p.x >= 0 &&
        p.y >= 0 &&
        p.x + size.x <= canvas.width &&
        p.y + size.y <= canvas.height &&
        !obstacles.some((r: EventRect): boolean => overlaps({ ...p, w: size.x, h: size.y }, r))
    );
  return candidates.sort(
    (a: Point2D, b: Point2D): number =>
      Math.hypot(a.x - point.x, a.y - point.y) - Math.hypot(b.x - point.x, b.y - point.y)
  )[0];
}
