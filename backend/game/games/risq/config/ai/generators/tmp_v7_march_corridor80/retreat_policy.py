from __future__ import annotations

from common import Record, aggregate, branch, selection, store, variable


def center_in(name: str) -> Record:
    return aggregate("count", selection("spaces", dict(in_set="army_space", **{"if": f"inSet({name}, item)"})))


def start_retreat(pick_escape: Record) -> list[Record]:
    return [store("escape_space", selection("spaces", {"if": "0"})), pick_escape,
            store("retreat_goal", selection("spaces", dict(in_set="escape_space")), True),
            branch("var(n_retreat_goal) > 0", [variable("army_retreat", 1, True),
                   variable("retreat_started", "gameVar(turn)", True), variable("retreat_regrouping", 0, True)])]


def retreat_policy(pick_escape: Record) -> list[Record]:
    finished = "(var(army_retreat) > 0) * ((var(retreat_center_base) > 0) + (var(retreat_center_goal) > 0) + (var(retreat_age) >= var(tune_retreat_timeout_turns)) + (var(retreat_goal_count) == 0) > 0)"
    enter = "(var(army_retreat) == 0) * (var(a_outnumbered) > 0) * (var(retreat_center_base) == 0) * (var(retreat_blocked) == 0) * (var(retreat_finished) == 0) * (gameVar(turn) > var(retreat_defend_until))"
    return [variable("retreat_center_base", center_in("base")),
            variable("retreat_center_goal", center_in("retreat_goal")),
            variable("retreat_goal_count", aggregate("count", selection("spaces", dict(in_set="retreat_goal")))),
            variable("retreat_age", "max(0, gameVar(turn) - var(retreat_started))"),
            variable("retreat_blocked", "var(retreat_blocked) * (var(a_outnumbered) > 0)", True),
            variable("retreat_finished", finished),
            branch("var(retreat_finished)", [variable("army_retreat", 0, True),
                   variable("retreat_regrouping", 0, True), variable("retreat_blocked", 1, True),
                   variable("retreat_defend_until", "gameVar(turn) + var(tune_retreat_recovery_turns)", True)]),
            branch(enter, start_retreat(pick_escape))]
