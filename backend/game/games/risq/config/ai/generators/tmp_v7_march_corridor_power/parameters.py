from __future__ import annotations

from dataclasses import dataclass
from math import isfinite


@dataclass(frozen=True)
class Parameter:
    default: float
    minimum: float = 0
    maximum: float = float("inf")
    integer: bool = False

    def validate(self, name: str, value: float) -> float:
        if not isfinite(value) or not self.minimum <= value <= self.maximum:
            raise ValueError(f"{name} must be finite and in [{self.minimum}, {self.maximum}]")
        if self.integer and not value.is_integer():
            raise ValueError(f"{name} must be an integer")
        return value


PARAMETERS = {
    "power_blunt": Parameter(1, 0.001), "power_pierce": Parameter(2),
    "power_heavy": Parameter(4), "power_siege": Parameter(1.5),
    "power_villager": Parameter(0.6), "power_farming_bonus": Parameter(0.4),
    "memory_unit_decay": Parameter(0.3), "memory_retention": Parameter(0.98, 0, 1),
    "memory_producer_growth": Parameter(0.2), "memory_recent_weight": Parameter(0.3, 0, 1),
    "memory_loss_weight": Parameter(1, 0, 1), "memory_population_power": Parameter(2),
    "unknown_min_power": Parameter(1, 0.001), "unknown_villager_power": Parameter(0.2),
    "unknown_villager_share": Parameter(0.7, 0, 1), "unknown_weight": Parameter(1),
    "unknown_base_radius": Parameter(3, integer=True),
    "composition_prior": Parameter(1),
    "enemy_smith_bonus": Parameter(0.25), "strategic_tech_bonus": Parameter(0.125),
    "tactical_attack_bonus": Parameter(0.25), "tactical_armor_bonus": Parameter(0.15),
    "worker_income": Parameter(6, 0.001), "worker_farming_income": Parameter(1.5),
    "bound_theoretical_income": Parameter(8, 0.001), "bound_reasonable_income": Parameter(7, 0.001),
    "bound_theoretical_barracks_turn": Parameter(6, 1, integer=True),
    "bound_reasonable_barracks_turn": Parameter(10, 1, integer=True),
    "bound_worker_floor": Parameter(8, 1, integer=True),
    "bound_worker_first": Parameter(10, 1, integer=True),
    "bound_worker_second": Parameter(20, 1, integer=True),
    "bound_worker_third": Parameter(40, 1, integer=True),
    "bound_second_rate": Parameter(2 / 3, 0.001, 1),
    "bound_third_rate": Parameter(0.5, 0.001, 1),
    "bound_military_share": Parameter(0.9, 0, 1),
    "bound_uncertainty_rate": Parameter(0.4), "bound_uncertainty_limit": Parameter(10, 0.001),
    "bound_widening": Parameter(0.1), "bound_interpolation": Parameter(1, 0, 1),
    "local_radius": Parameter(1, integer=True),
    "worker_floor": Parameter(8, 1), "worker_share": Parameter(0.5, 0.01, 1),
    "worker_buffer": Parameter(2), "planning_horizon": Parameter(10, 1),
    "production_lookahead": Parameter(1.5), "production_turnover": Parameter(18, 1),
    "additional_producer_margin": Parameter(0.8), "investment_share": Parameter(0.3, 0, 1),
    "investment_horizon": Parameter(15, 1), "construction_horizon": Parameter(3, 1),
    "bank_surplus_reserve": Parameter(100), "housing_horizon": Parameter(4, 1),
    "gold_expansion_weight": Parameter(0.8), "gold_expansion_payback_turns": Parameter(6, 1),
    "bootstrap_batches": Parameter(3, 1), "emergency_radius": Parameter(2, integer=True),
    "early_blunt_bias": Parameter(3), "mix_sharpness": Parameter(4, 1, 8, True),
    "heavy_value_gate": Parameter(0.8), "pierce_heavies": Parameter(2),
    "farming_workers": Parameter(8), "wild_food_radius": Parameter(1, integer=True),
    "wild_food_walk_scale": Parameter(4, 0.01), "gold_smoothing": Parameter(0.25, 0, 1),
    "smith_urgency_turns": Parameter(6, 1), "redoubt_urgency_turns": Parameter(10, 1),
    "surplus_hire_batches": Parameter(3, 1), "queue_depth": Parameter(2, 1, 10, True),
    "transfer_deadband": Parameter(2), "shortage_turns": Parameter(2, 0.01),
    "transfer_fraction": Parameter(0.25, 0, 1), "gather_nodes_min": Parameter(12, 1, integer=True),
    "gather_nodes_per_worker": Parameter(2, 1), "gather_search_passes": Parameter(2, 1, 8, True),
    "siege_share_cap": Parameter(0.4, 0, 0.9), "siege_escort_floor": Parameter(6, 1),
    "pierce_pikes_gain": Parameter(0.5), "hire_price_estimate": Parameter(1.3, 1),
    "smith_saving_max": Parameter(3, 1), "redoubt_saving_max": Parameter(2, 1),
    "pikes_unit_threshold": Parameter(4), "renew_workers": Parameter(2, 1, integer=True),
    "utility_enemy_prior": Parameter(1), "utility_building_weight": Parameter(1),
    "utility_survival_exponent": Parameter(0.5, 0, 1), "utility_range_bonus": Parameter(1.25, 1),
    "mix_prior_101": Parameter(1), "mix_prior_102": Parameter(0.5), "mix_prior_103": Parameter(1),
    "gather_distance_weight": Parameter(100, 0.01), "gather_depletion_penalty": Parameter(10000),
    "food_gather_speed_weight": Parameter(0),
    "site_dist": Parameter(1, 0.01), "site_expand": Parameter(0), "site_aggr": Parameter(0),
    "site_barracks_aggr": Parameter(0.25), "site_redoubt_aggr": Parameter(0.25),
    "site_safe": Parameter(1, 0.01), "site_spread": Parameter(0.25), "site_vision": Parameter(0),
    "site_region": Parameter(0), "site_terrain": Parameter(4), "site_urgency": Parameter(9),
    "vc_site_walk": Parameter(3.617635, 0.01), "vc_site_food": Parameter(1.148617, 0.01),
    "vc_site_food_second": Parameter(0.093866), "vc_site_build": Parameter(68.95007, 1),
    "vc_site_limit": Parameter(6.260566, 0.1), "vc_age_scale": Parameter(3.134538, 0.1),
    "vc_site_threat": Parameter(10), "vc_cluster": Parameter(0),
    "building_tries": Parameter(3, 1, integer=True), "house_tries": Parameter(1, 1, integer=True),
    "ban_wait": Parameter(4, 1, integer=True), "resend_every": Parameter(4, 1, integer=True),
    "max_new_farms": Parameter(3, 1, 10, integer=True),
    "vil_crush_odds": Parameter(2.0, 0.1), "vil_unknown": Parameter(0.25, 0.0),
    "raid_odds": Parameter(1.4, 0.1), "raid_min": Parameter(3, 1, integer=True),
    "raid_max": Parameter(8, 1, integer=True), "raid_frac": Parameter(0.3, 0.0, 1.0),
    "raid_pause": Parameter(20, 1, integer=True),
    "assault_min": Parameter(10, 1), "assault_odds": Parameter(1.3, 0.1),
    "assault_odds_noeco": Parameter(1.1, 0.1), "power_odds": Parameter(1.05, 0.1),
    "power_odds_noeco": Parameter(0.85, 0.1), "cap_odds": Parameter(0.9, 0.1),
    "cap_odds_noeco": Parameter(0.8, 0.1), "cap_ramp": Parameter(0.8, 0.0, 1.0),
    "retreat_margin": Parameter(0.05, 0.0, 1.0), "engaged_odds": Parameter(1.0, 0.1),
    "engaged_odds_noeco": Parameter(0.8, 0.1), "stage_dist": Parameter(3, 1, integer=True),
    "rest_dist": Parameter(1, 1, integer=True), "rest_turns": Parameter(2, 1, integer=True),
    "retreat_timeout_turns": Parameter(3, 1, 30, True), "retreat_recovery_turns": Parameter(1, 0, 10, True),
    "skip_rest_margin": Parameter(0.3, 0.0), "skip_stage_margin": Parameter(0.6, 0.0),
    "est_close_ref": Parameter(6.0, 1.0), "est_prio_close": Parameter(0.5, 0.0, 1.0),
    "est_prio_behind": Parameter(0.1, 0.0, 1.0), "est_inspect_reach": Parameter(2, 1, integer=True),
    "est_inspect_min": Parameter(2, 1, integer=True), "est_insp_safe": Parameter(1, 0, 1, integer=True),
    "est_insp_cool": Parameter(2.0, 0.0), "evac_threat_min": Parameter(8.0, 1.0),
    "evac_threat_ratio": Parameter(1.5, 0.1),
    "group_share": Parameter(1.0, 0.1, 1.0), "hit_damage": Parameter(1.5, 0.1),
    "kill_ticks": Parameter(1, 1, integer=True),
    "focus": Parameter(4.0, 0.0),
}

PARAMETERS.update({f"matchup_{unit}_{enemy}": Parameter(value)
                   for unit, values in ((101, (1, 1.3, 0.4)), (102, (0.75, 1, 0.8)), (103, (1.3, 0.8, 1)))
                   for enemy, value in zip((101, 102, 103), values)})


def parameters(overrides: dict[str, float]) -> dict[str, float]:
    unknown = overrides.keys() - PARAMETERS.keys()
    if unknown:
        raise ValueError(f"Unknown parameters: {', '.join(sorted(unknown))}")
    values = {name: spec.validate(name, float(overrides.get(name, spec.default)))
              for name, spec in PARAMETERS.items()}
    if not values["bound_worker_first"] <= values["bound_worker_second"] <= values["bound_worker_third"]:
        raise ValueError("Worker-growth breakpoints must be in ascending order")
    return values
