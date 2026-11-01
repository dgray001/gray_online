from __future__ import annotations

from common import Record, aggregate, selection, store
from locations import space_key


def hire_destinations() -> list[Record]:
    region_allows = ("(gameVar(mercenaries_need_region) == 0) + (var(target.region.owner) == var(player_id))"
                     " + (var(target.region.index) < 0) > 0")
    held = "(var(has_economy) > 0) + inSet(holder_spaces, target) > 0"
    legal = f"({region_allows}) * ({held})"
    toward_target = aggregate("min", selection("spaces", dict(in_set="army_target_space")), "distance(target, item)")
    toward_target["min"]["default"] = 999
    to_home = aggregate("min", selection("spaces", dict(in_set="home")), "distance(target, item)")
    to_home["min"]["default"] = 999
    ranking = ["inSet(enemy_mil_spaces, target)", "inSet(enemy_mil_near_spaces, target)",
               "0 - var(target.toward_target)", "0 - var(target.to_home)", f"0 - {space_key('target')}"]
    destinations = selection("spaces", dict(in_set="hireable_spaces"), let=dict(toward_target=toward_target, to_home=to_home),
                             rank=ranking, limit="var(tune_hire_spaces)")
    return [store("hireable_spaces", selection("spaces", dict(in_set="owned_spaces", **{"if": legal}))),
            store("hire_space", destinations),
            store("hire_zones", selection("zones", dict(at=dict(in_set="hire_space"))))]
