from __future__ import annotations

from common import Record, aggregate, branch, selection, store, variable


def center_in(name: str) -> Record:
    return aggregate("count", selection("spaces", dict(in_set="army_space", **{"if": f"inSet({name}, item)"})))


def cohort_refresh() -> list[Record]:
    safe = selection("spaces", {"if": "inSet(base, target) + inSet(retreat_goal, target) > 0"})
    arrived = selection("units", dict(owner="own", at=dict(in_set="retreat_safe_spaces"),
                                      **{"if": "inSet(retreat_cohort, target)"}))
    remaining = selection("units", dict(in_set="retreat_cohort", **{"if": "1 - inSet(retreat_arrived, target)"}))
    cohort = selection("units", dict(in_set="retreat_cohort"))
    center = selection("spaces", dict(near=dict(distance=0, of=cohort)),
                       let=dict(travel=aggregate("sum", cohort, "var(item.distance_to_target)")),
                       rank=["0 - var(target.travel)", "0 - var(target.x)", "0 - var(target.y)"], limit=1)
    return [store("retreat_safe_spaces", safe), store("retreat_arrived", arrived),
            store("retreat_cohort", remaining, True), store("retreat_center", center)]


def nearby_army(anchor: str, role: str = "item") -> Record:
    return selection("units", dict(owner="own", near=dict(
        distance="var(tune_retreat_cohesion_radius)", of=selection("spaces", dict(in_set=anchor))),
        **{"if": f"inSet(retreat_cohort, {role})"}))


def group_measures() -> list[Record]:
    return [variable("retreat_cohesion", aggregate("count", nearby_army("retreat_center"))),
            variable("retreat_cohesion", "var(retreat_cohesion) / max(1, var(n_retreat_cohort))"),
            variable("retreat_group_share", aggregate("count", nearby_army("retreat_group_goal"))),
            variable("retreat_group_share", "var(retreat_group_share) / max(1, var(n_retreat_cohort))")]


def micro_available(role: str) -> str:
    return f"(var(army_retreat) == 0) + (1 - inSet(retreat_cohort, {role})) + inSet(retreat_holding, {role}) > 0"


def regroup_state() -> list[Record]:
    trigger = "(var(army_retreat) > 0) * (var(retreat_regrouping) == 0) * ((var(retreat_cohesion) < var(tune_retreat_cohesion_share)) + (var(go) > 0) + (var(retreat_center_goal) > 0) > 0)"
    start = [store("retreat_group_goal", selection("spaces", dict(in_set="retreat_goal")), True),
             variable("retreat_regrouping", 1, True), variable("retreat_group_started", "gameVar(turn)", True)]
    holding = nearby_army("retreat_group_goal", "target")
    holding["where"]["if"] += " * (var(army_retreat) > 0) * (var(retreat_regrouping) > 0)"
    return [branch(trigger, start), store("retreat_destination", selection("spaces", dict(in_set="retreat_goal"))),
            branch("var(retreat_regrouping) > 0", [store("retreat_destination", selection("spaces", dict(in_set="retreat_group_goal")))]),
            store("retreat_holding", holding)]


def start_retreat(pick_escape: Record) -> list[Record]:
    return [store("escape_space", selection("spaces", {"if": "0"})), pick_escape,
            store("retreat_goal", selection("spaces", dict(in_set="escape_space")), True),
            branch("var(n_retreat_goal) > 0", [variable("army_retreat", 1, True),
                   store("retreat_cohort", selection("units", dict(owner="own", bucket="army")), True),
                   variable("retreat_started", "gameVar(turn)", True), variable("retreat_regrouping", 0, True)])]


def retreat_policy(pick_escape: Record) -> list[Record]:
    finished = "(var(army_retreat) > 0) * ((var(n_retreat_cohort) == 0) + (var(retreat_age) >= var(tune_retreat_timeout_turns)) + (var(retreat_goal_count) == 0) + ((var(retreat_regrouping) > 0) * (var(retreat_group_share) >= var(tune_retreat_cohesion_share)) * (gameVar(turn) - var(retreat_group_started) >= var(tune_retreat_recovery_turns))) > 0)"
    enter = "(var(army_retreat) == 0) * (var(a_outnumbered) > 0) * (var(retreat_center_base) == 0) * (var(retreat_blocked) == 0) * (var(retreat_finished) == 0) * (gameVar(turn) > var(retreat_defend_until))"
    return [*cohort_refresh(), *group_measures(), variable("retreat_center_base", center_in("base")),
            variable("retreat_center_goal", center_in("retreat_goal")),
            variable("retreat_goal_count", aggregate("count", selection("spaces", dict(in_set="retreat_goal")))),
            variable("retreat_age", "max(0, gameVar(turn) - var(retreat_started))"),
            variable("retreat_blocked", "var(retreat_blocked) * (var(a_outnumbered) > 0)", True),
            variable("retreat_finished", finished),
            branch("var(retreat_finished)", [variable("army_retreat", 0, True),
                   variable("retreat_regrouping", 0, True), variable("retreat_blocked", 1, True),
                   variable("retreat_defend_until", "gameVar(turn) + var(tune_retreat_recovery_turns)", True)]),
            branch(enter, start_retreat(pick_escape)), *cohort_refresh(), *group_measures(), *regroup_state()]
