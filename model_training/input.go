package main

import "strconv"

type InputObject interface {
	addLine(q []InputObject, k string, v string) []InputObject
}

type inputStatic struct{}

var InputStatic inputStatic

func (inputStatic) parseInt(s string) int {
	i, e := strconv.Atoi(s)
	if e != nil {
		return 0
	}
	return i
}

type Input struct{}

func createInput() Input {
	return Input{}
}

func (i Input) addLine(q []InputObject, k string, v string) []InputObject {
	switch k {
	default:
		printError("Unknown key for Input: %s", k)
	}
	return q
}

type InputGame struct {
	min_round    uint8
	max_round    uint8
	trick_points uint8
	round_points uint8
}

func (i InputGame) addLine(q []InputObject, k string, v string) []InputObject {
	switch k {
	case "min_round":
		i.min_round = uint8(InputStatic.parseInt(v))
	case "max_round":
		i.max_round = uint8(InputStatic.parseInt(v))
	case "trick_points":
		i.trick_points = uint8(InputStatic.parseInt(v))
	case "round_points":
		i.round_points = uint8(InputStatic.parseInt(v))
	default:
		printError("Unknown key for InputGame: %s", k)
	}
	return q
}
