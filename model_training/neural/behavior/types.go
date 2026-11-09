package behavior

type Options struct {
	Run    string
	Output string
	Corpus string
	Games  int
	Seed   int64
}
type Prediction struct {
	Probabilities []float64
	MeanBid       float64
	Choice        int
}
type Pair struct {
	A                   string
	B                   string
	BidCount            int
	CardCount           int
	BidMAE              float64
	BidTV               float64
	BidDisagreement     float64
	CardTV              float64
	CardDisagreement    float64
	ParameterRMS        float64
	ParameterComparable bool
}

type Summary struct {
	Trials     []string
	Games      int
	Situations int
	Pairs      []Pair
}
