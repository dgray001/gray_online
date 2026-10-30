from __future__ import annotations

from common import Record, aggregate, branch, selection, store, variable
from locations import space_key


def scouting_state() -> list[Record]:
    """Computes 3-phase scouting state, fog-age threshold, and allocates the scout bucket."""
    unexplored_local = aggregate("count", selection("spaces", dict(vision=0, near=dict(distance=4, of=selection("spaces", dict(in_set="home"))))))
    count_enemies = dict(action="for_each", source="players", where=dict(owner="enemy"), **{"as": "p"}, do=[
        variable("enemies_found", "var(enemies_found) + (var(p.buildings_known) > 0)"),
    ])
    fog_turns = "(var(explored_spaces) >= 0.9 * var(board_spaces)) * (110 - 100 * var(explored_spaces) / max(1, var(board_spaces))) - (var(explored_spaces) < 0.9 * var(board_spaces))"
    n_inf = "(var(own_101) + var(own_102) + var(own_103))"
    n_mil = "(var(own_101) + var(own_102) + var(own_103) + var(own_401))"
    p1 = f"({n_inf} >= 1) * (var(unexplored_local) >= 1)"
    p2 = f"(var(scout_p1) == 0) * (var(enemies_found) <= gameVar(num_players) - 2)"
    p3 = f"(var(scout_p1) == 0) * (var(scout_p2) == 0) * (var(unexplored_total) >= 1) * (((var(has_economy) >= 1) * (var(capacity_stone) <= 0)) + ({n_mil} >= 21) > 0)"
    size = f"var(scout_p1) * 1 + (var(scout_p2) + var(scout_p3)) * (1 + ({n_mil} >= 40))"
    pick_scout = selection("units", dict(owner="own", unit_ids=[101, 102, 103]),
                           let=dict(in_scout="inBucket(scout, target)"),
                           rank=["var(target.in_scout)", "0 - var(target.uid)"], limit="var(scout_size)")
    scouts_near = aggregate("count", selection("units", dict(owner="own", bucket="scout", near=dict(distance=1, of=selection("spaces", dict(in_set="home"))))))
    release_cond = f"(var(scouts_near) >= var(n_scout_units)) + ((var(enemy_near_home) >= 1) * ({n_mil} <= 3)) > 0"
    retire_scout = [
        dict(action="select", actors=selection("units", dict(owner="own", bucket="scout", **{"in": dict(order_kind=["none"])})),
             pick=selection("spaces", dict(in_set="home"), limit=1), issue=[dict(move={})]),
        variable("scouts_near", scouts_near),
        branch(release_cond, [dict(action="select", pick=selection("units", dict(owner="own", bucket="scout")), into=dict(bucket="scout", mode="remove"))]),
    ]
    return [
        variable("unexplored_local", unexplored_local), variable("unexplored_total", "var(board_spaces) - var(explored_spaces)"),
        variable("enemies_found", 0), count_enemies, variable("scout_fog_turns", fog_turns),
        variable("scout_p1", p1), variable("scout_p2", p2), variable("scout_p3", p3),
        variable("scout_phase", "var(scout_p1) * 1 + var(scout_p2) * 2 + var(scout_p3) * 3"), variable("scout_size", size),
        branch("var(scout_phase) > 0", [dict(action="select", pick=pick_scout, into=dict(bucket="scout", mode="replace"))], otherwise=retire_scout),
        store("scout_units", selection("units", dict(owner="own", bucket="scout")), persist=True),
    ]


