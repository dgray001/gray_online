from __future__ import annotations

import unittest

from common import variable
from estimates import strategy
from retreat_policy import retreat_policy
from test_gold import evaluate


def finish_value(initial: dict[str, float]) -> float:
    action = next(a for a in retreat_policy({}) if a.get("name") == "retreat_finished")
    defaults = dict(army_retreat=1, retreat_goal_count=1, tune_retreat_timeout_turns=3)
    return evaluate([action], defaults | initial)["retreat_finished"]


def enter_value(initial: dict[str, float]) -> float:
    expression = retreat_policy({})[-1]["when"]["value_at_least"]["value"]
    return evaluate([variable("entry", expression)], dict(game_turn=10, a_outnumbered=1) | initial)["entry"]


class RetreatRegressionTests(unittest.TestCase):
    def test_leaving_enemy_range_does_not_count_as_kills(self) -> None:
        actions = [a for a in strategy() if a.get("name") in ("enemy_losses", "enemy_growth", "enemy_army_memory")]
        initial = dict(enemy_army_memory=10, previous_enemy_local=15, army_center_enemy_power=0,
                       previous_engaged=1, n_army_units=10, tune_memory_loss_weight=1,
                       tune_memory_retention=0.98, tune_memory_population_power=2, game_population_max=100)
        result = evaluate(actions, initial)
        self.assertEqual(result["enemy_losses"], 0)
        self.assertAlmostEqual(result["enemy_army_memory"], 9.8)

    def test_retreat_ends_at_base_goal_timeout_or_missing_goal(self) -> None:
        for initial in (dict(retreat_center_base=1), dict(retreat_center_goal=1),
                        dict(retreat_age=3), dict(retreat_goal_count=0)):
            with self.subTest(initial=initial):
                self.assertEqual(finish_value(initial), 1)

    def test_retreat_continues_before_arrival_and_timeout(self) -> None:
        self.assertEqual(finish_value(dict(retreat_age=2)), 0)

    def test_no_retargeting_or_immediate_restart(self) -> None:
        for initial in (dict(army_retreat=1), dict(retreat_center_base=1), dict(retreat_blocked=1),
                        dict(retreat_finished=1), dict(retreat_defend_until=10)):
            with self.subTest(initial=initial):
                self.assertEqual(enter_value(initial), 0)

    def test_fresh_threat_can_start_after_recovery(self) -> None:
        self.assertEqual(enter_value(dict(retreat_defend_until=9)), 1)
