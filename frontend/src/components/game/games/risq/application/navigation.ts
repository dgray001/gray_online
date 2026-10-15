import type { Point2D } from '../../../util/objects2d';
import { addPoint2D, multiplyPoint2D } from '../../../util/objects2d';
import type { RisqBuilding, RisqUnit } from '../model/types';
import { zoneCenterOffset } from '../rendering/zones/geometry';
import type { DwgRisq } from '../risq';

/** Moves the camera and selects idle orderables */
export class RisqNavigation {
  constructor(private risq: DwgRisq) {}

  goToVillageCenter(player_id: number): void {
    if (player_id < 0) {
      return;
    }
    const player = this.risq.getGame()?.players[player_id];
    const buildings = [...(player?.buildings.values() ?? [])];
    const target = buildings.find((b) => b.building_id === 1) ?? buildings[0] ?? [...(player?.units.values() ?? [])][0];
    if (!target) {
      return;
    }
    const location = 'building_id' in target ? target : this.risq.session.unitLocation(target);
    if (location) {
      this.goToCoordinate(location.space_coordinate);
    }
  }

  goToCoordinate(coordinate: Point2D): void {
    const scale = this.risq.viewport.lastTransform().scale ?? 1;
    this.risq.board.setView(multiplyPoint2D(scale, this.risq.viewport.coordinateToCanvas(coordinate)));
  }

  selectNextIdleUnit(): void {
    if (!this.risq.session.givingOrders()) {
      return;
    }
    const entity = this.risq.planning.nextIdleOrderable();
    if (entity) {
      this.selectIdleOrderable(entity);
    }
  }

  selectIdleOrderable(entity: RisqUnit | RisqBuilding): void {
    if ('building_id' in entity) {
      this.risq.selection.selectBuilding(entity);
    } else {
      this.risq.selection.selectUnit(entity);
    }
    const location = 'building_id' in entity ? entity : this.risq.session.unitLocation(entity);
    if (!location) {
      return;
    }
    const view = addPoint2D(
      this.risq.viewport.coordinateToCanvas(location.space_coordinate),
      zoneCenterOffset(location.zone_coordinate, this.risq.viewport.hexR())
    );
    this.risq.board.setView(multiplyPoint2D(this.risq.viewport.lastTransform().scale, view));
  }
}
