export function techImage(tech_id: number): string {
  switch (tech_id) {
    case 1:
      return 'risq/techs/loom';
    default:
      return 'icons/research64';
  }
}
