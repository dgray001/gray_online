package defs

type StartBalance uint8

const (
	StartBalance_NONE StartBalance = iota
	StartBalance_RANDOM
	StartBalance_MIRRORED
)
