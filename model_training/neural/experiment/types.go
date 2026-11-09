package experiment

type Options struct {
	Output          string
	TrainGames      int
	ValidationGames int
	TestGames       int
	MaxEpochs       int
	Patience        int
	Workers         int
}

type Trial struct {
	ID     string
	Method string
	Start  string
	Rate   float64
	Seed   int64
}
