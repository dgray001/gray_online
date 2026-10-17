package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/dgray001/gray_online/game/games/risq"
)

func parseIDs(s string, separator string) ([]uint32, error) {
	ids := []uint32{}
	for _, part := range strings.Split(s, separator) {
		if strings.TrimSpace(part) == "" {
			continue
		}
		id, err := strconv.ParseUint(strings.TrimSpace(part), 10, 32)
		if err != nil {
			return nil, err
		}
		ids = append(ids, uint32(id))
	}
	return ids, nil
}

// Matchups are written "a:b,c:d"; each pair reports the multiple of a over b
func parseMatchups(s string) ([][2]uint32, error) {
	matchups := [][2]uint32{}
	for _, pair := range strings.Split(s, ",") {
		ids, err := parseIDs(pair, ":")
		if err != nil || len(ids) != 2 {
			return nil, fmt.Errorf("matchup %q must look like 11:12", pair)
		}
		matchups = append(matchups, [2]uint32{ids[0], ids[1]})
	}
	return matchups, nil
}

func printEfficiency(matchups [][2]uint32, techs []uint32) {
	fmt.Printf("%-28s %10s %10s %7s %7s %9s\n", "matchup (a > b)", "a kills/t", "b kills/t", "a cost", "b cost", "multiple")
	for _, m := range matchups {
		a, b := m[0], m[1]
		multiple, ok := risq.ComputeCostEfficiencyMultiple(a, b, techs, techs)
		label := fmt.Sprintf("%s > %s", risq.UnitName(a), risq.UnitName(b))
		a_rate := risq.ComputeUnitKillRate(a, b, techs, techs)
		b_rate := risq.ComputeUnitKillRate(b, a, techs, techs)
		if !ok {
			fmt.Printf("%-28s %10.3f %10.3f %7.0f %7.0f %9s\n", label, a_rate, b_rate, risq.UnitCost(a), risq.UnitCost(b), "-")
			continue
		}
		fmt.Printf("%-28s %10.3f %10.3f %7.0f %7.0f %9.2f\n", label, a_rate, b_rate, risq.UnitCost(a), risq.UnitCost(b), multiple)
	}
}
