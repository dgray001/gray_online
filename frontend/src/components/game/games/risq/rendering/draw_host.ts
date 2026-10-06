import type { ColorRGB } from '../../../../../scripts/color_rgb';
import type { RisqArmedState } from '../application/input/armed_state';
import type { RisqHover } from '../application/input/hover';
import type { RisqOrderPlanning } from '../application/orders/planning';
import type { RisqSelection } from '../application/selection/selection';
import type { RisqSession } from '../application/session';
import type { GameRisq, RisqPlayer } from '../model/types';
import type { RisqImageCache } from './assets/image_cache';
import type { RisqViewport } from './board/viewport';

export interface RisqDrawHost {
  readonly session: Pick<RisqSession, 'getRegionForSpace' | 'getRegionLookup' | 'getResourceConfigs'>;
  readonly planning: Pick<RisqOrderPlanning, 'getLocalFoundation' | 'hasPlannedFoundation'>;
  readonly selection: Pick<
    RisqSelection,
    'isUnitSelected' | 'isBuildingSelected' | 'isResourceSelected' | 'isRegionSelected'
  >;
  readonly armed: Pick<RisqArmedState, 'getArmedOrder' | 'getArmedBuilding'>;
  readonly hover: Pick<RisqHover, 'hoveredRegion' | 'space'>;
  readonly viewport: Pick<RisqViewport, 'regionLabelPosition'>;
  getIcon(name: string): HTMLImageElement;
  getPlayerColoredIcon(name: string, color: ColorRGB): HTMLImageElement | HTMLCanvasElement;
  getImageCache(): RisqImageCache;
  getGame(): GameRisq | undefined;
  getPlayer(): RisqPlayer | undefined;
  getPlayerId(): number;
}
