/** Returns image path of the building; pass plain to get the uncolored source (no player-color marker pixels) */
export function buildingImage(building_id: number | undefined, under_construction?: boolean, plain = false): string {
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
      case 21:
        filename = 'outpost';
        break;
      case 11: // blacksmith
      case 22: // barracks
      case 23: // redoubt
        filename = 'empty_plot'; // no building image yet
        break;
      default:
        console.error('Trying to get building image from unknown building id', building_id);
        return '';
    }
    if (plain) {
      filename += '_plain';
    }
  }
  return `risq/buildings/${filename}`;
}
