/** Returns image path of the building */
export function buildingImage(building_id: number | undefined, under_construction?: boolean): string {
  if (building_id !== undefined && under_construction) {
    return 'risq/buildings/construction';
  }
  let filename = 'empty_plot';
  if (building_id !== undefined) {
    switch (building_id) {
      case 1:
        filename = 'village_center';
        break;
      case 2:
        filename = 'housing';
        break;
      default:
        console.error('Trying to get building image from unknown building id', building_id);
        return '';
    }
  }
  return `risq/buildings/${filename}`;
}
