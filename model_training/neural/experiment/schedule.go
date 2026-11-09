package experiment

func Scenarios(count int, seed int64) []Scenario { return scenarios(count, seed) }

func SourceHash() (string, error) { return sourceHash() }
