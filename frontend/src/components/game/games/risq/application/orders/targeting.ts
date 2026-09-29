import { equalsPoint2D } from '../../../../util/objects2d';
import type { RisqLeftPanel } from '../../canvas_components/left_panel/left_panel';
import { LeftPanelDataType } from '../../canvas_components/left_panel/left_panel_data';
import type { RisqBuilding, RisqZone } from '../../model/types';
import { RisqAttackType, RisqOrderType, RisqVisibilityLevel } from '../../model/types';
import type { RisqViewport } from '../../rendering/board/viewport';
import { hoveredZoneObject } from '../../rendering/zones/hit_testing';
import type { RisqArmedState } from '../input/armed_state';
import type { RisqHover } from '../input/hover';
import type { RisqSession } from '../session';
import type { RisqOrderPlanning } from './planning';

export declare interface BuildingAttackTarget {
  kind: 'unit' | 'building';
  internal_id: number;
}

/** Resolves what the hovered target means for the current selection and armed state */
export class RisqOrderTargeting {
  constructor(
    private session: RisqSession,
    private viewport: RisqViewport,
    private hover: RisqHover,
    private armed: RisqArmedState,
    private left_panel: RisqLeftPanel,
    private planning: RisqOrderPlanning
  ) {}

  /** The order a right-click would give right now, or NONE */
  resolveActiveOrderType(ctrl_held: boolean): RisqOrderType {
    const space = this.hover.space();
    if (!this.left_panel.getData() || !this.left_panel.isOrderable() || !this.session.canGiveOrders() || !space) {
      return RisqOrderType.NONE;
    }
    const armed_order = this.armed.getArmedOrder();
    if (armed_order !== RisqOrderType.NONE) {
      return this.resolveArmedOrderType(armed_order, ctrl_held);
    }
    if (!this.left_panel.isUnit()) {
      return RisqOrderType.NONE;
    }
    const zone = this.hover.zone();
    const has_villager = this.left_panel.hasVillager();
    const zone_valid = this.isZoneValid();
    const attack = zone_valid && this.left_panel.hasMilitary() ? this.hoveredAttackOrder() : undefined;
    if (attack) {
      return attack;
    }
    const building_slot_hovered = !!zone?.hovered_data[0]?.hovered;
    if (zone_valid && building_slot_hovered && this.garrisonTargetValid()) {
      return RisqOrderType.OrderType_UnitGarrison;
    }
    if (has_villager && building_slot_hovered && this.renewTargetValid()) {
      return RisqOrderType.OrderType_UnitRenew;
    }
    if (has_villager && building_slot_hovered && this.gatherTargetValid()) {
      return RisqOrderType.OrderType_UnitGather;
    }
    if (has_villager && building_slot_hovered && this.repairTargetValid()) {
      return RisqOrderType.OrderType_UnitRepair;
    }
    if (has_villager && zone_valid && building_slot_hovered) {
      if (zone!.building?.under_construction || this.planning.hasPlannedFoundation(zone)) {
        return RisqOrderType.OrderType_UnitBuild;
      }
    }
    if (this.atSelectedUnitPosition(zone_valid, ctrl_held)) {
      return RisqOrderType.NONE;
    }
    return zone_valid ? RisqOrderType.OrderType_UnitMoveZone : RisqOrderType.OrderType_UnitMoveSpace;
  }

