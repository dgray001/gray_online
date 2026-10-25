from __future__ import annotations

import argparse
import json
from pathlib import Path

from bounds import bounds, initialize
from common import Definitions, Record, rule, variable
from estimates import composition, unidentified, local_observations, strategy
from locations import army, base, home
from observations import observations
from parameters import PARAMETERS, parameters
from budget import resource_budgets, spending_rates, worker_targets
from economy import food_supply, infrastructure, production_capacity, workforce
from gathering import allocate_workers, gathering, worker_groups
from foundations import foundation_failures, foundation_observations
from construction import (
    expansion_buildings, first_barracks, housing_crew_sizing, isolate_builders,
    recover_abandoned_foundations, repairs, service_unbuilt_foundations,
    settlement_and_shelter, urgent_smith
)
from settlement import nomad_clustering, vc_urgency_rules
from sites import site_weights
from mix import composition as military_composition, unit_targets
from production import mercenaries, military, opening, renew_farms, upgrades
from defense import base_defense, civilian_defense, civilian_evacuation, threat_home_rules
from raiding import raiding_state, raiding_tactics
from scouting import home_inspector, scouting_state, scouting_tactics
from army import (
    army_formation_rules, army_local_odds_rules, army_maneuver_rules,
    army_target_selection, assault_odds_rules
)
from combat import combat_micro_rules

CONFIG = Path(__file__).resolve().parents[3]


def generate(directory: Path, overrides: dict[str, float]) -> Record:
    values = parameters(overrides)
    definitions = Definitions(directory)
    actions = [variable(f"tune_{name}", value) for name, value in values.items()]
    phases = [actions, observations(definitions), foundation_observations(), site_weights(), home(), base(), threat_home_rules(),
              scouting_state(), raiding_state(), army_formation_rules(), army(),
              [initialize()], composition(), unidentified(), local_observations(), bounds(definitions), strategy(),
              assault_odds_rules(), army_local_odds_rules(), army_target_selection()]
    phases.extend([production_capacity(definitions), military_composition(definitions, values), workforce(definitions),
                   unit_targets(values), infrastructure(definitions), food_supply(definitions), vc_urgency_rules(),
                   spending_rates(definitions), resource_budgets(definitions),
                   housing_crew_sizing(definitions), isolate_builders(),
                   opening(), urgent_smith(), settlement_and_shelter(int(values["max_new_farms"])), first_barracks(),
                   military(), upgrades(), expansion_buildings(), mercenaries(),
                   civilian_evacuation(), foundation_failures(),
                   service_unbuilt_foundations(), recover_abandoned_foundations(), repairs(), isolate_builders(),
                   nomad_clustering(), worker_targets(), worker_groups(), allocate_workers(),
                   renew_farms(), gathering(values),
                   scouting_tactics(), raiding_tactics(),
                   army_maneuver_rules(),
                   combat_micro_rules(),
                   base_defense(), home_inspector(), civilian_defense()])
    return dict(rules=[rule(phase) for phase in phases])


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
