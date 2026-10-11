export enum MapType {
  SCRIPT = 'script',
  CUSTOM = 'custom',
}

export enum StartingDevelopment {
  NONE,
  NOMAD,
  STANDARD,
  ESTABLISHED,
  ADVANCED,
}

export enum StartBalance {
  NONE,
  RANDOM,
  MIRRORED,
}

export const STARTING_DEVELOPMENT_LABELS: Record<StartingDevelopment, string> = {
  [StartingDevelopment.NONE]: 'Standard',
  [StartingDevelopment.NOMAD]: 'Nomad',
  [StartingDevelopment.STANDARD]: 'Standard',
  [StartingDevelopment.ESTABLISHED]: 'Established',
  [StartingDevelopment.ADVANCED]: 'Advanced',
};
export const START_BALANCE_LABELS: Record<StartBalance, string> = {
  [StartBalance.NONE]: 'Mirrored',
  [StartBalance.RANDOM]: 'Random',
  [StartBalance.MIRRORED]: 'Mirrored',
};