  private resolveArmedOrderType(armed_order: RisqOrderType, ctrl_held: boolean): RisqOrderType {
    const is_unit = this.left_panel.isUnit();
    const has_villager = this.left_panel.hasVillager();
    const zone_valid = this.isZoneValid();
    switch (armed_order) {
      case RisqOrderType.OrderType_UnitMoveSpace:
      case RisqOrderType.OrderType_UnitMoveZone:
        if (is_unit && !this.atSelectedUnitPosition(zone_valid, ctrl_held)) {
          return zone_valid ? RisqOrderType.OrderType_UnitMoveZone : RisqOrderType.OrderType_UnitMoveSpace;
        }
        return RisqOrderType.NONE;
      case RisqOrderType.OrderType_UnitAttackSpace:
      case RisqOrderType.OrderType_UnitAttackZone:
        if (!is_unit) {
          return RisqOrderType.NONE;
        }
        return (
          this.hoveredAttackOrder() ??
          (zone_valid ? RisqOrderType.OrderType_UnitAttackZone : RisqOrderType.OrderType_UnitAttackSpace)
        );
      case RisqOrderType.OrderType_UnitGather:
        return has_villager && this.gatherTargetValid() ? RisqOrderType.OrderType_UnitGather : RisqOrderType.NONE;
      case RisqOrderType.OrderType_UnitBuild:
        return this.buildTargetValid() ? RisqOrderType.OrderType_UnitBuild : RisqOrderType.NONE;
      case RisqOrderType.OrderType_UnitRepair:
        return has_villager && this.repairTargetValid() ? RisqOrderType.OrderType_UnitRepair : RisqOrderType.NONE;
      case RisqOrderType.OrderType_UnitRenew:
        return has_villager && this.renewTargetValid() ? RisqOrderType.OrderType_UnitRenew : RisqOrderType.NONE;
      case RisqOrderType.OrderType_UnitGarrison:
        return zone_valid && this.garrisonTargetValid() ? RisqOrderType.OrderType_UnitGarrison : RisqOrderType.NONE;
      default:
        return RisqOrderType.NONE;
    }
  }

  /** Attack order implied by hovering an enemy building or unit slot in the hovered zone */
  private hoveredAttackOrder(): RisqOrderType | undefined {
    const zone = this.hover.zone();
    const hovered = zone && hoveredZoneObject(zone);
    const player_id = this.session.getPlayerId();
    if (hovered?.kind === 'building' && zone!.building!.player_id !== player_id) {
      return RisqOrderType.OrderType_UnitAttackBuilding;
    }
    if (hovered?.kind === 'unit' && hovered.groups.some((t) => t.player_id !== player_id)) {
      return RisqOrderType.OrderType_UnitAttackUnit;
    }
    return undefined;
  }

  private atSelectedUnitPosition(zone_valid: boolean, ctrl_held: boolean): boolean {
    const data = this.left_panel.getData();
    if (data?.data_type !== LeftPanelDataType.UNIT) {
      return false;
    }
    return this.isUnitAtHoverTarget(data.data.internal_id, zone_valid, ctrl_held, data.data);
  }

  /** Whether the unit already sits at the current hover target (never true while ctrl is held) */
  isUnitAtHoverTarget(
    internal_id: number,
    zone_valid: boolean,
    ctrl_held: boolean,
    unit = this.session.getPlayer()?.units.get(internal_id)
  ): boolean {
    const space = this.hover.space();
    if (ctrl_held || !space || !unit || !equalsPoint2D(space.coordinate, unit.space_coordinate)) {
      return false;
    }
    return zone_valid ? equalsPoint2D(this.hover.zone()?.coordinate, unit.zone_coordinate) : true;
  }

  isZoneValid(): boolean {
    const space = this.hover.space();
    return this.viewport.zoneView() && !!space && space.visibility >= RisqVisibilityLevel.FOG && !!this.hover.zone();
  }

  buildTargetValid(): boolean {
    const zone = this.hover.zone();
    const ownership = this.hover.space()?.ownership;
    return (
      this.left_panel.hasVillager() &&
      this.isZoneValid() &&
      !zone?.resource &&
      !zone?.building &&
      !this.planning.hasPlannedFoundation(zone) &&
      (ownership === undefined || ownership < 0 || ownership === this.session.getPlayerId())
    );
  }

  private ownCompletedBuilding(): RisqBuilding | undefined {
    const b = this.hover.zone()?.building;
    return b && b.player_id === this.session.getPlayerId() && !b.under_construction ? b : undefined;
  }

