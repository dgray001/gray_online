from __future__ import annotations

import argparse
import json
from pathlib import Path
from typing import Any

from bounds import bounds, initialize
from common import Definitions, Record, rule, variable
from estimates import composition, unidentified, local_observations, strategy
from locations import army, base, home
from observations import observations
from parameters import PARAMETERS, parameters
from budget import resource_budgets, spending_rates, worker_targets
from gold import gold_expansion, mercenary_rates
from economy import food_supply, infrastructure, production_capacity, rax_reserve, workforce
from gathering import allocate_workers, gathering, worker_groups
from foundations import foundation_failures, foundation_observations
from construction import (
    apply_outpost_safety, buildable_spaces, buildable_zones, expansion_buildings, farms, first_barracks, housing_crew_sizing, isolate_builders,
    recover_abandoned_foundations, repairs, service_unbuilt_foundations,
    settlement_and_shelter, urgent_smith
)
from settlement import nomad_clustering, vc_urgency_rules
from sites import configure as configure_sites, site_weights
from mix import composition as military_composition, unit_targets
from production import mercenaries, military, opening, renew_farms, upgrades
from defense import base_defense, civilian_defense, civilian_evacuation, threat_home_rules
from raiding import raiding_state, raiding_tactics
from scouting import home_inspector, scouting_state, scouting_tactics
from army import (
    army_formation_rules, army_local_odds_rules, army_maneuver_rules,
    army_target_selection, assault_odds_rules
)
from combat import attack_stack_rules, combat_micro_rules
from hiring import hire_destinations
from techs import tech_demand, tech_hold, tech_buy
from holders import holder_observations, holder_orders, release_holders
from policies import apply_policies

CONFIG = Path(__file__).resolve().parents[3]


def narrow_unidentified_sums(model: Record) -> None:
    def walk(o: Any) -> None:
        if isinstance(o, dict):
            inner = o.get("sum")
            where = inner.get("where") if isinstance(inner, dict) else None
            if isinstance(inner, dict) and inner.get("kind") == "spaces" and inner.get("of") == "var(item.unit_count)" and isinstance(where, dict) and where.get("in_set") == "poor_vision":
                where["in_set"] = "unidentified_spaces"
            for v in o.values():
                walk(v)
        elif isinstance(o, list):
            for v in o:
                walk(v)

    for rule in model["rules"]:
        walk(rule["then"])


def construction_phases(order: int) -> list[list[Record]]:
    sites = [buildable_spaces(), buildable_zones()]
    sequences = {0: [opening(), *sites, urgent_smith(), settlement_and_shelter(), first_barracks()],
                 1: [opening(), *sites, urgent_smith(), first_barracks(), settlement_and_shelter()],
                 2: [opening(), *sites, first_barracks(), urgent_smith(), settlement_and_shelter()],
                 3: [*sites, first_barracks(), opening(), urgent_smith(), settlement_and_shelter()]}
    return sequences[order]


def generate(directory: Path, overrides: dict[str, float]) -> Record:
    values = parameters(overrides)
    configure_sites(values)
    definitions = Definitions(directory)
    actions = [variable(f"tune_{name}", value) for name, value in values.items() if not name.startswith("issue_")]
    phases = [actions, observations(definitions), foundation_observations(), site_weights(), home(), base(), threat_home_rules(),
              scouting_state(), raiding_state(), release_holders(), holder_observations(), holder_orders(), army_formation_rules(), army(),
              [initialize()], composition(), unidentified(), local_observations(), bounds(definitions), strategy(),
              assault_odds_rules(), army_local_odds_rules(), army_target_selection(), hire_destinations()]
    phases.extend([production_capacity(definitions), military_composition(definitions, values), workforce(definitions),
                   unit_targets(values), infrastructure(definitions), food_supply(definitions), vc_urgency_rules(),
                   mercenary_rates(), spending_rates(definitions), tech_demand(), gold_expansion(), resource_budgets(definitions), rax_reserve(), tech_hold(values),
                   housing_crew_sizing(definitions), isolate_builders(),
                   *construction_phases(int(values["rax_order"])), farms(int(values["max_new_farms"])),
                   upgrades(), military(), *([tech_buy(values)] if int(values.get("issue_tech_order", 0)) == 1 else []), expansion_buildings(), mercenaries(),
                   civilian_evacuation(), foundation_failures(),
                   service_unbuilt_foundations(), recover_abandoned_foundations(), repairs(), isolate_builders(),
                   nomad_clustering(), worker_targets(), worker_groups(), allocate_workers(),
                   renew_farms(), gathering(values),
                   scouting_tactics(), raiding_tactics(),
                   army_maneuver_rules(bool(values["stage_passive"]), bool(values["rest_move"])),
                   attack_stack_rules(int(values["attack_stack"])),
                   combat_micro_rules(values),
                   base_defense(), home_inspector(), civilian_defense(values)])
    model = dict(rules=[rule(phase) for phase in phases])
    apply_policies(model)
    apply_outpost_safety(model, values)
    narrow_unidentified_sums(model)
    return model


def main() -> None:
    parser = argparse.ArgumentParser(description="Generate the Risq v7 AI configuration")
    parser.add_argument("settings", nargs="*", metavar="name=value")
    parser.add_argument("--overrides", nargs="*", default=[], metavar="name=value")
    parser.add_argument("--config", type=Path, default=CONFIG)
    parser.add_argument("--out", type=Path)
    parser.add_argument("--list-parameters", action="store_true")
    args = parser.parse_args()
    if args.list_parameters:
        for name, spec in PARAMETERS.items():
            print(f"{name}={spec.default:g} [{spec.minimum:g}, {spec.maximum:g}]")
        return
    overrides: dict[str, float] = {}
    try:
        for override in args.settings + args.overrides:
            name, value = override.split("=", 1)
            overrides[name] = float(value)
        result = generate(args.config, overrides)
    except (ValueError, KeyError) as error:
        parser.error(str(error))
    output = args.out or args.config / "ai" / "v0.7.json"
    output.write_text(json.dumps(result, indent=2) + "\n")
    print(f"{len(result['rules'])} rules -> {output}")


if __name__ == "__main__":
    main()
