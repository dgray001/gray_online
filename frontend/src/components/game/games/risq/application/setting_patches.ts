import { RisqResourceType } from '../model/types';
import type {
  AutoRenewSetData,
  BuildingBehaviorSetData,
  DefaultUnitBehaviorSetData,
  GatherPointSetData,
  UnitBehaviorSetData,
} from '../transport/snapshot_types';
import type { RisqSession } from './session';

export function patchUnitBehavior(session: RisqSession, data: UnitBehaviorSetData): void {
  for (const unit of data.internal_ids.map((id: number) => session.findUnitById(id))) {
    if (unit) {
      unit.stance = data.stance ?? unit.stance;
      unit.interrupt_current = data.interrupt_current ?? unit.interrupt_current;
      unit.attack_back = data.attack_back ?? unit.attack_back;
      unit.target_priority = data.target_priority ?? unit.target_priority;
    }
  }
}

export function patchBuildingBehavior(session: RisqSession, data: BuildingBehaviorSetData): void {
  for (const building of data.internal_ids.map((id: number) => session.findBuildingById(id))) {
    if (building) {
      building.auto_attack = data.auto_attack ?? building.auto_attack;
      building.interrupt_current = data.interrupt_current ?? building.interrupt_current;
      building.target_priority = data.target_priority ?? building.target_priority;
    }
  }
}

export function patchDefaultUnitBehavior(session: RisqSession, data: DefaultUnitBehaviorSetData): void {
  const player = session.getPlayer();
  if (player) {
    player.default_unit_stance = data.stance;
    player.default_unit_attack_back = data.attack_back;
    player.default_unit_interrupt_current = data.interrupt_current;
    player.default_unit_target_priority = data.target_priority;
  }
}

export function patchGatherPoint(session: RisqSession, data: GatherPointSetData): void {
  const building = session.findBuildingById(data.building_id);
  if (building) {
    building.gather_point = data.gather_point;
  }
}

export function patchAutoRenew(session: RisqSession, data: AutoRenewSetData): void {
  const player = session.getPlayer();
  if (!player) {
    return;
  }
  player.auto_renewals.set(data.building_id, data.count);
  const amounts: [RisqResourceType, number][] = [
    [RisqResourceType.FOOD, data.resources.food],
    [RisqResourceType.WOOD, data.resources.wood],
    [RisqResourceType.STONE, data.resources.stone],
    [RisqResourceType.GOLD, data.resources.gold],
  ];
  for (const [type, amount] of amounts) {
    const resource = player.resources.get(type);
    if (resource) {
      resource.amount = amount;
    }
  }
}
