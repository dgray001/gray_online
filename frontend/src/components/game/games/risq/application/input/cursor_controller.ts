import type { DwgCanvasBoard } from '../../../../util/canvas_board/canvas_board';
import { RisqOrderType, type RisqBuilding } from '../../model/types';
import {
  DEFAULT_CURSOR_IMAGE,
  armedCursorAlpha,
  cursorImageForOrderType,
  resolveBuildCursorUrl,
  resolveMercenaryCursorUrl,
} from '../../rendering/cursor';
import type { DwgRisq } from '../../risq';
import type { RisqOrderTargeting } from '../orders/targeting';
import type { RisqSelection } from '../selection/selection';
import type { RisqArmedState } from './armed_state';

/** Picks the board cursor from the armed interaction and what a click on the hovered target would do */
export class RisqCursorController {
  constructor(
    private risq: DwgRisq,
    private board: () => DwgCanvasBoard | undefined,
    private armed: RisqArmedState,
    private targeting: RisqOrderTargeting,
    private selection: RisqSelection
  ) {}

  setDefault(): void {
    this.board()?.setCursor(DEFAULT_CURSOR_IMAGE);
  }

  update(ctrl_held: boolean): void {
    const board = this.board();
    if (!board) {
      return;
    }
    if (this.armed.isGatherPointArmed()) {
      board.setCursor('gather_point');
      return;
    }
    const armed_building = this.armed.getArmedBuilding();
    if (this.armed.getArmedOrder() === RisqOrderType.OrderType_UnitBuild && armed_building) {
      const valid = this.targeting.buildTargetValid();
      const url = resolveBuildCursorUrl(this.risq, armed_building, valid);
      if (url) {
        board.setCursorUrl(url, armedCursorAlpha(valid));
        return;
      }
    }
    const mercenary = this.armed.getArmedMercenary();
    if (mercenary) {
      const valid = !this.targeting.mercenaryPlacementInvalidReason(this.targeting.mercenaryTargetZone());
      const url = resolveMercenaryCursorUrl(this.risq, mercenary.id, valid);
      if (url) {
        board.setCursorUrl(url, armedCursorAlpha(valid));
        return;
      }
    }
    if (this.armed.isBuildingAttackArmed()) {
      const valid = this.selection
        .selectedBuildings()
        .some((building: RisqBuilding): boolean => !!this.targeting.buildingAttackTarget(building));
      board.setCursor(cursorImageForOrderType(RisqOrderType.OrderType_BuildingAttackUnit), armedCursorAlpha(valid));
      return;
    }
    const active_order = this.targeting.resolveActiveOrderType(ctrl_held);
    const armed_order = this.armed.getArmedOrder();
    if (active_order === RisqOrderType.NONE && armed_order !== RisqOrderType.NONE) {
      board.setCursor(cursorImageForOrderType(armed_order), armedCursorAlpha(false));
      return;
    }
    board.setCursor(cursorImageForOrderType(active_order));
  }
}
