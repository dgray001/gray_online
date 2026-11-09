package rl

import (
	"fmt"
	"strconv"
	"strings"
)

func parseWidths(value string) ([]int, error) {
	if value == "" {
		return nil, nil
	}
	result, seen := []int{}, map[int]bool{}
	for _, item := range strings.Split(value, ",") {
		width, err := strconv.Atoi(strings.TrimSpace(item))
		if err != nil || width < 1 || seen[width] {
			return nil, fmt.Errorf("widths must be distinct positive integers")
		}
		seen[width] = true
		result = append(result, width)
	}
	return result, nil
}
