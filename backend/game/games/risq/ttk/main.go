package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/dgray001/gray_online/game/games/risq"
)

func formatResult(r risq.TTKResult) string {
	if !r.Possible {
		return "-"
	}
	return fmt.Sprintf("%d", r.Turns)
}

func main() {
	unitsOnly := flag.Bool("units", false, "Only show units in the target columns")
	flag.Parse()

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
