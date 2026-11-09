package rl

type Options struct {
	Output           string
	GamesPerEpoch    int
	EvaluationGames  int
	TestGames        int
	MaxEpochs        int
	Patience         int
	Workers          int
	Sweep            bool
	Seed             int64
	Rate             float64
	Entropy          float64
	Start            string
	Hidden           int
	CoreShare        float64
	LinearControls   bool
	Reference        string
	Widths           []int
	ReferenceHidden4 string
}

type Trial struct {
	Inputs  int
	Hidden  int
	ID      string
	Seed    int64
	Rate    float64
	Entropy float64
	Start   string
}
