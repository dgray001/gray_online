from __future__ import annotations

import re
import unittest
from collections import defaultdict
from pathlib import Path

from budget import resource_budgets, worker_targets
from common import Definitions, Record, variable
from gold import gold_expansion, mercenary_rates


def test_definitions() -> Definitions:
    definitions = Definitions(Path(__file__).resolve().parents[4] / "tests" / "config")
    definitions.units = {target: definitions.units[source] for target, source in ((1, 11), (101, 21), (102, 31), (103, 41))}
    definitions.buildings = {target: definitions.buildings[source] for target, source in
                            ((1, 201), (2, 202), (3, 203), (11, 209), (21, 207), (22, 205), (23, 208))}
    return definitions


def evaluate(actions: list[Record], initial: dict[str, float]) -> dict[str, float]:
    values = defaultdict(float, initial)
    definitions = test_definitions()

    def config(table: str, field: str, item: int) -> float:
        return float(getattr(definitions, table + "s")[item]["cost"].get(field.removeprefix("cost_"), 0))

    def game_var(field: str) -> float:
        return values["game_" + field]

    environment = dict(var=values.__getitem__, gameVar=game_var, config=config, min=min, max=max, round=round)
    for action in actions:
        if not isinstance(action["value"], dict):
            expression = re.sub(r"(var|gameVar)\(([\w.]+)\)", r'\1("\2")', str(action["value"]))
            expression = re.sub(r"config\((\w+), (\w+),", r'config("\1", "\2",', expression)
            values[action["name"]] = float(eval(expression, {"__builtins__": {}}, environment))
    return values


def allocation(weight: float, competing: float = 0, available: float = 10, protected: float = 0) -> dict[str, float]:
    return evaluate(worker_targets(), dict(own_1=10, worker_goal=10, n_available_builders=available,
                    tune_investment_share=1, protected_food=protected, investment_food=competing,
                    raw_investment_food=competing, raw_investment_wood=2, raw_investment_stone=1,
                    investment_wood=2 * weight, investment_stone=weight, capacity_food=100,
                    capacity_wood=100, capacity_stone=100, farming=0, worker_income=6))


class GoldInvestmentTests(unittest.TestCase):
    def test_single_demand_weight_cancels(self) -> None:
        normal, tiny = allocation(0.8), allocation(0.00001)
        for field in ("workers_food", "workers_wood", "workers_stone", "investment_workers"):
            self.assertAlmostEqual(normal[field], tiny[field])

    def test_weight_changes_competing_shares(self) -> None:
        low, high = allocation(0.1, competing=3), allocation(0.8, competing=3)
        self.assertGreater(high["investment_scale"] * high["investment_wood"],
                           low["investment_scale"] * low["investment_wood"])

    def test_production_shortfall_leaves_no_investment_workers(self) -> None:
        result = allocation(0.8, available=1, protected=2)
        self.assertEqual(result["investment_workers"], 0)
        self.assertEqual(result["workers_food"], 1)

    def test_payback_spreads_funding(self) -> None:
        definitions = test_definitions()
        initial = dict(worker_income=6, tune_planning_horizon=10, tune_construction_horizon=3,
                       tune_renew_workers=2, tune_gold_expansion_weight=0.8, capacity_wood=100)
        base = evaluate(resource_budgets(definitions), initial | dict(want_21=0, tune_gold_expansion_payback_turns=6))
        short = evaluate(resource_budgets(definitions), initial | dict(want_21=12, tune_gold_expansion_payback_turns=6))
        long = evaluate(resource_budgets(definitions), initial | dict(want_21=12, tune_gold_expansion_payback_turns=12))
        self.assertAlmostEqual(short["raw_investment_wood"] - base["raw_investment_wood"],
                               2 * (long["raw_investment_wood"] - base["raw_investment_wood"]))

    def test_zero_weight_disables_expansion(self) -> None:
        result = evaluate([gold_expansion()[-1]], dict(gold_site_income=2, gold_shortage=5, tune_gold_expansion_weight=0))
        self.assertEqual(result["want_21"], 0)

    def test_mercenary_demand_without_local_production(self) -> None:
        action = next(action for action in mercenary_rates() if action["action"] == "for_each")["do"][0]
        initial = {"offer.id": 101, "offer.cost_gold": 90, "goal_101": 15, "own_101": 5,
                   "game_population_max": 100, "game_population": 10, "tune_planning_horizon": 10}
        condition = action["when"]["value_at_least"]["value"]
        self.assertEqual(evaluate([variable("eligible", condition)], initial)["eligible"], 1)
        self.assertEqual(evaluate(action["then"], initial)["mercenary_rate_gold"], 90)
        self.assertEqual(evaluate([variable("eligible", condition)], initial | dict(native_producers_101=1))["eligible"], 0)

    def test_gold_income_sets_expansion_target_without_barracks(self) -> None:
        initial = dict(gold_shortage=10, tune_gold_expansion_weight=0.8)
        low = evaluate([gold_expansion()[-1]], initial | dict(gold_site_income=2))
        high = evaluate([gold_expansion()[-1]], initial | dict(gold_site_income=20))
        self.assertEqual(low["want_21"], 5)
        self.assertEqual(high["want_21"], 0.5)
