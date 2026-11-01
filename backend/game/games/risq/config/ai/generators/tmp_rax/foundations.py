from __future__ import annotations

from common import Definitions, Record, aggregate, branch, selection, variable
from locations import space_key

TRACKED: dict[str, tuple[int, str]] = {
    "house": (2, "var(tune_house_tries)"),
    "vc": (1, "var(tune_building_tries)"),
    "farm": (3, "var(tune_building_tries)"),
    "barracks": (22, "var(tune_building_tries)"),
    "smith": (11, "var(tune_building_tries)"),
    "redoubt": (23, "var(tune_building_tries)"),
    "outpost": (21, "var(tune_building_tries)"),
}


def foundation_observations() -> list[Record]:
    actions: list[Record] = []
    for name, (building, _) in TRACKED.items():
        with_builders = aggregate("count", selection("foundations", dict(
            building_ids=[building], **{"if": "var(item.builders) > 0"})))
        without_builders = aggregate("count", selection("foundations", dict(
            building_ids=[building], **{"if": "var(item.builders) == 0"})))
        actions.extend([
            variable(f"{name}_exist", f"var(foundations_{building})"),
            variable(f"{name}_with_builders", with_builders),
            variable(f"{name}_without_builders", without_builders),
            variable(f"{name}_abandoned", f"min(var({name}_abandoned), var({name}_exist))", True),
            variable(f"{name}_replacement", f"max(0, var({name}_exist) - var({name}_abandoned))"),
            variable(f"{name}_resend_wait",
                     f"(var({name}_resend_wait) + 1) * (var({name}_abandoned) > 0) * (var({name}_replacement) == 0)", True),
            variable(f"{name}_lost",
                     f"max(0, min(var({name}_exist), var({name}_with_post)) - var({name}_with_builders))"),
            variable(f"{name}_tries",
                     f"(var({name}_tries) + var({name}_lost)) * (var({name}_replacement) > 0)", True),
            variable(f"{name}_ban_wait",
                     f"(var({name}_ban_wait) + 1) * (var({name}_ban) > 0) * (var({name}_replacement) == 0)", True),
            variable(f"{name}_ban",
                     f"(var({name}_ban) > 0) * (var({name}_replacement) == 0) * (var({name}_ban_wait) < var(tune_ban_wait))", True),
            variable(f"{name}_threat_count", aggregate("count", selection("units", dict(
                owner="enemy", unit_types=["military"], near=dict(distance=1, of=dict(x=f"var({name}_bad_x)", y=f"var({name}_bad_y)")))))),
        ])
    return actions


def foundation_failures() -> list[Record]:
    actions: list[Record] = []
    for name, (building, limit) in TRACKED.items():
        giveup = f"var({name}_tries) >= {limit}"
        cancel_where = dict(building_ids=[building], **{
            "if": f"(var(target.x) == var({name}_bad_x)) * (var(target.y) == var({name}_bad_y)) * var(target.is_planned) * (1 - var(target.ordered_this_turn))"
        })
        cancel = dict(action="select", pick=selection("foundations", cancel_where, limit=1),
                      issue=[dict(cancel_foundation={})])
        record_coords = dict(
            action="for_each",
            select=selection(
                "foundations",
                dict(building_ids=[building], **{"if": "var(failed.builders) == 0"}),
                rank=["var(failed.ordered_this_turn)", f"0 - {space_key('failed')}", "0 - var(failed.zone_x)", "0 - var(failed.zone_y)"],
                limit=1,
            ),
            **{"as": "failed"},
            do=[
                variable(f"{name}_bad_x", "var(failed.x)", True),
                variable(f"{name}_bad_y", "var(failed.y)", True),
            ],
        )
        on_giveup = [
            record_coords,
            variable(f"{name}_ban", 1, True),
            variable(f"{name}_ban_wait", 0, True),
            variable(f"{name}_abandoned", f"var({name}_abandoned) + 1", True),
            variable(f"{name}_tries", 0, True),
            cancel,
        ]
        actions.extend([
            variable(f"{name}_with_post", f"var({name}_with_builders)", True),
            branch(giveup, on_giveup),
        ])
    return actions


def site_clear(name: str) -> str:
    return f"(var({name}_threat_count) == 0)"
