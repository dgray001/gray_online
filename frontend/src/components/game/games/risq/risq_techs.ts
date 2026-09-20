import { err } from '../../../../scripts/log';

export function techImage(tech_id: number): string {
  switch (tech_id) {
    case 1:
      return 'risq/techs/loom';
    default:
      err('Trying to get tech image from unknown tech id', tech_id);
      return '';
  }
}
