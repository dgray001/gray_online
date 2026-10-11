export enum StartingStockpile {
  NONE,
  LOW,
  STANDARD,
  HIGH,
  DEATHMATCH,
}

export enum MapResourceLevel {
  NONE,
  SPARSE,
  STANDARD,
  RICH,
  ABUNDANT,
}

export const STARTING_STOCKPILE_LABELS: Record<StartingStockpile, string> = {
  [StartingStockpile.NONE]: 'Standard',
  [StartingStockpile.LOW]: 'Low',
  [StartingStockpile.STANDARD]: 'Standard',
  [StartingStockpile.HIGH]: 'High',
  [StartingStockpile.DEATHMATCH]: 'Deathmatch',
};
export const MAP_RESOURCE_LABELS: Record<MapResourceLevel, string> = {
  [MapResourceLevel.NONE]: 'Standard',
  [MapResourceLevel.SPARSE]: 'Sparse',
  [MapResourceLevel.STANDARD]: 'Standard',
  [MapResourceLevel.RICH]: 'Rich',
  [MapResourceLevel.ABUNDANT]: 'Abundant',
};