  private garrisonTargetValid(): boolean {
    const b = this.ownCompletedBuilding();
    return !!b && (b.garrisoned_units?.length ?? 0) < b.garrison_capacity;
  }

  private repairTargetValid(): boolean {
    const b = this.ownCompletedBuilding();
    return this.isZoneValid() && !!b && b.combat_stats.health < b.combat_stats.max_health;
  }

  private gatherTargetValid(): boolean {
    if (!this.isZoneValid()) {
      return false;
    }
    if (!!this.hover.zone()?.resource) {
      return true;
    }
    const b = this.ownCompletedBuilding();
    return !!b && b.gather_capacity !== undefined && (b.resources_left ?? 0) > 0;
  }

  private renewTargetValid(): boolean {
    const b = this.ownCompletedBuilding();
    return this.isZoneValid() && !!b && b.gather_capacity !== undefined && (b.resources_left ?? 0) <= 0;
  }

  /** internal_id of the enemy unit in whichever unit-slot is currently hovered, if any */
  hoveredEnemyUnitId(): number | undefined {
    const zone = this.hover.zone();
    for (const [i, slot] of (zone?.unit_slots ?? []).entries()) {
      if (!zone?.hovered_data[i + 1]?.hovered) {
        continue;
      }
      const enemy = slot.find((t) => t.player_id !== this.session.getPlayerId());
      return enemy ? [...enemy.units][0] : undefined;
    }
    return undefined;
  }

  buildingAttackTarget(building: RisqBuilding): BuildingAttackTarget | undefined {
    if (building.combat_stats.attack_type === RisqAttackType.NONE) {
      return undefined;
    }
    const player_id = this.session.getPlayerId();
    const zone = this.hover.zone();
    if (this.viewport.zoneView() && zone) {
      const hovered = hoveredZoneObject(zone);
      if (hovered?.kind === 'building' && zone.building && zone.building.player_id !== player_id) {
        return { kind: 'building', internal_id: zone.building.internal_id };
      }
      if (hovered?.kind === 'unit') {
        const enemy = hovered.groups.find((t) => t.player_id !== player_id);
        const unit_id = enemy ? [...enemy.units][0] : undefined;
        if (unit_id !== undefined) {
          return { kind: 'unit', internal_id: unit_id };
        }
      }
      if (!this.armed.isBuildingAttackArmed()) {
        return undefined;
      }
      if (zone.building && zone.building.player_id !== player_id) {
        return { kind: 'building', internal_id: zone.building.internal_id };
      }
      const enemy_unit = [...zone.units.values()].find((u) => u.player_id !== player_id);
      return enemy_unit ? { kind: 'unit', internal_id: enemy_unit.internal_id } : undefined;
    }
    const space = this.hover.space();
    if (this.armed.isBuildingAttackArmed() && space) {
      const enemy_building = [...(space.buildings?.values() ?? [])].find((b) => b.player_id !== player_id);
      if (enemy_building) {
        return { kind: 'building', internal_id: enemy_building.internal_id };
      }
      const enemy_unit = [...(space.units?.values() ?? [])].find((u) => u.player_id !== player_id);
      if (enemy_unit) {
        return { kind: 'unit', internal_id: enemy_unit.internal_id };
      }
    }
    return undefined;
  }

  mercenaryTargetZone(): RisqZone | undefined {
    return this.hover.zone() ?? this.hover.space()?.zones?.[1][1];
  }

  mercenaryPlacementInvalidReason(zone: RisqZone | undefined): string | undefined {
    const mercenary = this.armed.getArmedMercenary();
    const space = this.hover.space();
    const player_id = this.session.getPlayerId();
    if (!mercenary || !space || !zone) {
      return 'No valid location';
    }
    if (space.ownership !== player_id || zone.ownership !== player_id) {
      return 'Must hire in owned territory';
    }
    const region = this.session.getRegionForSpace(space.coordinate_key);
    if (region && region.owner !== player_id) {
      return 'Region not fully owned';
    }
    return this.planning.mercenaryUnavailableReason(mercenary);
  }
}