def scouting_tactics() -> list[Record]:
    """Issues 3-phase scout movement: local unexplored, enemy-ring + fog-age search, and general unexplored."""
    scouts = selection("units", dict(owner="own", bucket="scout"))
    idle_scouts = selection("units", dict(owner="own", bucket="scout", **{"in": dict(order_kind=["none"])}))
    behavior = dict(action="select", actors=scouts, issue=[dict(set_behavior=dict(stance="passive", attack_back=False, interrupt_current=False))])
    unexplored = selection("spaces", dict(vision=0), rank=["0 - distance(home_origin, target)", f"0 - {space_key('target')}"] , limit=25, capacity="1")
    phase1 = dict(action="for_each", select=selection("spaces", dict(in_set="home"), limit=1), **{"as": "home_origin"}, do=[
        dict(action="select", actors=idle_scouts, pick=unexplored,
             match=dict(score=["0 - distance(home_origin, target)", "0 - distance(actor, target)"]), issue=[dict(move={})]),
    ])
    hex_r = "max(max(var(target.x), 0 - var(target.x)), max(var(target.y), 0 - var(target.y)), max(var(target.x) + var(target.y), 0 - var(target.x) - var(target.y)))"
    qual_p2 = "(var(target.vision) == 0) + (var(target.vision) == 1) * (var(scout_fog_turns) >= 0) * (var(target.turns_since_seen) >= var(scout_fog_turns)) > 0"
    p2_spaces = selection("spaces", dict(vision_at_most=1, **{"if": qual_p2}), let=dict(r=hex_r),
                          rank=["var(target.r) >= max(0, gameVar(map_size) - 2)", "0 - var(target.r)", f"0 - {space_key('target')}"] , limit=25, capacity="1")
    phase2 = dict(action="select", actors=idle_scouts, pick=p2_spaces,
                  match=dict(score=["var(target.r) >= max(0, gameVar(map_size) - 2)", "0 - var(target.r)", "0 - distance(actor, target)"]), issue=[dict(move={})])
    p3_spaces = selection("spaces", dict(vision=0), rank=[f"0 - {space_key('target')}"] , limit=25, capacity="1")
    phase3 = dict(action="select", actors=idle_scouts, pick=p3_spaces,
                  match=dict(score="0 - distance(actor, target)"), issue=[dict(move={})])
    return [behavior, branch("var(scout_phase) == 1", [phase1]),
            branch("var(scout_phase) == 2", [phase2]),
            branch("var(scout_phase) == 3", [phase3])]


def home_inspector() -> list[Record]:
    """Steps 1 unit to the facing edge zone to identify unknown units near home when safe."""
    insp_spaces = selection("spaces", dict(
        near=dict(distance="var(tune_est_inspect_reach)", of=selection("spaces", dict(in_set="home"))),
        **{"if": "(max(0, var(target.unit_count)) >= var(tune_est_inspect_min)) * (var(target.e_mil_near) * var(tune_est_insp_safe) == 0)"}
    ), let=dict(e_mil_near=aggregate("count", selection("units", dict(owner="enemy", unit_types=["military"], **{"if": "distance(target, item) <= 1"})))),
       rank=["max(0, var(target.unit_count))", f"0 - {space_key('target')}"] , limit=1)
    insp_zones = selection("zones", dict(
        near=dict(distance=1, of=selection("spaces", dict(in_set="insp_target"))),
        **{"if": "(distance(unid_space, target) == 1) * (var(target.space.owner) == var(player_id))"}
    ), rank=["0 - distance(unid_space, target, 1)", f"0 - {space_key('target')}"] , limit=1)
    pick_insp = dict(action="for_each", select=selection("spaces", dict(in_set="home"), limit=1), **{"as": "home_origin"}, do=[
        store("insp_target", insp_spaces),
        dict(action="for_each", select=selection("spaces", dict(in_set="insp_target"), limit=1), **{"as": "unid_space"}, do=[
            store("insp_zone", insp_zones),
        ]),
    ])
    mil_inspector = selection("units", dict(
        owner="own", unit_types=["military"], not_in_set="raiding_party",
        **{"in": dict(order_kind=["none", "move"]), "if": "var(actor.is_scout) == 0"}
    ), let=dict(is_scout="inSet(scout_units, actor)"),
       rank=["var(actor.idle)", "0 - var(actor.uid)"], limit=1)
    zone_pick = selection("zones", dict(in_set="insp_zone"), limit=1)
    send_mil = dict(action="select", actors=mil_inspector, pick=zone_pick, issue=[dict(move=dict(to="zone"))])
    return [
        variable("insp_cool", "max(0, var(insp_cool) - 1)", persist=True),
        store("insp_target", selection("spaces", {"if": "0"})),
        store("insp_zone", selection("zones", {"if": "0"})),
        pick_insp,
        branch("(var(n_insp_zone) > 0) * (var(insp_cool) <= 0)", [send_mil, variable("insp_cool", "var(tune_est_insp_cool)", persist=True)]),
    ]
