from __future__ import annotations

from typing import Any, Union

from common import Record, aggregate, branch, power, selection, variable
from foundations import TRACKED
from locations import space_key

SITE_WEIGHTS = ("dist", "expand", "aggr", "safe", "spread", "vision", "region")

URGENCY_BY_TRACKER = {
    "vc": "var(vc_urgency)",
    "house": "var(house_urgency)",
    "smith": "var(smith_urgency)",
    "redoubt": "var(redoubt_urgency)",
}


def site_weights() -> list[Record]:
    actions: list[Record] = []
    for name in TRACKED:
        for weight in SITE_WEIGHTS:
            override = f"site_{name}_{weight}" if weight == "aggr" and name in ("barracks", "redoubt") else f"site_{weight}"
            actions.append(variable(f"site_{weight}_{name}", f"var(tune_{override})"))
    return actions


def away_from_bad(tracker: str) -> str:
    """Hexagonal dot-product check: returns > 0 when candidate target is oriented away from bad site relative to home."""
    x = f"(var({tracker}_bad_x) - var(home_x))"
    y = f"(var({tracker}_bad_y) - var(home_y))"
    a = f"(var(target.x) - var(home_x))"
    b = f"(var(target.y) - var(home_y))"
    return f"(({a}) * ({x}) + ({b}) * ({y}) + (({a}) + ({b})) * (({x}) + ({y})) < 0)"


def site_threat_lets() -> dict[str, Any]:
    enemy_units = selection("units", dict(
        owner="enemy", unit_types=["military"],
        **{"if": "distance(target, item) <= 1"}
    ))
    enemy_unidentified = selection("spaces", dict(in_set="poor_vision", **{"if": "distance(target, item) <= 1"}))
    enemy_towers = selection("buildings", dict(
        owner="enemy", building_ids=[1, 21, 23],
        **{"if": "distance(target, item) <= 1"}
    ))
    return dict(
        threat_units=aggregate("sum", enemy_units, power()),
        threat_unidentified=aggregate("sum", enemy_unidentified, "var(item.unit_count)"),
        threat_towers=aggregate("count", enemy_towers),
    )


def site_threat_expression() -> str:
    return ("(var(target.threat_units) + "
            "var(target.threat_unidentified) * var(unknown_unit_power) + "
            "3 * var(target.threat_towers))")


def site_score(tracker: str, urgency: str | None = None) -> str:
    if urgency is None:
        urgency = URGENCY_BY_TRACKER.get(tracker, "0")
    w = lambda name: f"var(site_{name}_{tracker})"
    score = (f"{w('dist')} * (0 - distance(home_space, target)) + "
             f"{w('expand')} * (1 - min(1, var(target.own_buildings))) + "
             f"{w('aggr')} * (var(enemy_distance) - var(target.enemy_dist)) + "
             f"{w('safe')} * (0 - {site_threat_expression()}) + "
             f"{w('spread')} * (0 - var(target.own_buildings)) + "
             f"{w('vision')} * (4 - var(target.space.vision)) + "
             f"{w('region')} * (var(target.region.index) >= 0) * (1 - min(1, var(target.own_buildings))) * (var(target.region.owned_spaces) + 1) / max(1, var(target.region.known_spaces))")
    if tracker != "vc":
        terrain_penalty = (f"var(tune_site_terrain) * max(0, 1 / max(0.01, var(target.build_speed)) - 1) * "
                           f"(1 + var(tune_site_urgency) * ({urgency}))")
        score += f" - {terrain_penalty}"
    return score


def pick_build_zone(tracker: str, building_id: int, urgency: str | None = None) -> Record:
    avoid_bad = f"(var({tracker}_abandoned) == 0) + (var(target.x) != var({tracker}_bad_x)) + (var(target.y) != var({tracker}_bad_y)) > 0"
    ban_check = f"(var({tracker}_ban) == 0) + ({away_from_bad(tracker)}) > 0"
    where_if = f"({avoid_bad}) * ({ban_check}) * (var(target.has_foundation) == 0)"
    if tracker == "outpost":
        where_if += " * (var(target.space.owner) == -1) * (var(target.space.gold_income) > 0)"

    enemy_dist = aggregate("min", selection("buildings", dict(owner="enemy", in_set="enemy_buildings")), "distance(target, item)")
    enemy_dist["min"]["default"] = -1
    build_speed = "var(target.space.build_speed_modifier)"
    has_foundation = aggregate("count", selection("foundations", {"if": "distance(target, item, 1) == 0"}))

    lets = dict(
        enemy_dist=enemy_dist,
        build_speed=build_speed,
        has_foundation=has_foundation,
        **site_threat_lets(),
    )

    ranking = [
        site_score(tracker, urgency),
        f"0 - {space_key('target')}",
        "0 - var(target.zone_x)",
        "0 - var(target.zone_y)",
    ]

    return selection("zones", dict(in_set="buildable_zones", building=False, resource=False, **{"if": where_if}),
                     let=lets, rank=ranking, limit=1)


def build_action(tracker: str, building_id: int, builders: Union[int, str], condition: str = "1",
                 actors_set: str = "available_builders") -> list[Record]:
    actors = selection("units", dict(in_set=actors_set, **{"in": dict(order_kind=["none", "gather"])}, **{"if": "1 - var(actor.is_garrisoned)"}),
                       rank=["var(actor.order_kind) == -1", "0 - var(actor.uid)"], limit=builders)
    pick = pick_build_zone(tracker, building_id)
    issue = [dict(build=dict(building_id=building_id))]

    select_action = dict(
        action="select",
        actors=actors,
        pick=pick,
        issue=issue,
    )

    wrapped = dict(
        action="for_each",
        select=selection("spaces", dict(in_set="home"), limit=1),
        **{"as": "home_space"},
        do=[select_action],
    )

    return [branch(condition, [wrapped])]
