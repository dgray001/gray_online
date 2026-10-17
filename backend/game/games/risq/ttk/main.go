package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/dgray001/gray_online/game/games/risq"
)

func formatResult(r risq.TTKResult) string {
	if !r.Possible {
		return "-"
	}
	return fmt.Sprintf("%d", r.Turns)
}

type techCombo struct {
	label string
	techs []uint32
}

// per-unit applicable tech combos, keyed by unit_id
var unitTechCombos = map[uint32][]techCombo{
	1:   {{"none", nil}, {"loom", []uint32{1}}},
	101: {{"none", nil}, {"atk", []uint32{2}}, {"arm", []uint32{3}}, {"both", []uint32{2, 3}}},
	102: {{"none", nil}, {"atk", []uint32{2}}, {"arm", []uint32{3}}, {"both", []uint32{2, 3}}},
}

func printTechComboTable(attacker_id uint32, target_id uint32) {
	attacker_combos := unitTechCombos[attacker_id]
	target_combos := unitTechCombos[target_id]
	fmt.Printf("\n%s attacking %s (rows: attacker tech, cols: target tech)\n", risq.UnitName(attacker_id), risq.UnitName(target_id))
	fmt.Printf("%-6s", "")
	for _, c := range target_combos {
		fmt.Printf("%8s", c.label)
	}
	fmt.Println()
	for _, a := range attacker_combos {
		fmt.Printf("%-6s", a.label)
		for _, t := range target_combos {
			fmt.Printf("%8s", formatResult(risq.ComputeUnitVsUnitTTKWithTechs(attacker_id, target_id, a.techs, t.techs)))
		}
		fmt.Println()
	}
}

func main() {
	unitsOnly := flag.Bool("units", false, "Only show units in the target columns")
	barracksTechs := flag.Bool("barracks-techs", false, "Show tech-combo TTK tables for villager and the two barracks units (1, 101, 102)")
	unitsFile := flag.String("units-file", "", "Units json to use instead of ../config/units.json")
	efficiency := flag.Bool("efficiency", false, "Show cost-efficiency multiples for -matchups instead of the TTK table")
	techs := flag.String("techs", "2,3,5", "Comma-separated tech ids researched by both sides for -efficiency")
	matchups := flag.String("matchups", "101:102,102:103,103:101", "Comma-separated a:b unit id pairs for -efficiency")
	flag.Parse()
	if err := risq.LoadConfig("../config"); err != nil {
		fmt.Fprintln(os.Stderr, "loading risq config:", err)
		os.Exit(1)
	}
	if *unitsFile != "" {
		if err := risq.LoadUnitsFile(*unitsFile); err != nil {
			fmt.Fprintln(os.Stderr, "loading units file:", err)
			os.Exit(1)
		}
	}
	if *efficiency {
		tech_ids, err := parseIDs(*techs, ",")
		pairs, pairs_err := parseMatchups(*matchups)
		if err != nil || pairs_err != nil {
			fmt.Fprintln(os.Stderr, "parsing -techs or -matchups:", err, pairs_err)
			os.Exit(1)
		}
		printEfficiency(pairs, tech_ids)
		return
	}

	if *barracksTechs {
		ids := []uint32{1, 101, 102}
		for _, attacker_id := range ids {
			for _, target_id := range ids {
				printTechComboTable(attacker_id, target_id)
			}
		}
		return
	}

	unit_ids := risq.AllUnitIDs()
	building_ids := risq.AllBuildingIDs()

	headers := make([]string, 0, len(unit_ids)+len(building_ids))
	for _, id := range unit_ids {
		headers = append(headers, risq.UnitName(id))
	}
	if unitsOnly == nil || !*unitsOnly {
		for _, id := range building_ids {
			headers = append(headers, risq.BuildingName(id)+" (bldg)")
		}
	}

	col_width := 10
	for _, h := range headers {
		if len(h)+2 > col_width {
			col_width = len(h) + 2
		}
	}

	fmt.Printf("%-18s", "attacker \\ target")
	for _, h := range headers {
		fmt.Printf("%*s", col_width, h)
	}
	fmt.Println()
	fmt.Println(strings.Repeat("-", 18+col_width*len(headers)))

	for _, attacker_id := range unit_ids {
		fmt.Printf("%-18s", risq.UnitName(attacker_id))
		for _, target_id := range unit_ids {
			fmt.Printf("%*s", col_width, formatResult(risq.ComputeUnitVsUnitTTK(attacker_id, target_id)))
		}
		if unitsOnly == nil || !*unitsOnly {
			for _, target_id := range building_ids {
				fmt.Printf("%*s", col_width, formatResult(risq.ComputeUnitVsBuildingTTK(attacker_id, target_id)))
			}
		}
		fmt.Println()
	}
}
