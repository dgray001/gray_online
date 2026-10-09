import type { ColorRGB } from '../../scripts/color_rgb';
import { createImage, resolveImage } from '../../scripts/image';
import { RisqHover } from '../game/games/risq/application/input/hover';
import type { RisqSession } from '../game/games/risq/application/session';
import type { GameRisq, RisqPlayer } from '../game/games/risq/model/types';
import { RisqOrderType } from '../game/games/risq/model/types';
import { PLAYER_ICON_SIZE, RisqImageCache } from '../game/games/risq/rendering/assets/image_cache';
import type { RisqViewport } from '../game/games/risq/rendering/board/viewport';
import { RisqCorpseLayout } from '../game/games/risq/rendering/zones/corpses';
import type { RisqDrawHost } from '../game/games/risq/rendering/draw_host';

export class EditorDrawHost implements RisqDrawHost {
  readonly hover: RisqHover;
  readonly corpse_layout = new RisqCorpseLayout();
  readonly planning = { getLocalFoundation: () => undefined, hasPlannedFoundation: () => false };
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
