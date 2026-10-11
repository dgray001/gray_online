package defs

type MapResourceLevel uint8

const (
	MapResourceLevel_NONE MapResourceLevel = iota
	MapResourceLevel_SPARSE
	MapResourceLevel_STANDARD
	MapResourceLevel_RICH
	MapResourceLevel_ABUNDANT
)
