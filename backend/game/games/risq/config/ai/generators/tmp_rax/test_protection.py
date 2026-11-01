from __future__ import annotations

import unittest
from budget import spending_rates
from test_gold import evaluate, test_definitions


class ProtectionTests(unittest.TestCase):
    def rates(self, **overrides: float) -> dict[str, float]:
        definitions = test_definitions()
        actions = [action for action in spending_rates(definitions) if action.get("name") in ("protected_rate_food", "rate_food")]
        for action in actions:
            action["value"] = str(action["value"]).replace("config(unit, cost_food, 401)", "0")
        return evaluate(actions, dict(flow_1=1, flow_101=2, complete_22=2, protected_producers_22=1, own_1=10, goal_1=50,
                                     n_military=10, tune_protected_villager_share=0.8,
                                     tune_protected_military_producers=1, tune_protected_military_count=20) | overrides)

    def test_one_producer_is_protected(self) -> None:
        self.assertEqual(self.rates()["protected_rate_food"], 50 + test_definitions().units[101]["cost"].get("food", 0))

    def test_villagers_lose_protection_at_eighty_percent(self) -> None:
        before = self.rates(own_1=39)["protected_rate_food"]
        after = self.rates(own_1=40)["protected_rate_food"]
        self.assertEqual(before - after, 50)

    def test_army_loses_protection_at_threshold(self) -> None:
        self.assertEqual(self.rates(n_military=20)["protected_rate_food"], 50)
        self.assertGreater(self.rates(n_military=19)["protected_rate_food"], 50)

    def test_both_thresholds_leave_no_unit_flow_protected(self) -> None:
        self.assertEqual(self.rates(own_1=40, n_military=20)["protected_rate_food"], 0)

    def test_unprotected_producers_still_contribute_total_demand(self) -> None:
        protected = self.rates(protected_producers_22=2)
        unprotected = self.rates(protected_producers_22=0)
        self.assertEqual(protected["rate_food"], unprotected["rate_food"])
        self.assertGreater(protected["protected_rate_food"], unprotected["protected_rate_food"])

    def test_one_producer_does_not_protect_both(self) -> None:
        one = self.rates(protected_producers_22=1)["protected_rate_food"]
        two = self.rates(protected_producers_22=2)["protected_rate_food"]
        self.assertEqual(two - one, test_definitions().units[101]["cost"].get("food", 0))
