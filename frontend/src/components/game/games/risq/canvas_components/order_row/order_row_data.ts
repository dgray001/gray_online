import type { DwgRisq } from '../../risq';
import type { RisqFrontendOrder } from '../../risq_data';

export interface RisqOrderRowConfig {
  game: DwgRisq;
  w: number;
  /** Whether to draw the subject icon/count badge; true in the right panel, false in the left panel */
  show_subject: boolean;
  order: RisqFrontendOrder;
  /** Other same-producible BuildingCreate orders collapsed behind this one, not yet started */
  collapsed_orders?: RisqFrontendOrder[];
  onCancel: (order: RisqFrontendOrder) => void;
  onCancelAll?: (orders: RisqFrontendOrder[]) => void;
}
