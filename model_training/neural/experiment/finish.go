package experiment

import "path/filepath"

func finish(options Options, results []Result, validation []Scenario) error {
	games, evaluations, err := evaluate(results, validation)
	if err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(options.Output, "validation-games.json"), games); err != nil {
		return err
	}
	ranking := regimens(results, evaluations)
	selected := champions(results, ranking)
	testGames, test, err := evaluate(selected, scenarios(options.TestGames, 44))
	if err != nil {
		return err
	}
	baseGames, base := baselines(scenarios(options.TestGames, 44))
	testGames, test = append(testGames, baseGames...), append(test, base...)
	if err := writeJSON(filepath.Join(options.Output, "test-games.json"), testGames); err != nil {
		return err
	}
	tournamentGames, matches, err := tournament(selected, options.TestGames)
	if err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(options.Output, "tournament-games.json"), tournamentGames); err != nil {
		return err
	}
	return saveSummary(options, results, ranking, evaluations, test, matches)
}
