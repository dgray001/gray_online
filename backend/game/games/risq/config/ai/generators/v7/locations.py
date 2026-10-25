from __future__ import annotations

from common import Record, aggregate, branch, coordinates, selection, store, variable


def space_key(role: str) -> str:
    x = f"(2 * max(var({role}.x), 0 - var({role}.x)) - (var({role}.x) < 0))"
    y = f"(2 * max(var({role}.y), 0 - var({role}.y)) - (var({role}.y) < 0))"
    return f"(({x} + {y}) * ({x} + {y} + 1) / 2 + {y})"


def home() -> list[Record]:
    building = selection("buildings", dict(owner="own"),
                         rank=["var(target.building_id) == 1", "0 - var(target.uid)"], limit=1)
    nearby = dict(distance=0, of=selection("buildings", dict(in_set="home_building")))
    fallback = selection("spaces", {"if": "var(target.own_units) > 0"})
    fallback["rank"] = ["var(target.owner) == var(player_id)", "var(target.own_units)", f"0 - {space_key('target')}"]
    fallback["limit"] = 1
    return [store("home_building", building), store("home", selection("spaces", dict(near=nearby))),
            branch("var(n_home_building) == 0", [store("home", fallback)]), *coordinates("home", "spaces")]


def base() -> list[Record]:
    workers = selection("units", dict(in_set="villagers", **{"if": "1 - var(target.is_garrisoned)"},
                                      **{"in": dict(order_kind=["gather", "build", "repair", "renew"])}))
    working = aggregate("count", selection("units", dict(in_set="working_villagers", **{
        "if": "(var(item.x) == var(target.x)) * (var(item.y) == var(target.y))"})))
    home_space = "(var(target.x) == var(home_x)) * (var(target.y) == var(home_y)) * var(n_home)"
    eligible = selection("spaces", {"if": f"(var(target.owner) == var(player_id)) + var(target.workers) + {home_space}"},
                         **{"let": dict(workers=working)})
    connected = dict(distance="var(board_spaces)", of=selection("spaces", dict(in_set="home")),
                     through=dict(in_set="base_eligible"))
    return [store("working_villagers", workers), store("base_eligible", eligible),
            store("base", selection("spaces", dict(near=connected)))]


def army() -> list[Record]:
    members = selection("units", dict(owner="own", bucket="army"))
    occupied = dict(distance=0, of=members)
    travel = aggregate("sum", members, "var(item.distance_to_target)")
    center = selection("spaces", dict(near=occupied), **{"let": dict(travel=travel)},
                       rank=["0 - var(target.travel)", f"0 - {space_key('target')}"], limit=1)
    representative = selection("units", dict(owner="own", bucket="army", near=dict(
        distance=0, of=selection("spaces", dict(in_set="army_space")))), rank="0 - var(target.uid)", limit=1)
    return [store("army_units", members), store("army_space", center),
            store("army_center", representative), *coordinates("army_center", "units")]


def enemy_proximity() -> list[Record]:
    distance = aggregate("min", selection("buildings", dict(in_set="enemy_buildings")), "distance(origin, item)")
    distance["min"]["default"] = -1
    far = "distance(origin, item) >= max(0, var(enemy_distance) - var(tune_unknown_base_radius))"
    near_base = aggregate("sum", selection("spaces", {"if": far}), "var(item.unidentified_count)")
    return [variable("enemy_distance", -1), variable("unknown_near_base", 0),
            dict(action="for_each", select=selection("spaces", dict(in_set="home")), **{"as": "origin"}, do=[
                variable("enemy_distance", distance),
                branch("var(n_enemy_buildings) > 0", [variable("unknown_near_base", near_base)])])]
