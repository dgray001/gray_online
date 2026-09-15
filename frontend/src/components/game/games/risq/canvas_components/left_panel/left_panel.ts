import type { BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import type { CanvasComponent } from '../../../../util/canvas_components/canvas_component';
import { configDraw } from '../../../../util/canvas_components/canvas_component';
import { drawHexagon, drawLine, drawRect, drawText } from '../../../../util/canvas_util';
import type { Point2D } from '../../../../util/objects2d';
import { equalsPoint2D } from '../../../../util/objects2d';
import type { DwgRisq } from '../../risq';
import { buildingImage } from '../../risq_buildings';
import type {
  RectHoverData,
  RisqBuilding,
  RisqCombatStats,
  RisqResource,
  RisqSpace,
  RisqUnit,
  RisqZone,
  UnitByTypeData,
} from '../../risq_data';
import {
  RisqAttackType,
  RisqOrderType,
  RisqProducibleKind,
  RisqResourceType,
  RisqUnitStance,
  RisqUnitType,
  RisqVisibilityLevel,
} from '../../risq_data';
import { coordinateToIndex } from '../../risq_coordinates';
import { createTooltipState, drawTooltip, shouldShowTooltip } from '../../../../util/canvas_components/tooltip';
import { resourceImage, resourceTypeImage } from '../../risq_resources';
import { borderStrokeStyle, drawHexImage } from '../../risq_space';
import {
  COMBO_UNIT_ICON_SIZE,
  UNIT_HEALTHBAR_COLOR_BACKGROUND,
  UNIT_HEALTHBAR_COLOR_HEALTH,
  comboUnitIconKey,
  drawComboUnitIcon,
  unitImage,
} from '../../risq_unit';
import {
  INNER_ZONE_MULTIPLIER,
  OUTER_ZONE_INDICES,
  getZoneFill,
  resolveHoveredZones,
  unhoverRisqZone,
  unitsByPlayerFiltered,
} from '../../risq_zone';
import { RisqViewMode, spaceOwnerColor, terrainImage } from '../../risq_terrain';
import { ColorRGB } from '../../../../../../scripts/color_rgb';
import { RisqOrdersList } from '../right_panel/orders_list';
import type { RisqActionButton } from './action_button/action_button';
import { RisqBuildButton } from './action_button/build_button';
import { RisqCreateButton } from './action_button/create_button';
import { RisqDeleteButton } from './action_button/delete_button';
import { RisqDeleteFoundationButton } from './action_button/delete_foundation_button';
import { RisqGarrisonButton } from './action_button/garrison_button';
import { RisqResearchButton } from './action_button/research_button';
import { RisqStopButton } from './action_button/stop_button';
import { RisqLeftPanelButton } from './left_panel_close';
import type {
  LeftPanelConfig,
  LeftPanelData,
  PlayerUnitsDrawData,
  UnitsDrawData,
  FoundationDrawData,
} from './left_panel_data';
import { HoverableObjectType, LeftPanelDataType } from './left_panel_data';
import { RisqOrderButton } from './action_button/order_button';
import { RisqStanceButton } from './action_button/stance_button';
import { RisqSpaceUnitsRowButton } from './space_units_row_button';

export class RisqLeftPanel implements CanvasComponent {
  // For use in the draw function
  private static PADDING = 5;
  private static ACTION_GRID_ROWS = 4;
  private static ACTION_GRID_COLS = 5;

  private close_button: RisqLeftPanelButton;
  private order_rows_list: RisqOrdersList;

  private risq: DwgRisq;
  private config: LeftPanelConfig;
  private size: Point2D = { x: 0, y: 0 };
  private showing = false;
  private hovering = false;
  private visibility?: number;
  private data?: LeftPanelData;
  private buttons: RisqActionButton[] = [];
  private hovered_zone?: RisqZone; // relevant when drawing space and zone
  private hovered_object?: RisqUnit | RisqBuilding | RisqResource;
  private hovered_object_type: HoverableObjectType = HoverableObjectType.NONE;
  private space_villager_row_button?: RisqSpaceUnitsRowButton;
  private space_military_row_button?: RisqSpaceUnitsRowButton;
  private healthbar_row: RectHoverData = { ps: { x: 0, y: 0 }, pe: { x: 0, y: 0 } };
  private healthbar_tooltip = createTooltipState();
  private eye_badge_hover: RectHoverData = { ps: { x: 0, y: 0 }, pe: { x: 0, y: 0 } };
  private eye_badge_tooltip = createTooltipState();

  constructor(risq: DwgRisq, config: LeftPanelConfig) {
    this.risq = risq;
    if (config.w < 1) {
      config.w = 120;
    }
    this.config = config;
    this.close_button = new RisqLeftPanelButton(risq);
    this.order_rows_list = new RisqOrdersList(
      risq,
      config.w,
      new ColorRGB(222, 184, 135).addColor(255, 0, 0, 0.2),
      false
    );
    this.resolveSize();
  }

  private pushUnitActionRow(unit_internal_ids: number[]) {
    this.buttons.push(
      new RisqOrderButton(
        {
          row: 0,
          col: 0,
          order_type: RisqOrderType.OrderType_UnitMoveSpace,
          image_path: 'icons/move128',
          description: 'Move',
        },
        this.risq,
        0
      ),
      new RisqOrderButton(
        {
          row: 0,
          col: 1,
          order_type: RisqOrderType.OrderType_UnitAttackSpace,
          image_path: 'icons/sword128',
          description: 'Attack',
        },
        this.risq,
        0
      ),
      new RisqGarrisonButton({ row: 0, col: 2, unit_internal_ids }, this.risq, 0),
      new RisqStopButton(
        { row: 0, col: 3, image_path: 'icons/hand_stop128', description: 'Stop', unit_internal_ids },
        this.risq,
        0
      ),
      new RisqDeleteButton(
        {
          row: 0,
          col: RisqLeftPanel.ACTION_GRID_COLS - 1,
          image_path: 'icons/skull128',
          description: 'Delete',
          unit_internal_ids,
        },
        this.risq,
        0
      )
    );
  }

  private pushVillagerActionRow() {
    this.buttons.push(
      new RisqOrderButton(
        {
          row: 1,
          col: 0,
          order_type: RisqOrderType.OrderType_UnitGather,
          image_path: 'icons/gather128',
          description: 'Gather',
        },
        this.risq,
        0
      ),
      new RisqOrderButton(
        {
          row: 1,
          col: 1,
          order_type: RisqOrderType.OrderType_UnitRepair,
          image_path: 'icons/repair128',
          description: 'Repair',
        },
        this.risq,
        0
      )
    );
  }

  private pushMilitaryActionButtons(unit_internal_ids: number[]) {
    this.buttons.push(
      new RisqStanceButton(
        {
          row: 1,
          col: 0,
          unit_internal_ids,
          stance: RisqUnitStance.AGGRESSIVE,
          image_path: 'icons/swords32',
          description: 'Aggressive',
        },
        this.risq,
        0
      ),
      new RisqStanceButton(
        {
          row: 1,
          col: 1,
          unit_internal_ids,
          stance: RisqUnitStance.DEFENSIVE,
          image_path: 'icons/shield32',
          description: 'Defensive',
        },
        this.risq,
        0
      ),
      new RisqStanceButton(
        {
          row: 1,
          col: 2,
          unit_internal_ids,
          stance: RisqUnitStance.STAND_GROUND,
          image_path: '',
          description: 'Stand Ground',
        },
        this.risq,
        0
      ),
      new RisqStanceButton(
        { row: 1, col: 3, unit_internal_ids, stance: RisqUnitStance.PASSIVE, image_path: '', description: 'Passive' },
        this.risq,
        0
      )
    );
  }

  private refreshActionButtons() {
    this.buttons = [];
    this.refreshOrderRows();
    if (!this.isOrderable()) {
      this.resolveSize();
      return;
    }
    const own_player_id = this.risq.getPlayer()?.player.player_id;
    switch (this.data?.data_type) {
      case LeftPanelDataType.UNIT:
        this.pushUnitActionRow([this.data.data.internal_id]);
        if (this.isOnlyVillagers()) {
          this.pushVillagerActionRow();
          for (const producible of this.data.data.builds) {
            this.buttons.push(new RisqBuildButton({ producible }, this.risq, 0));
          }
        }
        if (this.isOnlyMilitary()) {
          this.pushMilitaryActionButtons([this.data.data.internal_id]);
        }
        break;
      case LeftPanelDataType.UNITS_BY_TYPE:
      case LeftPanelDataType.ECONOMIC_UNITS:
      case LeftPanelDataType.MILITARY_UNITS:
        this.pushUnitActionRow(this.data.data.units.flatMap((u) => [...u.units]));
        if (this.isOnlyVillagers()) {
          this.pushVillagerActionRow();
        }
        if (this.isOnlyMilitary()) {
          this.pushMilitaryActionButtons(this.data.data.units.flatMap((u) => [...u.units]));
        }
        break;
      case LeftPanelDataType.BUILDING:
        if (this.data.data.under_construction) {
          break;
        }
        for (const producible of this.data.data.produces) {
          if (producible.kind === RisqProducibleKind.UNIT) {
            this.buttons.push(
              new RisqCreateButton(
                {
                  building_id: this.data.data.internal_id,
                  producible,
                },
                this.risq,
                0
              )
            );
          } else if (producible.kind === RisqProducibleKind.TECH) {
            if (this.risq.getPlayer()?.researched_techs.get(producible.id)) {
              continue;
            }
            this.buttons.push(
              new RisqResearchButton(
                {
                  building_id: this.data.data.internal_id,
                  producible,
                },
                this.risq,
                0
              )
            );
          }
        }
        break;
      case LeftPanelDataType.FOUNDATION:
        if (this.data.data.player_id === own_player_id && this.risq.givingOrders()) {
          this.buttons.push(
            new RisqDeleteFoundationButton(
              {
                row: 0,
                col: RisqLeftPanel.ACTION_GRID_COLS - 1,
                image_path: 'icons/skull128',
                description: 'Cancel Foundation',
                foundation_data: this.data.data,
              },
              this.risq,
              0
            )
          );
        }
        break;
      default:
        break;
    }
    this.resolveSize();
    for (const button of this.buttons) {
      button.dataRefreshed();
    }
  }

  // Whether the currently selected unit(s)/building's own order queue should be shown
  private showOrderRows(): boolean {
    if (!this.data) {
      return false;
    }
    const own_player_id = this.risq.getPlayer()?.player.player_id;
    switch (this.data.data_type) {
      case LeftPanelDataType.UNIT:
      case LeftPanelDataType.BUILDING:
        return this.data.data.player_id === own_player_id;
      case LeftPanelDataType.UNITS_BY_TYPE:
      case LeftPanelDataType.ECONOMIC_UNITS:
      case LeftPanelDataType.MILITARY_UNITS:
        return this.data.data.units.some((u) => u.player_id === own_player_id);
      default:
        return false;
    }
  }

  private refreshOrderRows() {
    // an empty list never matches a real internal_id, so an unowned/non-orderable selection just shows an empty list
    let subject_internal_ids: number[] = [];
    let subject_kind: 'unit' | 'building' | undefined;
    if (this.showOrderRows() && this.data) {
      switch (this.data.data_type) {
        case LeftPanelDataType.UNIT:
        case LeftPanelDataType.BUILDING:
          subject_internal_ids = [this.data.data.internal_id];
          subject_kind = this.data.data_type === LeftPanelDataType.UNIT ? 'unit' : 'building';
          break;
        case LeftPanelDataType.UNITS_BY_TYPE:
        case LeftPanelDataType.ECONOMIC_UNITS:
        case LeftPanelDataType.MILITARY_UNITS:
          subject_internal_ids = this.data.data.units.flatMap((u) => [...u.units]);
          subject_kind = 'unit';
          break;
        default:
          break;
      }
    }
    this.order_rows_list.setSubject(subject_internal_ids, subject_kind);
    this.order_rows_list.refresh();
    const player = this.risq.getPlayer();
    if (this.risq.givingOrders() && !!player && !player.orders_submitted) {
      this.order_rows_list.enable();
    } else {
      this.order_rows_list.disable();
    }
  }

  resolveSize() {
    let h = 4 * this.config.w;
    if (h > this.risq.canvasSize().height) {
      h = this.risq.canvasSize().height;
    }
    this.size = { x: h / 3, y: h };
    this.close_button.setPosition({
      x: this.size.x,
      y: this.yi() + 0.5 * this.close_button.h(),
    });
    const y0 = this.yi() + 0.5 * this.size.y + RisqLeftPanel.PADDING;
    const y1 = this.yi() + 0.75 * this.size.y;
    const s = Math.min(
      (this.w() - (RisqLeftPanel.ACTION_GRID_COLS + 1) * RisqLeftPanel.PADDING) / RisqLeftPanel.ACTION_GRID_COLS,
      (y1 - y0 - (RisqLeftPanel.ACTION_GRID_ROWS - 1) * RisqLeftPanel.PADDING) / RisqLeftPanel.ACTION_GRID_ROWS
    );
    const grid_w = RisqLeftPanel.ACTION_GRID_COLS * s + (RisqLeftPanel.ACTION_GRID_COLS - 1) * RisqLeftPanel.PADDING;
    const x0 = this.xi() + 0.5 * (this.w() - grid_w);
    for (const button of this.buttons) {
      button.setSize(s, s);
      button.setPosition({
        x: x0 + button.col * (s + RisqLeftPanel.PADDING),
        y: y0 + button.row * (s + RisqLeftPanel.PADDING),
      });
    }
    const orders_y0 = this.yi() + 0.75 * this.size.y + RisqLeftPanel.PADDING;
    this.order_rows_list.setAllSizes(
      Math.min(0.1 * this.w(), 16),
      { x: this.xi() + RisqLeftPanel.PADDING, y: orders_y0 },
      this.w() - 2 * RisqLeftPanel.PADDING,
      this.yi() + this.size.y - orders_y0 - RisqLeftPanel.PADDING
    );
  }

  isHovering(): boolean {
    return this.hovering;
  }

  setHovering(hovering: boolean): void {
    this.hovering = hovering;
  }

  isClicking(): boolean {
    return false;
  }

  setClicking(_clicking: boolean): void {}

  isShowing(): boolean {
    return this.showing;
  }

  getData(): LeftPanelData | undefined {
    return this.data;
  }

  dataRefreshed() {
    this.refreshActionButtons();
  }

  // Returns whether the current selection in the left panel is orderable
  isOrderable(): boolean {
    const player_id = this.risq.getPlayer()?.player.player_id ?? -1;
    if (
      !this.showing ||
      !this.data ||
      !this.visibility ||
      this.visibility < RisqVisibilityLevel.GOOD ||
      player_id < 0
    ) {
      return false;
    }
    switch (this.data.data_type) {
      case LeftPanelDataType.BUILDING:
      case LeftPanelDataType.UNIT:
      case LeftPanelDataType.FOUNDATION:
        return this.data.data.player_id === player_id;
      case LeftPanelDataType.UNITS_BY_TYPE:
      case LeftPanelDataType.ECONOMIC_UNITS:
      case LeftPanelDataType.MILITARY_UNITS:
        if (this.data.data.units.length < 1) {
          return false;
        }
        return this.data.data.units[0].player_id === player_id;
      default:
        return false;
    }
  }

  // Returns whether the current selection is a unit or units
  isUnit(): boolean {
    if (!this.showing || !this.data) {
      return false;
    }
    switch (this.data.data_type) {
      case LeftPanelDataType.MULTIPLE_PLAYERS_UNITS:
      case LeftPanelDataType.UNITS:
      case LeftPanelDataType.UNITS_BY_TYPE:
      case LeftPanelDataType.ECONOMIC_UNITS:
      case LeftPanelDataType.MILITARY_UNITS:
      case LeftPanelDataType.UNIT:
        return true;
      default:
        return false;
    }
  }

  hasVillager(): boolean {
    if (!this.isUnit() || !this.data) {
      return false;
    }
    switch (this.data.data_type) {
      case LeftPanelDataType.UNIT:
        return this.data.data.unit_id === 1;
      case LeftPanelDataType.MULTIPLE_PLAYERS_UNITS:
        return this.data.data.units_by_player.some(([, units]) => units.some((u) => u.unit_id === 1));
      case LeftPanelDataType.UNITS_BY_TYPE:
      case LeftPanelDataType.ECONOMIC_UNITS:
      case LeftPanelDataType.MILITARY_UNITS:
        return this.data.data.units.some((u) => u.unit_id === 1);
      default:
        return false;
    }
  }

  hasMilitary(): boolean {
    if (!this.isUnit() || !this.data) {
      return false;
    }
    switch (this.data.data_type) {
      case LeftPanelDataType.UNIT:
        return this.data.data.unit_id > 10;
      case LeftPanelDataType.MULTIPLE_PLAYERS_UNITS:
        return this.data.data.units_by_player.some(([, units]) => units.some((u) => u.unit_id > 10));
      case LeftPanelDataType.UNITS_BY_TYPE:
      case LeftPanelDataType.ECONOMIC_UNITS:
      case LeftPanelDataType.MILITARY_UNITS:
        return this.data.data.units.some((u) => u.unit_id > 10);
      default:
        return false;
    }
  }

  // Returns whether every currently selected unit is a villager
  isOnlyVillagers(): boolean {
    if (!this.isUnit() || !this.data) {
      return false;
    }
    switch (this.data.data_type) {
      case LeftPanelDataType.UNIT:
        return this.data.data.unit_id === 1;
      case LeftPanelDataType.UNITS_BY_TYPE:
      case LeftPanelDataType.ECONOMIC_UNITS:
      case LeftPanelDataType.MILITARY_UNITS:
        return this.data.data.units.every((u) => u.unit_id === 1);
      default:
        return false;
    }
  }

  // Returns whether every currently selected unit is military
  isOnlyMilitary(): boolean {
    if (!this.isUnit() || !this.data) {
      return false;
    }
    switch (this.data.data_type) {
      case LeftPanelDataType.UNIT:
        return this.data.data.unit_id > 10;
      case LeftPanelDataType.UNITS_BY_TYPE:
      case LeftPanelDataType.ECONOMIC_UNITS:
      case LeftPanelDataType.MILITARY_UNITS:
        return this.data.data.units.every((u) => u.unit_id > 10);
      default:
        return false;
    }
  }

  isOnlyGarrisoned(): boolean {
    if (!this.isUnit() || !this.data) {
      return false;
    }
    const all_garrisoned = (player_id: number, t: UnitByTypeData): boolean =>
      [...t.units].every((id) => this.resolveUnit(player_id, id)?.garrisoned_in !== undefined);
    switch (this.data.data_type) {
      case LeftPanelDataType.UNIT:
        return this.data.data.garrisoned_in !== undefined;
      case LeftPanelDataType.MULTIPLE_PLAYERS_UNITS:
        return this.data.data.units_by_player.every(([player_id, units]) =>
          units.every((t) => all_garrisoned(player_id, t))
        );
      case LeftPanelDataType.UNITS_BY_TYPE:
      case LeftPanelDataType.ECONOMIC_UNITS:
      case LeftPanelDataType.MILITARY_UNITS:
        return this.data.data.units.every((t) => all_garrisoned(t.player_id, t));
      default:
        return false;
    }
  }

  // Returns whether the current selection is a building
  isBuilding(): boolean {
    if (!this.showing || !this.data) {
      return false;
    }
    switch (this.data.data_type) {
      case LeftPanelDataType.BUILDING:
        return true;
      default:
        return false;
    }
  }

  close() {
    this.showing = false;
    this.visibility = undefined;
    this.data = undefined;
    if (!!this.hovered_zone) {
      unhoverRisqZone(this.hovered_zone);
      this.hovered_zone = undefined;
    }
    this.healthbar_row.hovered = false;
    this.buttons = [];
  }

  openPanel(open_data: LeftPanelData, visibility: number) {
    if (visibility < RisqVisibilityLevel.FOG) {
      return; // not explored
    }
    if (
      visibility < RisqVisibilityLevel.GOOD &&
      [
        LeftPanelDataType.MULTIPLE_PLAYERS_UNITS,
        LeftPanelDataType.UNITS,
        LeftPanelDataType.UNITS_BY_TYPE,
        LeftPanelDataType.ECONOMIC_UNITS,
        LeftPanelDataType.MILITARY_UNITS,
        LeftPanelDataType.UNIT,
      ].includes(open_data.data_type)
    ) {
      return; // units not individually identifiable yet
    }
    this.hovered_object = undefined;
    this.hovered_object_type = HoverableObjectType.NONE;
    this.visibility = visibility;
    this.data = open_data;
    this.showing = true;
    switch (open_data.data_type) {
      case LeftPanelDataType.UNITS_BY_TYPE:
      case LeftPanelDataType.ECONOMIC_UNITS:
      case LeftPanelDataType.MILITARY_UNITS:
        this.checkUnitsByTypeData(open_data.data);
        break;
      case LeftPanelDataType.UNITS:
        this.checkUnitsData(open_data.data);
        break;
      default:
        break;
    }
    this.refreshActionButtons();
    this.risq.recalculateHover();
  }

  private checkUnitsData(data: UnitsDrawData) {
    if (!data || !data.space || !data.space.units || !data.units_by_player || data.units_by_player.size < 1) {
      this.close();
      return;
    }
    const new_data: [number, UnitByTypeData[]][] = [];
    for (const [player_id, units_by_type] of data.units_by_player.entries()) {
      if (player_id < 0 || units_by_type.length < 1) {
        continue;
      }
      new_data.push([player_id, units_by_type.filter((u) => u.units.size > 0)]);
    }
    if (new_data.length < 1) {
      this.close();
      return;
    }
    if (new_data.length > 1) {
      this.data = {
        data_type: LeftPanelDataType.MULTIPLE_PLAYERS_UNITS,
        data: { space: data.space, units_by_player: new_data },
      };
      return;
    }
    this.checkUnitsByTypeData({ space: data.space, units: new_data[0][1] });
  }

  private resolveUnit(player_id: number, unit_id: number): RisqUnit | undefined {
    return this.risq.getGame()?.players[player_id]?.units.get(unit_id);
  }

  /** Check units of a single player and change datatype accordingly */
  private checkUnitsByTypeData(data: PlayerUnitsDrawData) {
    if (data.units.length === 1 && data.units[0].units.size === 1) {
      const unit = this.resolveUnit(data.units[0].player_id, [...data.units[0].units.values()][0]);
      if (!!unit) {
        this.data = { data_type: LeftPanelDataType.UNIT, data: unit }; // set data as internal id
      } else {
        this.close();
      }
      return;
    }
    const new_data = { space: data.space, units: data.units };
    const data_type = (() => {
      const has_economic_units = data.units.some((u) => u.unit_id < 11);
      const has_military_units = data.units.some((u) => u.unit_id > 10);
      if (has_economic_units && has_military_units) {
        return LeftPanelDataType.UNITS_BY_TYPE;
      } else if (has_economic_units) {
        return LeftPanelDataType.ECONOMIC_UNITS;
      } else if (has_military_units) {
        return LeftPanelDataType.MILITARY_UNITS;
      }
      return undefined;
    })();
    if (!data_type) {
      this.close();
      return;
    }
    this.data = { data_type, data: new_data };
  }

  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number) {
    if (!this.isShowing()) {
      return;
    }
    configDraw(
      ctx,
      transform,
      {
        fill_style: this.config.background,
        stroke_style: 'transparent',
        stroke_width: 0,
        fixed_position: true,
      },
      false,
      false,
      () => {
        drawRect(ctx, { x: this.xi(), y: this.yi() }, this.w(), this.h());
        ctx.save();
        ctx.beginPath();
        ctx.rect(this.xi(), this.yi(), this.w(), this.h());
        ctx.clip();
        switch (this.data?.data_type) {
          case LeftPanelDataType.RESOURCE:
            this.drawResource(ctx, this.data.data);
            break;
          case LeftPanelDataType.BUILDING:
            this.drawBuilding(ctx, this.data.data);
            break;
          case LeftPanelDataType.SPACE:
            this.drawSpace(ctx, this.data.data);
            break;
          case LeftPanelDataType.ZONE:
            this.drawZone(ctx, this.data.data);
            break;
          case LeftPanelDataType.MULTIPLE_PLAYERS_UNITS:
            this.drawUnitsGeneric(ctx, this.data.data.units_by_player);
            break;
          case LeftPanelDataType.UNITS_BY_TYPE:
          case LeftPanelDataType.ECONOMIC_UNITS:
          case LeftPanelDataType.MILITARY_UNITS:
            this.drawUnitsGeneric(ctx, [[this.data.data.units[0].player_id, this.data.data.units]]);
            break;
          case LeftPanelDataType.UNITS:
            console.error('LeftPanelDataType.UNITS should have been converted by checkUnitsData', this.data);
            break;
          case LeftPanelDataType.UNIT:
            this.drawUnit(ctx, this.data.data);
            break;
          case LeftPanelDataType.FOUNDATION:
            this.drawFoundation(ctx, this.data.data);
            break;
          default:
            console.error('Unknown data type for left panel', this.data);
            break;
        }
        ctx.restore();
      }
    );
    if (
      (this.data?.data_type === LeftPanelDataType.UNIT || this.data?.data_type === LeftPanelDataType.BUILDING) &&
      shouldShowTooltip(this.healthbar_tooltip, !!this.healthbar_row.hovered, !!this.healthbar_row.clicked, dt)
    ) {
      const cs = this.data.data.combat_stats;
      drawTooltip(
        this.healthbar_tooltip,
        ctx,
        transform,
        this.risq.canvasSize(),
        { x: this.healthbar_row.pe.x, y: this.healthbar_row.pe.y },
        `${cs.health.toFixed(1)} / ${cs.max_health}`
      );
    }
    if (
      (this.data?.data_type === LeftPanelDataType.SPACE || this.data?.data_type === LeftPanelDataType.ZONE) &&
      shouldShowTooltip(this.eye_badge_tooltip, !!this.eye_badge_hover.hovered, !!this.eye_badge_hover.clicked, dt)
    ) {
      const space = this.data.data_type === LeftPanelDataType.SPACE ? this.data.data : this.data.data.space;
      drawTooltip(
        this.eye_badge_tooltip,
        ctx,
        transform,
        this.risq.canvasSize(),
        { x: this.eye_badge_hover.pe.x, y: this.eye_badge_hover.pe.y },
        space.visibility === RisqVisibilityLevel.POOR
          ? 'Poor visibility on this space; overall unit count seen but no specific units or zone-level unit information'
          : ''
      );
    }
    ctx.beginPath();
    for (const button of this.buttons) {
      button.draw(ctx, transform, dt);
      button.drawTooltip(ctx, transform, this.risq, dt);
    }
    if (this.data?.data_type === LeftPanelDataType.SPACE && (this.visibility ?? 0) > RisqVisibilityLevel.POOR) {
      this.space_villager_row_button?.draw(ctx, transform, dt);
      this.space_military_row_button?.draw(ctx, transform, dt);
    }
    if (this.showOrderRows()) {
      this.order_rows_list.draw(ctx, transform, dt);
    }
    ctx.beginPath();
    this.close_button.draw(ctx, transform, dt);
  }

  private drawUnitImage(ctx: CanvasRenderingContext2D, unit: RisqUnit, p: Point2D, s: number) {
    const color = this.risq.getGame()?.players[unit.player_id]?.color;
    const icon = color
      ? this.risq.getPlayerColoredIcon(unitImage(unit.unit_id), color)
      : this.risq.getIcon(unitImage(unit.unit_id));
    ctx.drawImage(icon, p.x, p.y, s, s);
    ctx.strokeStyle = UNIT_HEALTHBAR_COLOR_BACKGROUND;
    ctx.lineWidth = 0.4;
    ctx.fillStyle = UNIT_HEALTHBAR_COLOR_BACKGROUND;
    drawRect(ctx, { x: p.x, y: p.y + 0.8 * s }, s, 0.18 * s);
    if (unit.combat_stats.max_health > 0 && unit.combat_stats.health > 0) {
      ctx.fillStyle = UNIT_HEALTHBAR_COLOR_HEALTH;
      drawRect(
        ctx,
        { x: p.x, y: p.y + 0.8 * s },
        (unit.combat_stats.health / unit.combat_stats.max_health) * s,
        0.2 * s
      );
    }
    if (unit.hover_data.hovered) {
      if (unit.hover_data.clicked) {
        ctx.fillStyle = 'rgba(250, 250, 250, 0.4)';
      } else {
        ctx.fillStyle = 'rgba(220, 220, 220, 0.2)';
      }
      ctx.strokeStyle = 'transparent';
      drawRect(ctx, p, s, s);
    }
    unit.hover_data.ps = p;
    unit.hover_data.pe = { x: p.x + s, y: p.y + s };
  }

  private drawUnitCountBadge(ctx: CanvasRenderingContext2D, count: number, p: Point2D, s: number) {
    const text = `×${count}`;
    const font_size = Math.max(10, 0.22 * s);
    ctx.font = `bold ${font_size}px serif`;
    const badge_w = ctx.measureText(text).width + 6;
    const badge_h = font_size + 4;
    const badge_p = { x: p.x + s - badge_w, y: p.y + s - badge_h };
    ctx.fillStyle = 'rgba(0, 0, 0, 0.75)';
    ctx.strokeStyle = 'transparent';
    drawRect(ctx, badge_p, badge_w, badge_h, 3);
    drawText(ctx, text, {
      p: { x: badge_p.x + 0.5 * badge_w, y: badge_p.y + 0.5 * badge_h },
      w: badge_w,
      fill_style: 'white',
      align: 'center',
      baseline: 'middle',
      font: `bold ${font_size}px serif`,
    });
  }

  private drawUnitCountBlock(
    ctx: CanvasRenderingContext2D,
    icon: CanvasImageSource,
    count: number,
    p: Point2D,
    s: number
  ) {
    ctx.drawImage(icon, p.x, p.y, s, s);
    this.drawUnitCountBadge(ctx, count, p, s);
  }

  private chunkBlocks<T>(items: T[], size: number): T[][] {
    const rows: T[][] = [];
    for (let i = 0; i < items.length; i += size) {
      rows.push(items.slice(i, i + size));
    }
    return rows;
  }

  private drawUnitSeparators(ctx: CanvasRenderingContext2D, single_owner_player_id?: number) {
    if (single_owner_player_id !== undefined && this.risq.getPlayer()?.player.player_id === single_owner_player_id) {
      this.drawSeparator(ctx, this.yi() + 0.5 * this.size.y);
      this.drawSeparator(ctx, this.yi() + 0.75 * this.size.y);
    }
  }

  // Selection can hold units from one or several players; renders a 4-blocks-per-row grid that
  // collapses through individual units -> unit_id -> unit_type -> one block per player as it overflows.
  private drawUnitsGeneric(ctx: CanvasRenderingContext2D, groups: [number, UnitByTypeData[]][]) {
    const multi_player = groups.length > 1;
    const single_owner_player_id = multi_player ? undefined : groups[0]?.[0];
    const total_units = groups.reduce((sum, [, units]) => sum + units.reduce((s, u) => s + u.units.size, 0), 0);
    const content_yi = this.yi() + this.drawName(ctx, `${total_units} Unit${total_units === 1 ? '' : 's'}`);
    const gap = 8;
    const per_row = 4;
    const content_x = this.xi() + 0.1 * this.w();
    const content_w = 0.8 * this.w();
    const block_size = (content_w - (per_row - 1) * gap) / per_row;
    const grid_bottom = this.yi() + 0.5 * this.size.y;
    const max_rows = Math.floor((grid_bottom - gap - content_yi) / (block_size + gap));

    const layout_rows = <T>(row_groups: T[][], draw_item: (item: T, p: Point2D) => void) => {
      let by = grid_bottom - row_groups.length * (block_size + gap);
      for (const row of row_groups) {
        let bx = content_x;
        for (const item of row) {
          draw_item(item, { x: bx, y: by });
          bx += block_size + gap;
        }
        by += block_size + gap;
      }
    };

    interface UnitRef {
      player_id: number;
      internal_id: number;
    }
    const per_player_refs: [number, UnitRef[]][] = groups
      .map(([player_id, units]): [number, UnitRef[]] => [
        player_id,
        units.flatMap((u) => [...u.units].map((internal_id) => ({ player_id, internal_id }))),
      ])
      .filter(([, refs]) => refs.length > 0);
    const draw_unit_ref = (ref: UnitRef, p: Point2D) => {
      const unit = this.resolveUnit(ref.player_id, ref.internal_id);
      if (unit) {
        this.drawUnitImage(ctx, unit, p, block_size);
      }
    };

    // Tier 1: individual units, one row-run per player
    const tier1_row_groups = per_player_refs.map(([, refs]) => this.chunkBlocks(refs, per_row));
    const tier1_rows = tier1_row_groups.reduce((sum, rows) => sum + rows.length, 0);
    if (tier1_rows > 0 && tier1_rows <= max_rows) {
      let by = grid_bottom - tier1_rows * (block_size + gap);
      for (const [i, [player_id]] of per_player_refs.entries()) {
        const rows = tier1_row_groups[i];
        if (multi_player) {
          const color = this.risq.getGame()?.players[player_id]?.color;
          ctx.fillStyle = color
            ? `rgba(${color.getR()}, ${color.getG()}, ${color.getB()}, 0.12)`
            : 'rgba(255, 255, 255, 0.06)';
          ctx.strokeStyle = 'transparent';
          drawRect(
            ctx,
            { x: content_x - 0.5 * gap, y: by - 0.5 * gap },
            content_w + gap,
            rows.length * (block_size + gap)
          );
        }
        for (const row of rows) {
          let bx = content_x;
          for (const ref of row) {
            draw_unit_ref(ref, { x: bx, y: by });
            bx += block_size + gap;
          }
          by += block_size + gap;
        }
      }
      this.drawUnitSeparators(ctx, single_owner_player_id);
      return;
    }

    // Tier 2: individual units, player-row-boundary dropped
    const all_refs = per_player_refs.flatMap(([, refs]) => refs);
    const tier2_row_groups = this.chunkBlocks(all_refs, per_row);
    if (tier2_row_groups.length > 0 && tier2_row_groups.length <= max_rows) {
      layout_rows(tier2_row_groups, draw_unit_ref);
      this.drawUnitSeparators(ctx, single_owner_player_id);
      return;
    }

    // Tier 3: one block per unit_id
    interface IdBlock {
      player_id: number;
      unit_id: number;
      count: number;
    }
    const id_blocks: IdBlock[] = groups.flatMap(([player_id, units]) =>
      units.filter((u) => u.units.size > 0).map((u) => ({ player_id, unit_id: u.unit_id, count: u.units.size }))
    );
    const tier3_row_groups = this.chunkBlocks(id_blocks, per_row);
    if (tier3_row_groups.length > 0 && tier3_row_groups.length <= max_rows) {
      layout_rows(tier3_row_groups, (block, p) => {
        const color = this.risq.getGame()?.players[block.player_id]?.color;
        const icon = color
          ? this.risq.getPlayerColoredIcon(unitImage(block.unit_id), color)
          : this.risq.getIcon(unitImage(block.unit_id));
        this.drawUnitCountBlock(ctx, icon, block.count, p, block_size);
      });
      return;
    }

    // Tier 4: one block per unit_type (merges unit_ids sharing a type)
    interface TypeBlock {
      player_id: number;
      unit_type: RisqUnitType;
      representative_unit_id: number;
      count: number;
    }
    const type_blocks_by_key = new Map<string, TypeBlock>();
    for (const [player_id, units] of groups) {
      for (const u of units) {
        if (u.units.size < 1) {
          continue;
        }
        const sample = this.resolveUnit(player_id, [...u.units][0]);
        const unit_type = sample?.unit_type ?? RisqUnitType.NONE;
        const key = `${player_id}:${unit_type}`;
        const existing = type_blocks_by_key.get(key);
        if (existing) {
          existing.count += u.units.size;
        } else {
          type_blocks_by_key.set(key, { player_id, unit_type, representative_unit_id: u.unit_id, count: u.units.size });
        }
      }
    }
    const type_blocks = [...type_blocks_by_key.values()];
    const tier4_row_groups = this.chunkBlocks(type_blocks, per_row);
    if (type_blocks.length > 0 && (tier4_row_groups.length <= max_rows || !multi_player)) {
      layout_rows(tier4_row_groups, (block, p) => {
        const color = this.risq.getGame()?.players[block.player_id]?.color;
        const icon = color
          ? this.risq.getPlayerColoredIcon(unitImage(block.representative_unit_id), color)
          : this.risq.getIcon(unitImage(block.representative_unit_id));
        this.drawUnitCountBlock(ctx, icon, block.count, p, block_size);
      });
      return;
    }

    // Tier 5: one block per player (only reachable for a multi-player selection)
    const player_blocks = groups
      .map(([player_id, units]) => ({ player_id, count: units.reduce((s, u) => s + u.units.size, 0) }))
      .filter((b) => b.count > 0);
    const tier5_row_groups = this.chunkBlocks(player_blocks, per_row);
    layout_rows(tier5_row_groups, (block, p) => {
      const color = this.risq.getGame()?.players[block.player_id]?.color;
      ctx.fillStyle = color ? color.getString() : 'rgba(255, 255, 255, 0.3)';
      ctx.strokeStyle = 'black';
      ctx.lineWidth = 1;
      drawRect(ctx, p, block_size, block_size);
      this.drawUnitCountBadge(ctx, block.count, p, block_size);
    });
  }

  private drawUnit(ctx: CanvasRenderingContext2D, unit: RisqUnit) {
    let yi = this.yi() + this.drawName(ctx, unit.display_name);
    yi += this.drawImage(ctx, yi, unitImage(unit.unit_id), this.risq.getGame()?.players[unit.player_id]?.color);
    this.drawSeparator(ctx, yi);
    yi = this.yi() + 0.25 * this.size.y + 6;
    this.drawCombatStats(ctx, yi, yi + 0.25 * this.size.y - 12, unit.combat_stats, unit.current_stamina);
    if (this.risq.getPlayer()?.player.player_id === unit.player_id) {
      this.drawSeparator(ctx, this.yi() + 0.5 * this.size.y);
      this.drawSeparator(ctx, this.yi() + 0.75 * this.size.y);
    }
  }

  private drawResource(ctx: CanvasRenderingContext2D, resource: RisqResource) {
    let yi = this.yi() + this.drawName(ctx, resource.display_name);
    yi += this.drawImage(ctx, yi, resourceImage(resource));
    this.drawSeparator(ctx, yi);
    yi += 8;
    ctx.beginPath();
    ctx.drawImage(this.risq.getIcon(resourceTypeImage(resource)), this.xi() + 0.1 * this.w(), yi, 40, 40);
    const resources_left = resource.resources_left.toFixed(1);
    drawText(ctx, resources_left, {
      p: { x: this.xi() + 0.1 * this.w() + 48, y: yi + 20 },
      w: 0.9 * this.w() - 48,
      fill_style: 'black',
      baseline: 'middle',
      font: '36px serif',
    });
    yi += 50;
    drawText(ctx, `Base gather speed: ${resource.base_gather_speed}`, {
      p: { x: this.xi() + 0.1 * this.w(), y: yi + 12 },
      w: 0.9 * this.w(),
      fill_style: 'black',
      baseline: 'middle',
      font: '18px serif',
    });
  }

  private drawBuilding(ctx: CanvasRenderingContext2D, building: RisqBuilding) {
    let yi = this.yi() + this.drawName(ctx, building?.display_name ?? 'Empty Plot');
    yi += this.drawImage(
      ctx,
      yi,
      buildingImage(building?.building_id, building?.under_construction),
      building ? this.risq.getGame()?.players[building.player_id]?.color : undefined
    );
    this.drawSeparator(ctx, yi);
    if (!building) {
      yi += 12;
      drawText(ctx, 'An empty lot that can be built on', {
        p: { x: this.xi() + 0.1 * this.w(), y: yi },
        w: 0.9 * this.w(),
        fill_style: 'black',
        align: 'left',
        font: `14px serif`,
      });
      return;
    }
    yi = this.yi() + 0.25 * this.size.y + 6;
    this.drawCombatStats(ctx, yi, yi + 0.25 * this.size.y - 12, building.combat_stats, building.current_stamina);
    if (this.risq.getPlayer()?.player.player_id === building.player_id) {
      this.drawSeparator(ctx, this.yi() + 0.5 * this.size.y);
      this.drawSeparator(ctx, this.yi() + 0.75 * this.size.y);
    }
  }

  private drawFoundation(ctx: CanvasRenderingContext2D, foundation: FoundationDrawData) {
    let yi = this.yi() + this.drawName(ctx, foundation.display_name);
    yi += this.drawImage(
      ctx,
      yi,
      'risq/buildings/construction',
      this.risq.getGame()?.players[foundation.player_id]?.color
    );
    this.drawSeparator(ctx, yi);
    yi += 12;
    drawText(ctx, 'A planned foundation awaiting construction units', {
      p: { x: this.xi() + 0.1 * this.w(), y: yi },
      w: 0.9 * this.w(),
      fill_style: 'black',
      align: 'left',
      font: `14px serif`,
    });
  }

  private drawSpace(ctx: CanvasRenderingContext2D, space: RisqSpace) {
    let yi = this.yi() + this.drawName(ctx, space.display_name);
    drawText(ctx, 'space', {
      p: { x: this.xc(), y: yi },
      w: this.w(),
      fill_style: 'black',
      align: 'center',
      font: '18px serif',
    });
    yi += 26;
    const separator_distance = 8;
    yi += this.drawSpaceHexagon(ctx, space, separator_distance, yi);
    this.drawSeparator(ctx, yi);
    yi += separator_distance;
    if ((this.visibility ?? 0) >= RisqVisibilityLevel.POOR) {
      const rows = 6;
      const image_size = Math.min(
        36,
        (1 / rows) * (0.6 * this.h() - separator_distance - (rows - 1) * separator_distance)
      );
      ctx.fillStyle = 'black';
      const draw_row_text = (x: number, text: string, max_w: number) => {
        const row_yc = yi + 0.5 * image_size;
        const font = `bold ${image_size}px serif`;
        ctx.font = font;
        const colon_w = ctx.measureText(': ').width;
        drawText(ctx, ': ', {
          p: { x, y: row_yc },
          w: colon_w,
          fill_style: 'black',
          align: 'left',
          baseline: 'middle',
          font,
        });
        drawText(ctx, text, {
          p: { x: x + colon_w, y: row_yc },
          w: max_w - colon_w,
          fill_style: 'black',
          align: 'left',
          baseline: 'middle',
          font,
        });
      };
      const draw_row = (img: CanvasImageSource, text: string, hover_data?: RectHoverData) => {
        const ps = { x: this.xi() + 0.1 * this.w(), y: yi };
        const pe = { x: ps.x + 0.8 * this.w(), y: ps.y + image_size };
        if (hover_data?.hovered) {
          ctx.strokeStyle = 'transparent';
          ctx.fillStyle = hover_data.clicked ? 'rgba(250, 250, 250, 0.4)' : 'rgba(210, 210, 210, 0.25)';
          drawRect(ctx, ps, pe.x - ps.x, pe.y - ps.y);
          ctx.fillStyle = 'black';
        }
        ctx.drawImage(img, ps.x, yi, image_size, image_size);
        draw_row_text(ps.x + image_size + 2, text, 0.8 * this.w() - image_size - 2);
        if (!!hover_data) {
          hover_data.ps = ps;
          hover_data.pe = pe;
        }
        yi += image_size + separator_distance;
      };
      draw_row(this.risq.getIcon('icons/building64'), space.buildings?.size.toString() ?? '??');
      draw_row(
        this.risq.getIcon(resourceTypeImage(RisqResourceType.GOLD)),
        `${(space.gold_income ?? 0).toFixed(1)}/turn`
      );
      const owner_color = spaceOwnerColor(space, this.risq.getGame()?.players ?? []);
      const owner_name = owner_color
        ? (this.risq.getGame()?.players[space.ownership ?? -1]?.player.nickname ?? 'Unknown')
        : '--Unclaimed--';
      const owner_ps = { x: this.xi() + 0.1 * this.w(), y: yi };
      ctx.fillStyle = owner_color ? owner_color.getString() : 'rgba(255, 255, 255, 0.3)';
      ctx.strokeStyle = 'black';
      ctx.lineWidth = 1;
      drawRect(ctx, owner_ps, image_size, image_size);
      draw_row_text(owner_ps.x + image_size + 2, owner_name, 0.8 * this.w() - image_size - 2);
      yi += image_size + separator_distance;
      if (this.visibility === RisqVisibilityLevel.POOR) {
        const villager_img = this.risq.getIcon('icons/villager64');
        const unit_img = this.risq.getIcon('icons/unit64');
        const combo_icon = this.risq
          .getImageCache()
          .getImage(comboUnitIconKey(false), COMBO_UNIT_ICON_SIZE, [villager_img, unit_img], (combo_ctx) =>
            drawComboUnitIcon(combo_ctx, villager_img, unit_img)
          );
        if (combo_icon) {
          draw_row(combo_icon, space.unit_count?.toString() ?? '??');
        }
      } else {
        yi += this.layoutSpaceUnitsRowButton(
          'villager',
          this.risq.getIcon('icons/villager64'),
          space.num_villager_units?.toString() ?? '??',
          true,
          yi,
          image_size,
          separator_distance
        );
        yi += this.layoutSpaceUnitsRowButton(
          'military',
          this.risq.getIcon('icons/unit64'),
          space.num_military_units?.toString() ?? '??',
          false,
          yi,
          image_size,
          separator_distance
        );
      }
      const resources = [...(space.total_resources?.entries() ?? [])]
        .filter((r) => r[1] > 0)
        .map((r) => r[0])
        .sort((a, b) => a - b);
      const num_resources = resources.length + 0.3 * (resources.length - 1); // account for slashes
      const resource_image_size = Math.min(image_size, (1.0 / num_resources) * 0.8 * this.w());
      for (const [i, r] of resources.entries()) {
        if (i > 0) {
          drawText(ctx, '/', {
            p: {
              x: this.xi() + 0.1 * this.w() + (i * 1.3 - 0.15) * resource_image_size,
              y: yi,
            },
            w: 0.3 * resource_image_size,
            fill_style: 'black',
            align: 'center',
            font: `bold ${resource_image_size}px serif`,
          });
        }
        ctx.drawImage(
          this.risq.getIcon(resourceTypeImage(r)),
          this.xi() + 0.1 * this.w() + i * 1.3 * resource_image_size,
          yi,
          resource_image_size,
          resource_image_size
        );
      }
      yi += image_size + separator_distance;
    }
  }

  openSpaceUnitsRow(economic: boolean): void {
    if (this.data?.data_type !== LeftPanelDataType.SPACE) {
      return;
    }
    const space = this.data.data;
    this.openPanel(
      {
        data_type: LeftPanelDataType.UNITS,
        data: { space, units_by_player: unitsByPlayerFiltered(space.units ?? new Map(), economic) },
      },
      this.visibility ?? 0
    );
  }

  /** Positions the space panel's villager/military row button, returning the row's height */
  private layoutSpaceUnitsRowButton(
    kind: 'villager' | 'military',
    icon: HTMLImageElement,
    count: string,
    economic: boolean,
    yi: number,
    image_size: number,
    separator_distance: number
  ): number {
    const p = { x: this.xi() + 0.1 * this.w(), y: yi };
    const w = 0.8 * this.w();
    let button = kind === 'villager' ? this.space_villager_row_button : this.space_military_row_button;
    if (!button) {
      button = new RisqSpaceUnitsRowButton({
        p,
        w,
        h: image_size,
        icon,
        panel: this,
        economic,
      });
      if (kind === 'villager') {
        this.space_villager_row_button = button;
      } else {
        this.space_military_row_button = button;
      }
    }
    button.setPosition(p);
    button.setSize(w, image_size);
    button.setRowText(`: ${count}`);
    return image_size + separator_distance;
  }

  private hexagon_r: number = 0;
  private hexagon_c: Point2D = { x: 0, y: 0 };
  private drawSpaceHexagon(
    ctx: CanvasRenderingContext2D,
    space: RisqSpace,
    separator_distance: number,
    yi: number,
    curr_zone: Point2D = { x: -1, y: -1 }
  ): number {
    const hexagon_height = Math.min(this.w(), this.yi() + 0.4 * this.h() - yi - separator_distance);
    const owner_color = spaceOwnerColor(space, this.risq.getGame()?.players ?? []);
    ctx.strokeStyle = borderStrokeStyle(owner_color, 1);
    ctx.lineWidth = 2;
    const r = 0.5 * hexagon_height;
    this.hexagon_r = r;
    const inner_r = INNER_ZONE_MULTIPLIER * r;
    const c = { x: this.xc(), y: yi + r };
    this.hexagon_c = c;
    drawHexImage(ctx, this.risq.getIcon(terrainImage(space.terrain_id)), c, r);
    ctx.fillStyle = 'transparent';
    drawHexagon(ctx, c, r);
    if (space.visibility === RisqVisibilityLevel.SPY || space.visibility === RisqVisibilityLevel.POOR) {
      const badge_size = 0.15 * hexagon_height;
      const badge_p =
        space.visibility === RisqVisibilityLevel.POOR
          ? { x: this.xi() + 0.05 * this.w(), y: yi }
          : { x: this.xi() + 0.1 * this.w(), y: yi + 0.05 * hexagon_height };
      ctx.drawImage(
        this.risq.getIcon(`icons/${space.visibility === RisqVisibilityLevel.SPY ? 'eye' : 'no_eye'}64`),
        badge_p.x,
        badge_p.y,
        badge_size,
        badge_size
      );
      this.eye_badge_hover.ps = badge_p;
      this.eye_badge_hover.pe = { x: badge_p.x + badge_size, y: badge_p.y + badge_size };
    } else {
      this.eye_badge_hover.hovered = false;
    }
    if ((this.visibility ?? 0) >= RisqVisibilityLevel.FOG && space.zones) {
      const draw_zone_icon = (zone: RisqZone, p: Point2D, icon_r: number) => {
        let icon: HTMLImageElement | HTMLCanvasElement | undefined;
        if (zone.resource) {
          icon = this.risq.getIcon(resourceImage(zone.resource));
        } else if (zone.building) {
          const building_color = this.risq.getGame()?.players[zone.building.player_id]?.color;
          const building_image = buildingImage(zone.building.building_id, zone.building.under_construction);
          icon = building_color
            ? this.risq.getPlayerColoredIcon(building_image, building_color)
            : this.risq.getIcon(building_image);
        }
        if (!icon) {
          return;
        }
        ctx.drawImage(icon, p.x - icon_r, p.y - icon_r, 2 * icon_r, 2 * icon_r);
      };
      ctx.strokeStyle = borderStrokeStyle(owner_color, 0.7);
      ctx.lineWidth = 0.5;
      let zone = space.zones[1][1];
      const zone_fill = getZoneFill(zone, RisqViewMode.ALL, undefined, true, 4);
      if (curr_zone.x === 1 && curr_zone.y === 1) {
        zone_fill.addColor(255, 255, 255, 0.2);
      }
      ctx.fillStyle = zone_fill.getString();
      drawHexagon(ctx, c, inner_r);
      draw_zone_icon(zone, c, 0.4 * inner_r);
      const a = Math.PI / 3;
      const mid_r = 0.5 * (inner_r + r);
      const icon_r = 0.32 * (r - inner_r);
      for (let i = 0; i < 6; i++) {
        const direction_vector = OUTER_ZONE_INDICES[i];
        zone = space.zones[direction_vector.x][direction_vector.y];
        const zone_fill = getZoneFill(zone, RisqViewMode.ALL, undefined, true, 4);
        if (curr_zone.x === direction_vector.x && curr_zone.y === direction_vector.y) {
          zone_fill.addColor(255, 255, 255, 0.2);
        }
        ctx.fillStyle = zone_fill.getString();
        ctx.beginPath();
        ctx.lineTo(c.x + inner_r * Math.cos(a * i + Math.PI / 6), c.y + inner_r * Math.sin(a * i + Math.PI / 6));
        ctx.lineTo(c.x + inner_r * Math.cos(a * i + Math.PI / 2), c.y + inner_r * Math.sin(a * i + Math.PI / 2));
        ctx.lineTo(c.x + r * Math.cos(a * i + Math.PI / 2), c.y + r * Math.sin(a * i + Math.PI / 2));
        ctx.lineTo(c.x + r * Math.cos(a * i + Math.PI / 6), c.y + r * Math.sin(a * i + Math.PI / 6));
        ctx.closePath();
        ctx.stroke();
        ctx.fill();
        const angle_mid = a * i + Math.PI / 3;
        draw_zone_icon(zone, { x: c.x + mid_r * Math.cos(angle_mid), y: c.y + mid_r * Math.sin(angle_mid) }, icon_r);
      }
    }
    return hexagon_height + separator_distance;
  }

  private drawZone(ctx: CanvasRenderingContext2D, data: { space: RisqSpace; zone: RisqZone }) {
    let yi = this.yi() + this.drawName(ctx, data.space.display_name);
    drawText(ctx, 'zone', {
      p: { x: this.xc(), y: yi },
      w: this.w(),
      fill_style: 'black',
      align: 'center',
      font: '18px serif',
    });
    yi += 26;
    const separator_distance = 8;
    yi += this.drawSpaceHexagon(ctx, data.space, separator_distance, yi, coordinateToIndex(1, data.zone.coordinate));
    this.drawSeparator(ctx, yi);
    yi += separator_distance;
    const max_image_size = 36; // it should be this size
    const u_img_mult = 1.3;
    const units_per_row = Math.floor((0.8 * this.w() - 1.6 * max_image_size) / (u_img_mult * max_image_size));
    const economic_rows = Math.ceil(data.zone.economic_units.length / units_per_row);
    const military_rows = Math.ceil(data.zone.military_units.length / units_per_row);
    const unit_count_rows = this.visibility === RisqVisibilityLevel.POOR && !!data.zone.unit_count ? 1 : 0;
    const rows = 1 + unit_count_rows + economic_rows + military_rows;
    const image_size = Math.min(
      max_image_size,
      (1 / rows) * (0.6 * this.h() - separator_distance - (rows - 1) * separator_distance)
    );
    ctx.fillStyle = 'black';
    const draw_row = (img: CanvasImageSource, text: string, hover_data?: RectHoverData) => {
      const ps = { x: this.xi() + 0.1 * this.w(), y: yi };
      const pe = { x: ps.x + 0.8 * this.w(), y: ps.y + image_size };
      if (hover_data?.hovered) {
        ctx.strokeStyle = 'transparent';
        if (hover_data?.clicked) {
          ctx.fillStyle = 'rgba(250, 250, 250, 0.4)';
        } else {
          ctx.fillStyle = 'rgba(210, 210, 210, 0.25)';
        }
        drawRect(ctx, ps, pe.x - ps.x, pe.y - ps.y);
      }
      ctx.drawImage(img, ps.x, ps.y, image_size, image_size);
      drawText(ctx, `: ${text}`, {
        p: { x: ps.x + image_size + 2, y: yi + 0.5 * image_size },
        w: 0.8 * this.w() - image_size - 2,
        fill_style: 'black',
        align: 'left',
        baseline: 'middle',
        font: `bold ${image_size}px serif`,
      });
      if (!!hover_data) {
        hover_data.ps = ps;
        hover_data.pe = pe;
      }
    };
    if (!!data.zone.resource) {
      draw_row(
        this.risq.getIcon(resourceImage(data.zone.resource)),
        data.zone.resource.display_name,
        data.zone.resource.hover_data
      );
    } else {
      const building_image = buildingImage(data.zone.building?.building_id, data.zone.building?.under_construction);
      const building_color = data.zone.building
        ? this.risq.getGame()?.players[data.zone.building.player_id]?.color
        : undefined;
      draw_row(
        building_color
          ? this.risq.getPlayerColoredIcon(building_image, building_color)
          : this.risq.getIcon(building_image),
        data.zone.building?.display_name ?? 'Empty Plot',
        data.zone.building?.hover_data
      );
    }
    yi += image_size + separator_distance;
    const xi = this.xi() + 0.1 * this.w() + 0.6 * image_size;
    if (unit_count_rows > 0) {
      const villager_img = this.risq.getIcon('icons/villager64');
      const unit_img = this.risq.getIcon('icons/unit64');
      const combo_icon = this.risq
        .getImageCache()
        .getImage(comboUnitIconKey(false), COMBO_UNIT_ICON_SIZE, [villager_img, unit_img], (combo_ctx) =>
          drawComboUnitIcon(combo_ctx, villager_img, unit_img)
        );
      if (combo_icon) {
        draw_row(combo_icon, data.zone.unit_count!.toString());
      }
      yi += image_size + separator_distance;
    }
    if (data.zone.economic_units.length > 0) {
      draw_row(this.risq.getIcon('icons/villager64'), '');
      let i = 1;
      let j = 0;
      for (const u of data.zone.economic_units) {
        const unit = data.zone.units.get(u);
        if (!unit) {
          continue;
        }
        const p = {
          x: xi + i * u_img_mult * image_size,
          y: yi + j * u_img_mult * image_size,
        };
        this.drawUnitImage(ctx, unit, p, image_size);
        i++;
        if (i > units_per_row) {
          i = 1;
          j++;
        }
      }
      yi += economic_rows * u_img_mult * image_size + separator_distance;
    }
    if (data.zone.military_units.length > 0) {
      draw_row(this.risq.getIcon('icons/unit64'), '');
      let i = 1;
      let j = 0;
      for (const u of data.zone.military_units) {
        const unit = data.zone.units.get(u);
        if (!unit) {
          continue;
        }
        const p = {
          x: xi + i * u_img_mult * image_size,
          y: yi + j * u_img_mult * image_size,
        };
        this.drawUnitImage(ctx, unit, p, image_size);
        i++;
        if (i > units_per_row) {
          i = 1;
          j++;
        }
      }
      yi += military_rows * u_img_mult * image_size + separator_distance;
    }
  }

  private drawName(ctx: CanvasRenderingContext2D, name: string): number {
    const text_size = Math.min(40, (1 / 12) * this.size.y);
    drawText(ctx, name, {
      p: { x: this.xc(), y: this.yi() + 3 },
      w: this.w(),
      fill_style: 'black',
      align: 'center',
      font: `bold ${0.85 * text_size}px serif`,
    });
    return text_size + 3;
  }

  private drawImage(ctx: CanvasRenderingContext2D, yi: number, img_name: string, color?: ColorRGB): number {
    ctx.beginPath();
    const max_img_height = 0.25 * this.size.y - yi + this.yi();
    const img_height = Math.min(max_img_height - 6, 0.8 * this.w());
    const icon = color ? this.risq.getPlayerColoredIcon(img_name, color) : this.risq.getIcon(img_name);
    ctx.drawImage(icon, 0.5 * (this.w() - img_height), yi, img_height, img_height);
    return max_img_height;
  }

  private drawSeparator(ctx: CanvasRenderingContext2D, yi: number) {
    ctx.strokeStyle = 'rgba(60, 60, 60, 0.7)';
    ctx.lineWidth = 2;
    drawLine(ctx, { x: this.xi() + 0.1 * this.w(), y: yi }, { x: this.xf() - 0.1 * this.w(), y: yi });
  }

  private drawCombatStats(
    ctx: CanvasRenderingContext2D,
    yi: number,
    yf: number,
    cs: RisqCombatStats,
    current_stamina: number
  ) {
    const gap_size = 4;
    const num_rows = 4; // health, attack, piercing, defense
    if (!(yf - yi > (num_rows + 1) * gap_size)) {
      return;
    }
    const row_size = (1 / num_rows) * (yf - yi);
    yi += gap_size;
    const xi = this.xi() + 0.1 * this.w();
    const health_height = Math.min(14, 0.4 * row_size);
    ctx.strokeStyle = UNIT_HEALTHBAR_COLOR_BACKGROUND;
    ctx.lineWidth = 0.4;
    ctx.fillStyle = UNIT_HEALTHBAR_COLOR_BACKGROUND;
    drawRect(ctx, { x: xi, y: yi }, 0.8 * this.w(), health_height);
    if (cs.max_health > 0 && cs.health > 0) {
      ctx.fillStyle = UNIT_HEALTHBAR_COLOR_HEALTH;
      drawRect(ctx, { x: xi, y: yi }, (cs.health / cs.max_health) * 0.8 * this.w(), health_height);
    }
    this.healthbar_row.ps = { x: xi, y: yi };
    this.healthbar_row.pe = { x: xi + 0.8 * this.w(), y: yi + health_height };
    const info_row_y = yi + health_height + 0.1 * row_size;
    drawText(ctx, `${Math.round(cs.health)} / ${cs.max_health}`, {
      p: { x: xi, y: info_row_y },
      w: 0.8 * this.w(),
      fill_style: 'black',
      align: 'left',
      font: `${health_height}px serif`,
    });
    const row_right = xi + 0.8 * this.w();
    const stamina_text = `${current_stamina}`;
    ctx.font = `${health_height}px serif`;
    const stamina_text_width = ctx.measureText(stamina_text).width;
    ctx.drawImage(
      this.risq.getIcon('risq/icons/stamina'),
      row_right - stamina_text_width - health_height - 4,
      info_row_y,
      health_height,
      health_height
    );
    drawText(ctx, stamina_text, {
      p: { x: row_right, y: info_row_y },
      w: stamina_text_width,
      fill_style: 'black',
      align: 'right',
      font: `${health_height}px serif`,
    });
    yi += row_size + gap_size;
    const draw_stat_row = (stats: [string, number][]) => {
      let xi = this.xi() + 0.1 * this.w();
      const dx = (0.8 * this.w()) / stats.length;
      const image_size = Math.min(0.7 * row_size, 0.3 * dx);
      let drew_one = false;
      for (const s of stats.filter((s) => !!s[1])) {
        drew_one = true;
        ctx.drawImage(this.risq.getIcon(s[0]), xi, yi, image_size, image_size);
        drawText(ctx, s[1].toString(), {
          p: { x: xi + image_size + 0.1 * dx, y: yi + 0.5 * image_size },
          w: dx - image_size,
          fill_style: 'black',
          align: 'left',
          baseline: 'middle',
          font: `${0.9 * image_size}px serif`,
        });
        xi += dx;
      }
      if (drew_one) {
        yi += row_size + gap_size;
      }
    };
    if (cs.attack_type !== RisqAttackType.NONE) {
      draw_stat_row([
        ['risq/icons/attack_blunt', cs.attack_blunt],
        ['risq/icons/attack_piercing', cs.attack_piercing],
        ['risq/icons/attack_magic', cs.attack_magic],
      ]);
      draw_stat_row([
        ['risq/icons/penetration_blunt', cs.penetration_blunt],
        ['risq/icons/penetration_piercing', cs.penetration_piercing],
        ['risq/icons/penetration_magic', cs.penetration_magic],
      ]);
    }
    draw_stat_row([
      ['risq/icons/defense_blunt', cs.defense_blunt],
      ['risq/icons/defense_piercing', cs.defense_piercing],
      ['risq/icons/defense_magic', cs.defense_magic],
    ]);
  }

  private objectHoverLogic(
    m: Point2D,
    object: RisqUnit | RisqBuilding | RisqResource | undefined,
    object_type: HoverableObjectType
  ): boolean {
    if (!object) {
      return false;
    }
    if (
      m.x < object.hover_data.ps.x ||
      m.y < object.hover_data.ps.y ||
      m.x > object.hover_data.pe.x ||
      m.y > object.hover_data.pe.y
    ) {
      object.hover_data.hovered = false;
      return false;
    } else {
      if (
        !!this.hovered_object &&
        (this.hovered_object.internal_id !== object.internal_id || this.hovered_object_type !== object_type)
      ) {
        this.hovered_object.hover_data.hovered = false;
        this.hovered_object.hover_data.clicked = false;
      }
      object.hover_data.hovered = true;
      this.hovered_object = object;
      this.hovered_object_type = !!object ? object_type : HoverableObjectType.NONE;
      return true;
    }
  }

  private rowHovered(m: Point2D, hover_data: RectHoverData): boolean {
    if (m.x < hover_data.ps.x || m.y < hover_data.ps.y || m.x > hover_data.pe.x || m.y > hover_data.pe.y) {
      hover_data.hovered = false;
      return false;
    }
    hover_data.hovered = true;
    return true;
  }

  scroll(dy: number, mode: number): boolean {
    if (this.showOrderRows() && this.order_rows_list.isHovering()) {
      return this.order_rows_list.scroll(dy, mode);
    }
    return false;
  }

  mousemove(canvas: Point2D, screen: Point2D, transform: BoardTransformData): boolean {
    if (this.close_button.mousemove(canvas, screen, transform)) {
      return true;
    }
    for (const button of this.buttons) {
      button.mousemove(canvas, screen, transform);
    }
    if (this.showOrderRows() && this.order_rows_list.mousemove(canvas, screen, transform)) {
      return true;
    }
    const m = screen;
    if (m.x > this.xi() && m.y > this.yi() && m.x < this.xf() && m.y < this.yf()) {
      this.hovering = true;
    } else {
      this.hovering = false;
    }
    switch (this.data?.data_type) {
      case LeftPanelDataType.SPACE:
      case LeftPanelDataType.ZONE:
        const space: RisqSpace =
          this.data?.data_type === LeftPanelDataType.SPACE ? this.data.data : this.data.data.space;
        this.rowHovered(m, this.eye_badge_hover);
        const new_hovered_zone = resolveHoveredZones(m, space, this.hexagon_r, this.hexagon_c, true);
        if (!!this.hovered_zone && !equalsPoint2D(this.hovered_zone.coordinate, new_hovered_zone?.coordinate)) {
          this.hovered_zone.hovered = false;
          this.hovered_zone.clicked = false;
        }
        this.hovered_zone = new_hovered_zone;
        if (this.data.data_type === LeftPanelDataType.SPACE) {
          this.space_villager_row_button?.mousemove(canvas, screen, transform);
          this.space_military_row_button?.mousemove(canvas, screen, transform);
        }
        if (this.data.data_type === LeftPanelDataType.ZONE) {
          const zone = this.data.data.zone;
          const unit_ids: number[] = [...zone.economic_units, ...zone.military_units];
          for (const unit_id of unit_ids) {
            const unit = zone.units.get(unit_id);
            if (!unit) {
              continue;
            }
            this.objectHoverLogic(m, unit, HoverableObjectType.UNIT);
          }
          this.objectHoverLogic(m, zone.resource, HoverableObjectType.RESOURCE);
          this.objectHoverLogic(m, zone.building, HoverableObjectType.BUILDING);
        }
        break;
      case LeftPanelDataType.MULTIPLE_PLAYERS_UNITS:
      case LeftPanelDataType.UNITS_BY_TYPE:
      case LeftPanelDataType.ECONOMIC_UNITS:
      case LeftPanelDataType.MILITARY_UNITS: {
        const unit_groups =
          this.data.data_type === LeftPanelDataType.MULTIPLE_PLAYERS_UNITS
            ? this.data.data.units_by_player.flatMap(([, units]) => units)
            : this.data.data.units;
        for (const unit_data of unit_groups) {
          if (unit_data.units.size < 1) {
            continue;
          }
          for (const unit_id of unit_data.units.values()) {
            const unit = this.resolveUnit(unit_data.player_id, unit_id);
            if (!unit) {
              continue;
            }
            this.objectHoverLogic(m, unit, HoverableObjectType.UNIT);
          }
        }
        break;
      }
      case LeftPanelDataType.UNIT:
      case LeftPanelDataType.BUILDING:
        this.rowHovered(m, this.healthbar_row);
        break;
      default:
        break;
    }
    return this.isHovering();
  }

  private handleUnitGridClick(
    space: RisqSpace | undefined,
    groups: [number, UnitByTypeData[]][],
    hovered_unit: RisqUnit,
    e: MouseEvent
  ) {
    if (!space) {
      return;
    }
    if (e.shiftKey) {
      const player_units = groups.find(([player_id]) => player_id === hovered_unit.player_id)?.[1] ?? [];
      const units_by_player = new Map<number, UnitByTypeData[]>([
        [hovered_unit.player_id, player_units.filter((u) => u.unit_id === hovered_unit.unit_id)],
      ]);
      this.openPanel({ data_type: LeftPanelDataType.UNITS, data: { space, units_by_player } }, this.visibility ?? 0);
    } else if (e.ctrlKey) {
      const units_by_player = new Map<number, UnitByTypeData[]>();
      for (const [player_id, units] of groups) {
        const new_units = units
          .map((u) => ({ ...u, units: new Set([...u.units].filter((id) => id !== hovered_unit.internal_id)) }))
          .filter((u) => u.units.size > 0);
        if (new_units.length > 0) {
          units_by_player.set(player_id, new_units);
        }
      }
      this.openPanel({ data_type: LeftPanelDataType.UNITS, data: { space, units_by_player } }, this.visibility ?? 0);
    } else {
      this.openPanel({ data_type: LeftPanelDataType.UNIT, data: hovered_unit }, this.visibility ?? 0);
    }
  }

  mousedown(e: MouseEvent): boolean {
    if (this.close_button.mousedown(e)) {
      return true;
    }
    switch (this.data?.data_type) {
      case LeftPanelDataType.SPACE:
      case LeftPanelDataType.ZONE:
        if (!!this.hovered_zone) {
          this.hovered_zone.clicked = true;
        } else if (!!this.hovered_object) {
          this.hovered_object.hover_data.clicked = true;
        } else if (this.data.data_type === LeftPanelDataType.SPACE) {
          this.space_villager_row_button?.mousedown(e);
          this.space_military_row_button?.mousedown(e);
        }
        break;
      case LeftPanelDataType.UNITS:
      case LeftPanelDataType.MULTIPLE_PLAYERS_UNITS:
      case LeftPanelDataType.UNITS_BY_TYPE:
      case LeftPanelDataType.ECONOMIC_UNITS:
      case LeftPanelDataType.MILITARY_UNITS:
        if (!!this.hovered_object) {
          this.hovered_object.hover_data.clicked = true;
        }
        break;
      default:
        break;
    }
    for (const button of this.buttons) {
      button.mousedown(e);
    }
    if (this.showOrderRows()) {
      this.order_rows_list.mousedown(e);
    }
    return this.isHovering();
  }

  mouseup(e: MouseEvent) {
    this.close_button.mouseup(e);
    if (this.showOrderRows()) {
      this.order_rows_list.mouseup(e);
    }
    switch (this.data?.data_type) {
      case LeftPanelDataType.SPACE:
      case LeftPanelDataType.ZONE:
        const space: RisqSpace =
          this.data?.data_type === LeftPanelDataType.SPACE ? this.data.data : this.data.data.space;
        if (!!this.hovered_zone && this.hovered_zone.clicked) {
          this.hovered_zone.clicked = false;
          if (this.hovered_zone.hovered) {
            if (
              this.data.data_type === LeftPanelDataType.ZONE &&
              this.hovered_zone.coordinate_key === this.data.data.zone.coordinate_key
            ) {
              this.openPanel({ data_type: LeftPanelDataType.SPACE, data: space }, this.visibility ?? 0);
            } else {
              this.openPanel(
                {
                  data_type: LeftPanelDataType.ZONE,
                  data: {
                    space,
                    zone: this.hovered_zone,
                  },
                },
                this.visibility ?? 0
              );
            }
          }
        } else if (!!this.hovered_object && this.hovered_object.hover_data.clicked) {
          this.hovered_object.hover_data.clicked = false;
          if (this.hovered_object.hover_data.hovered) {
            switch (this.hovered_object_type) {
              case HoverableObjectType.UNIT:
                if (e.shiftKey && this.data.data_type === LeftPanelDataType.ZONE) {
                  const units_by_type = this.data.data.zone.units_by_type.get(
                    (this.hovered_object as RisqUnit).player_id
                  );
                  if (!!units_by_type) {
                    this.openPanel(
                      {
                        data_type: LeftPanelDataType.UNITS_BY_TYPE,
                        data: {
                          space,
                          units: [...units_by_type.values()].filter(
                            (u: UnitByTypeData) => u.unit_id === (this.hovered_object as RisqUnit).unit_id
                          ),
                        },
                      },
                      this.visibility ?? 0
                    );
                  }
                } else {
                  // @ts-ignore
                  this.openPanel({ data_type: LeftPanelDataType.UNIT, data: this.hovered_object }, this.visibility);
                }
                break;
              case HoverableObjectType.BUILDING:
                // @ts-ignore
                this.openPanel({ data_type: LeftPanelDataType.BUILDING, data: this.hovered_object }, this.visibility);
                break;
              case HoverableObjectType.RESOURCE:
                // @ts-ignore
                this.openPanel({ data_type: LeftPanelDataType.RESOURCE, data: this.hovered_object }, this.visibility);
                break;
              default:
                break;
            }
          }
        } else if (this.data.data_type === LeftPanelDataType.SPACE) {
          this.space_villager_row_button?.mouseup(e);
          this.space_military_row_button?.mouseup(e);
        }
        break;
      case LeftPanelDataType.MULTIPLE_PLAYERS_UNITS:
        if (!!this.hovered_object && this.hovered_object.hover_data.clicked) {
          this.hovered_object.hover_data.clicked = false;
          if (this.hovered_object.hover_data.hovered) {
            this.handleUnitGridClick(
              this.data.data.space,
              this.data.data.units_by_player,
              this.hovered_object as RisqUnit,
              e
            );
          }
        }
        break;
      case LeftPanelDataType.UNITS_BY_TYPE:
      case LeftPanelDataType.ECONOMIC_UNITS:
      case LeftPanelDataType.MILITARY_UNITS:
        if (!!this.hovered_object && this.hovered_object.hover_data.clicked) {
          this.hovered_object.hover_data.clicked = false;
          if (this.hovered_object.hover_data.hovered) {
            const hovered_unit = this.hovered_object as RisqUnit;
            this.handleUnitGridClick(
              this.data.data.space,
              [[hovered_unit.player_id, this.data.data.units]],
              hovered_unit,
              e
            );
          }
        }
        break;
      default:
        break;
    }
    for (const button of this.buttons) {
      button.mouseup(e);
    }
  }

  xi(): number {
    return 0;
  }
  yi(): number {
    return 0.5 * (this.risq.canvasSize().height - this.size.y);
  }
  xf(): number {
    return this.showing ? this.xi() + this.w() : 0;
  }
  yf(): number {
    return this.showing ? this.yi() + this.h() : 0;
  }
  xc(): number {
    return this.xi() + 0.5 * this.w();
  }
  yc(): number {
    return this.yi() + 0.5 * this.h();
  }
  w(): number {
    return this.size.x;
  }
  h(): number {
    return this.size.y;
  }
}
