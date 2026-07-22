package game

type GameType uint8

const (
	GameType_UNSPECIFIED GameType = iota
	GameType_FIDDLESTICKS
	GameType_EUCHRE
	GameType_EGYPTIAN_RAT_SLAP
	GameType_RISQ
	GameType_TEST_GAME
)
