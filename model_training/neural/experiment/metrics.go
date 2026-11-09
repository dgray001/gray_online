package experiment

type Metrics struct {
	Count     int
	Loss      float64
	TeacherKL float64
	Accuracy  float64
	BidCount  int
	BidLoss   float64
	CardCount int
	CardLoss  float64
}

type Epoch struct {
	Number         int
	TrainLoss      float64
	ValidationLoss float64
}
type Result struct {
	Trial      Trial
	BestEpoch  int
	Epochs     int
	Stop       string
	Validation Metrics
	Test       Metrics
	Curve      []Epoch
	Model      string
}

type Evaluation struct {
	Agent          string
	Group          string
	Games          int
	WinShare       float64
	WinAboveChance float64
	Score          float64
	Margin         float64
}
