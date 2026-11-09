package behavior

import (
	"fiddlesticks.live/neural"
	"fmt"
	"os"
	"path/filepath"
)

func Run(options Options) error {
	if options.Games < 1 {
		return fmt.Errorf("games must be positive")
	}
	names, models, err := loadModels(options.Run)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(options.Output), 0755); err != nil {
		return err
	}
	if err := os.Mkdir(options.Output, 0755); err != nil {
		return err
	}
	set, err := corpus(options)
	if err != nil {
		return err
	}
	for _, model := range models {
		for _, example := range set.Situations {
			if len(example.Input) != model.InputCount() && !(model.InputCount() == neural.LegacyInputs && len(example.Input) == neural.Inputs) {
				return fmt.Errorf("corpus does not match checkpoint observation schema")
			}
		}
	}
	summary, profiles := analyze(names, models, set)
	if err := save(options, set, summary, profiles); err != nil {
		return err
	}
	fmt.Printf("compared %d checkpoints on %d situations: %s\n", len(names), len(set.Situations), filepath.Join(options.Output, "report.md"))
	return nil
}
