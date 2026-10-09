import json
import subprocess
import os

VARIANTS = {
    "v0.7a_aggro": {"assault_odds": 1.2, "power_odds": 0.95, "overwhelm": 1.5},
    "v0.7b_patient": {"assault_odds": 1.4, "retreat_margin": 0.1, "plan_turns": 4},
    "v0.7c_raider": {"raid_min": 2, "raid_pause": 10, "raid_frac": 0.4},
    "v0.7d_paranoid": {"est_decay": 0.8, "est_unid_w": 1.2, "est_interp": 0.5},
    "v0.7e_focus": {"hit_damage": 1.2, "kill_ticks": 2, "spread_queue": 5},
    "v0.7f_crush": {"overwhelm": 1.2, "overwhelm_global": 1, "skip_rest_margin": 0.2, "skip_stage_margin": 0.4},
    "v0.7g_boom": {"horizon": 12, "invest_share": 0.4, "vil_share": 0.55},
    "v0.7h_sticky": {"engaged_odds": 0.8, "retreat_margin": 0.15, "army_ab_entry": 1},
    "v0.7i_vision": {"scout_home": 0, "est_inspect": 1, "site_vision": 0.5},
    "v0.7j_mixed": {"assault_odds": 1.25, "raid_pause": 15, "overwhelm": 1.4, "retreat_margin": 0.08, "kill_ticks": 2},
    "v0.7k_slowest": {"est_decay": 0.99, "raid_pause": 30},
    "v0.7l_noflinch": {"army_int": 0, "army_ab": 0, "army_ab_entry": 0},
}

MAPS = [
    {"name": "sim_close", "map": "script:sim_close"},
    {"name": "sim_gold", "map": "script:sim_gold"},
    {"name": "sim_army", "map": "script:sim_army"},
    {"name": "sim_advanced", "map": "script:sim_advanced"},
    {"name": "sim_rich", "map": "script:sim_rich"},
    {"name": "sim_wide", "map": "script:sim_wide"}
]

print("Building AI variants...")
for name, params in VARIANTS.items():
    out_path = f"../config/ai/{name}.json"
    cmd = ["python3", "gen_v06.py", f"out={out_path}"]
    for k, v in params.items():
        cmd.append(f"{k}={v}")
    subprocess.run(cmd, check=True)
    
    eval_cfg = {
      "seed": 100000,
      "max_turns": 100,
      "games_per_bucket": 3,
      "swap_seats": True,
      "players": [{"ai_config": name}, {"ai_config": "v0.6"}],
      "buckets": MAPS
    }
    with open(f"inputs/eval_{name}.json", "w") as f:
        json.dump(eval_cfg, f, indent=2)

print("Running simulations...")
results = {}
for name in VARIANTS.keys():
    print(f"Running {name} vs v0.6...")
    subprocess.run(["go", "run", ".", "-quiet", f"eval_{name}"], check=True)
    
    # Parse results
    res_path = f"outputs/eval_{name}/results.json"
    wins = 0
    losses = 0
    draws = 0
    if os.path.exists(res_path):
        with open(res_path) as f:
            for line in f:
                if not line.strip(): continue
                game = json.loads(line)
                if game.get("Error"): continue
                
                try:
                    var_idx = game["Ais"].index(name)
                    var_outcome = game["Outcomes"][var_idx]
                    if var_outcome in ("winning", "ahead", "crush", "defeat"):
                        wins += 1
                    elif var_outcome in ("losing", "behind", "crushed", "defeated"):
                        losses += 1
                    else:
                        draws += 1
                except (ValueError, IndexError):
                    pass
    results[name] = {"wins": wins, "losses": losses, "draws": draws}
    print(f"  {name}: {wins} W - {losses} L - {draws} D")

print("\nFinal Results:")
for name, res in sorted(results.items(), key=lambda x: x[1]['wins'], reverse=True):
    print(f"{name}: {res['wins']} W - {res['losses']} L - {res['draws']} D")

