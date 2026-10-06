#!/usr/bin/env python3
import json
import sys
import argparse

def main():
    parser = argparse.ArgumentParser(description="Extract economy timeline from debug.log")
    parser.add_argument("log_file", help="Path to debug.log")
    parser.add_argument("--ai", default="1", help="AI player ID (default: 1)")
    parser.add_argument("--interval", type=int, default=1, help="Print every N turns (default: 1)")
    args = parser.parse_args()

    print(f"Turn | Vils | Mil | Pop/Lim | Food | Wood | w_food | w_wood")
    print("-" * 70)

    try:
        with open(args.log_file) as f:
            turn = 0
            for line in f:
                if f"ai ai-{args.ai}: new turn data {{" in line:
                    data = json.loads(line.split("data ", 1)[1])
                    vils = len([u for u in data.get("units", []) if u.get("Kind") == 0])
                    mil = len([u for u in data.get("units", []) if u.get("Kind") == 1])
                    pop = data.get("population", {})
                    curr = pop.get("current", 0)
                    limit = pop.get("limit", 0)
                    res = data.get("resources", {})
                    food = res.get("food", 0)
                    wood = res.get("wood", 0)
                    
                    # Next line usually has vars, but we can just store the turn info and wait for vars
                    turn_info = {"turn": turn, "vils": vils, "mil": mil, "curr": curr, "limit": limit, "food": food, "wood": wood}
                    turn += 1
                
                if f"ai ai-{args.ai}: vars {{" in line:
                    vars_data = json.loads(line.split("vars ", 1)[1])
                    w_food = vars_data.get("w_food", 0)
                    w_wood = vars_data.get("w_wood", 0)
                    
                    if (turn - 1) % args.interval == 0:
                        t = turn_info
                        print(f"{t['turn']:4} | {t['vils']:4} | {t['mil']:3} | {t['curr']:3}/{t['limit']:3} | {t['food']:4.0f} | {t['wood']:4.0f} | {w_food:6.1f} | {w_wood:6.1f}")
    except FileNotFoundError:
        print(f"File not found: {args.log_file}")

if __name__ == "__main__":
    main()
