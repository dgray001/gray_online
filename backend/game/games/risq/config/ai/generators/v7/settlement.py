from __future__ import annotations

from typing import Union

from common import Record, aggregate, branch, selection, variable
from locations import space_key
from sites import away_from_bad, site_threat_lets, site_threat_expression


def nomad_clustering() -> list[Record]:
    villagers = selection("units", dict(owner="own", unit_ids=[1]))
    sum_x = aggregate("sum", villagers, "var(item.x)")
    sum_y = aggregate("sum", villagers, "var(item.y)")

    dx = "(var(target.x) - var(centroid_x))"
    dy = "(var(target.y) - var(centroid_y))"
    dist_sq = f"(({dx}) * ({dx}) + ({dx}) * ({dy}) + ({dy}) * ({dy}))"

    space_pick = selection(
        "spaces",
        dict(passable=True),
        rank=[f"0 - ({dist_sq})", f"0 - {space_key('target')}"],
        limit=1,
    )

    actors = selection(
        "units",
        dict(
            owner="own",
            unit_ids=[1],
            **{"in": dict(order_kind=["none", "gather"])},
            **{"if": "1 - var(actor.is_garrisoned)"},
        ),
    )

    move_action = dict(
        action="select",
        actors=actors,
        pick=space_pick,
        issue=[dict(move=dict(to="space"))],
    )

    condition = "(var(complete_1) == 0) * (var(foundations_1) == 0) * (var(had_city) == 0)"
    return [branch(condition, [
        variable("villager_sum_x", sum_x),
        variable("villager_sum_y", sum_y),
        variable("centroid_x", "var(villager_sum_x) / max(1, var(own_1))"),
        variable("centroid_y", "var(villager_sum_y) / max(1, var(own_1))"),
        move_action,
    ])]


def vc_urgency_rules() -> list[Record]:
    return [
        variable("vc_age", "(var(vc_age) + 1) * (var(want_1) > 0) * (var(foundations_1) == 0)", True),
        variable("vc_urgency", "(var(want_1) > 0) * (max(0, 1 - var(own_1) / max(1, var(worker_goal))) + max(0, var(vc_age) - 1) / max(0.01, var(tune_vc_age_scale)))"),
        variable("vc_cost_limit", "var(tune_vc_site_limit) * var(vc_urgency)"),
    ]


def pick_vc_zone() -> Record:
    avoid_bad = "(var(vc_abandoned) == 0) + (var(target.x) != var(vc_bad_x)) + (var(target.y) != var(vc_bad_y)) > 0"
    ban_check = f"(var(vc_ban) == 0) + ({away_from_bad('vc')}) > 0"
    speed_check = "var(target.build_speed) > 0"

    food_first = aggregate("min", selection("resources", dict(category="food")), "distance(target, item)")
    food_first["min"]["default"] = 15

    food_second = aggregate(
        "min",
        selection("resources", dict(category="food", **{"if": "distance(target, item) > var(target.food_first)"})),
        "distance(target, item)",
    )
    food_second["min"]["default"] = 25

    lets = dict(
        build_speed=aggregate("max", selection("spaces", {"if": "distance(target, item) == 0"}), "var(item.build_speed_modifier)"),
        vc_walk_sum=aggregate("sum", selection("units", dict(owner="own", unit_ids=[1])), "distance(target, item)"),
        food_first=food_first,
        food_second=food_second,
        villagers_near=aggregate("count", selection("units", dict(owner="own", unit_ids=[1], **{"if": "distance(target, item) <= 1"}))),
        owned_near=aggregate("count", selection("spaces", dict(owner="own", **{"if": "distance(target, item) <= 1"}))),
        **site_threat_lets(),
    )

    vc_cost = (
        "var(tune_vc_site_walk) * max(0, var(target.vc_walk_sum) / max(1, var(own_1)) - var(vc_walk_min)) + "
        "var(tune_vc_site_food) * var(target.food_first) + "
        "var(tune_vc_site_food_second) * var(target.food_second) + "
        "var(tune_vc_site_build) * (1 / max(0.01, var(target.build_speed)) - 1) + "
        f"var(tune_vc_site_threat) * var(had_city) * {site_threat_expression()} - "
        "var(tune_vc_cluster) * var(target.villagers_near)"
    )

    cost_check = f"({vc_cost}) <= var(vc_cost_limit)"
    land_check = "(var(owned_land) == 0) + (var(target.owned_near) >= 1) > 0"
    where_if = f"({avoid_bad}) * ({ban_check}) * ({speed_check}) * ({cost_check}) * ({land_check})"

    ranking = [
        f"0 - ({vc_cost})",
        f"0 - {space_key('target')}",
        "0 - var(target.zone_x)",
        "0 - var(target.zone_y)",
    ]

    return selection(
        "zones",
        dict(building=False, resource=False, **{"if": where_if}),
        let=lets,
        rank=ranking,
        limit=1,
    )


def vc_placement(condition: str = "1", builders: Union[int, str] = 2, actors_set: str = "available_builders") -> list[Record]:
    actors = selection(
        "units",
        dict(in_set=actors_set, **{"in": dict(order_kind=["none", "gather"])}, **{"if": "1 - var(actor.is_garrisoned)"}),
        rank=["var(actor.order_kind) == -1", "0 - var(actor.uid)"],
        limit=builders,
    )

    pick = pick_vc_zone()
    issue = [dict(build=dict(building_id=1))]

    walk_sum = aggregate("sum", selection("units", dict(owner="own", unit_ids=[1])), "distance(vc_space, item)")
    min_space = selection("spaces", dict(passable=True, vision_at_least=1), let=dict(walk_sum=walk_sum), rank=["0 - var(vc_space.walk_sum)"], limit=1)
    find_min_walk = dict(action="for_each", select=min_space, **{"as": "vc_space"}, do=[
        variable("vc_walk_min", "var(vc_space.walk_sum) / max(1, var(own_1))"),
    ])

    select_action = dict(
        action="select",
        actors=actors,
        pick=pick,
        issue=issue,
    )

    return [branch(condition, [
        variable("owned_land", aggregate("count", selection("spaces", dict(owner="own")))),
        variable("vc_walk_min", 0),
        find_min_walk,
        select_action,
    ])]
