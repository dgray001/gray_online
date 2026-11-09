package experiment

import (
	"fiddlesticks.live/neural"
)

func tournament(selected []Result, games int) ([]GameResult, []Evaluation, error) {
	models, names := []*neural.Network{}, []string{}
	for _, result := range selected {
		network, err := neural.Load(result.Model)
		if err != nil {
			return nil, nil, err
		}
		models, names = append(models, network), append(names, result.Trial.ID)
	}
	models, names = append(models, nil), append(names, "theory4")
	rows := tournamentGames(models, names, games)
	return rows, summarize(rows), nil
}
