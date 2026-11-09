package experiment

import (
	"path/filepath"
	"runtime"
)

func manifest(options Options, training []Scenario, validation []Scenario, test []Scenario) error {
	source, err := sourceHash()
	if err != nil {
		return err
	}
	value := struct {
		Options                           Options
		Trials                            []Trial
		Training, Validation, Test, Arena []Scenario
		SourceSHA256, GoVersion           string
		Inputs, Outputs, Parameters       int
	}{options, trials(), training, validation, test, scenarios(options.TestGames, 44), source, runtime.Version(), 370, 79, 29309}
	return writeJSON(filepath.Join(options.Output, "manifest.json"), value)
}
