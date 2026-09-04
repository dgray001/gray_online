export function techImage(tech_id: number): string {
  switch (tech_id) {
    case 1:
      return 'risq/techs/loom';
    default:
      console.error('Trying to get tech image from unknown tech id', tech_id);
      return '';
  }
}
