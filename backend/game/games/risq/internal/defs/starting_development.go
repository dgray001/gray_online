package defs

type StartingDevelopment uint8

const (
	StartingDevelopment_NONE StartingDevelopment = iota
	StartingDevelopment_NOMAD
	StartingDevelopment_STANDARD
	StartingDevelopment_ESTABLISHED
	StartingDevelopment_ADVANCED
)
