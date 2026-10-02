import { RisqRange, RisqProducibleKind } from '../../model/types';
import type { RisqCost, RisqUnitType } from '../../model/types';
import type { DwgRisq } from '../../risq';
import { unitImage } from '../../rendering/assets/unit';
import { buildingImage, techImage } from '../../rendering/assets/buildings';
export interface RisqUnitStatsEntry {
  health: number;
  attack_blunt: number;
  attack_piercing: number;
  attack_range: RisqRange;
  defense_blunt: number;
  defense_piercing: number;
  penetration_blunt: number;
  penetration_piercing: number;
}

interface RisqTechBonusEntry {
  max_health: number;
  turn_stamina: number;
  attack_blunt: number;
  attack_piercing: number;
  attack_magic: number;
  defense_blunt: number;
  defense_piercing: number;
  defense_magic: number;
  penetration_blunt: number;
  penetration_piercing: number;
  penetration_magic: number;
}

interface RisqProducibleEntry {
  row: number;
  col: number;
  kind: RisqProducibleKind;
  id: number;
  cost: RisqCost;
  stamina_cost: number;
  display_name: string;
  description: string;
  required_tech_id: number;
  stats?: RisqUnitStatsEntry;
  bonus?: RisqTechBonusEntry;
  affects_unit_ids?: number[];
  affects_unit_types?: RisqUnitType[];
  unlocks_mercenaries?: { id: number; display_name: string }[];
}

export interface RisqBuildingTreeEntry {
  building_id: number;
  display_name: string;
  description: string;
  required_tech_id: number;
  produces: RisqProducibleEntry[];
  cost: RisqCost;
  stamina_cost: number;
  stats?: RisqUnitStatsEntry;
}

export interface TechTreeDialogData {
  risq: DwgRisq;
}

type TreeStatus = 'researched' | 'available' | 'locked';

export interface HeaderNode {
  kind: 'header';
  x: number;
  y: number;
  w: number;
  h: number;
  status: TreeStatus;
  building: RisqBuildingTreeEntry;
}

interface ProducibleNode {
  kind: 'producible';
  x: number;
  y: number;
  w: number;
  h: number;
  status: TreeStatus;
  entry: RisqProducibleEntry;
}

export type TreeNode = HeaderNode | ProducibleNode;

export const PADDING = 24;
export const COLUMN_W = 220;
export const COLUMN_GAP = 44;
export const HEADER_H = 74;
export const HEADER_TO_NODES_GAP = 24;
export const NODE_S = 56;
export const NODE_GAP = 10;
export const ICON_S_HEADER = 40;
export const ICON_PADDING_NODE = 9;

export function techStatus(
  required_tech_id: number,
  researched: boolean,
  researched_techs: Map<number, boolean> | undefined
): TreeStatus {
  if (researched) {
    return 'researched';
  }
  if (required_tech_id !== 0 && !researched_techs?.get(required_tech_id)) {
    return 'locked';
  }
  return 'available';
}

export function nodeIcon(entry: RisqProducibleEntry): string {
  switch (entry.kind) {
    case RisqProducibleKind.UNIT:
      return unitImage(entry.id, true);
    case RisqProducibleKind.BUILDING:
      return buildingImage(entry.id, false, true);
    case RisqProducibleKind.TECH:
      return techImage(entry.id);
    default:
      return '';
  }
}

export const STATUS_FILL: Record<TreeStatus, string> = {
  researched: 'rgba(80, 200, 80, 0.2)',
  available: 'rgba(255, 255, 255, 0.08)',
  locked: 'rgba(255, 255, 255, 0.03)',
};
export const STATUS_STROKE: Record<TreeStatus, string> = {
  researched: 'rgb(80, 200, 80)',
  available: 'rgb(200, 190, 175)',
  locked: 'rgb(90, 90, 90)',
};
export const STATUS_TEXT: Record<TreeStatus, string> = {
  researched: 'rgb(230, 230, 230)',
  available: 'rgb(230, 230, 230)',
  locked: 'rgb(110, 110, 110)',
};

export const RANGE_DISTANCE: Partial<Record<RisqRange, number>> = {
  [RisqRange.SPACE]: 0,
  [RisqRange.ADJACENT]: 1,
  [RisqRange.SECONDARY]: 2,
};
