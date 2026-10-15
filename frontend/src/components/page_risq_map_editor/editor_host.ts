import type { ColorRGB } from '../../scripts/color_rgb';
import { createImage, resolveImage } from '../../scripts/image';
import type { DwgCanvasBoard } from '../game/util/canvas_board/canvas_board';
import type { Point2D } from '../game/util/objects2d';
import { multiplyPoint2D } from '../game/util/objects2d';
import { RisqHover } from '../game/games/risq/application/input/hover';
import type { RisqSession } from '../game/games/risq/application/session';
import type { GameRisq, RisqPlayer } from '../game/games/risq/model/types';
import { RisqOrderType } from '../game/games/risq/model/types';
import { PLAYER_ICON_SIZE, RisqImageCache } from '../game/games/risq/rendering/assets/image_cache';
import type { RisqViewport } from '../game/games/risq/rendering/board/viewport';
import { RisqCorpseLayout } from '../game/games/risq/rendering/zones/corpses';
import type { RisqDrawHost } from '../game/games/risq/rendering/board/board_renderer';

export class EditorDrawHost implements RisqDrawHost {
  readonly hover: RisqHover;
  readonly corpse_layout = new RisqCorpseLayout();
  readonly planning = {
    getLocalFoundation: () => undefined,
    hasPlannedFoundation: () => false,
    idleIds: () => ({ units: new Set<number>(), buildings: new Set<number>() }),
  };
  readonly selection = {
    isUnitSelected: () => false,
    isBuildingSelected: () => false,
    isResourceSelected: () => false,
    isRegionSelected: () => false,
  };
  readonly armed = { getArmedOrder: () => RisqOrderType.NONE, getArmedBuilding: () => undefined };
  private icons = new Map<string, HTMLImageElement>();
  private image_cache = new RisqImageCache();

  constructor(
    readonly session: RisqSession,
    readonly viewport: RisqViewport
  ) {
    this.hover = new RisqHover(session, viewport);
  }

  getIcon(name: string): HTMLImageElement {
    let icon = this.icons.get(name);
    if (!icon) {
      icon = createImage(`/images/${name}.png`);
      icon.alt = name;
      this.icons.set(name, icon);
    }
    return resolveImage(icon);
  }

  getPlayerColoredIcon(name: string, color: ColorRGB): HTMLImageElement | HTMLCanvasElement {
    const icon = this.getIcon(name);
    return this.image_cache.getPlayerColoredIcon(name, icon, PLAYER_ICON_SIZE, color) ?? icon;
  }

  getImageCache(): RisqImageCache {
    return this.image_cache;
  }

  getGame(): GameRisq | undefined {
    return this.session.getGame();
  }

  getPlayer(): RisqPlayer | undefined {
    return undefined;
  }

  getPlayerId(): number {
    return -1;
  }
}

export function fitBoardView(
  session: RisqSession,
  viewport: RisqViewport,
  board: DwgCanvasBoard,
  board_size: Point2D,
  canvas_size: DOMRect,
  fitted_board_size: number
): number {
  const game = session.getGame();
  if (!game) {
    return fitted_board_size;
  }
  const { ratio, center } = viewport.resize(board_size, canvas_size, game.board_size);
  const zoom = viewport.zoomLimits(board_size);
  board.setMaxScale(zoom.max, zoom.min);
  board.scaleView(ratio);
  board.setOffset(center);
  for (const zone of game.spaces.flat().flatMap((space) => space?.zones?.flat() ?? [])) {
    zone.reset_hovered_data = true;
  }
  if (fitted_board_size !== game.board_size) {
    const scale = viewport.lastTransform().scale ?? 1;
    board.setView(multiplyPoint2D(scale, viewport.coordinateToCanvas({ x: 0, y: 0 })));
    return game.board_size;
  }
  return fitted_board_size;
}

import type { TerrainEntry } from './map_api';

export function patchTerrainSpaces(
  session: RisqSession,
  terrains: TerrainEntry[],
  coordinates: Point2D[],
  terrain_id: number
): void {
  const terrain = terrains.find((entry) => entry.terrain_id === terrain_id);
  for (const coordinate of coordinates) {
    const space = session.spaceAt(coordinate);
    if (space && terrain) {
      space.terrain_id = terrain_id;
      space.terrain_type = terrain.terrain_type;
      space.display_name = terrain.display_name;
    }
  }
}
