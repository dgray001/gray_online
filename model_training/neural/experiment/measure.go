package experiment

import "fiddlesticks.live/neural"

func measure(network *neural.Network, examples []neural.Example, method string) Metrics {
	m := Metrics{}
	for _, example := range examples {
		loss, kl, correct := predictionMetrics(network.Predict(example.Input, example.Legal), example, method)
		m.Count++
		m.Loss += loss
		m.TeacherKL += kl
		if correct {
			m.Accuracy++
		}
		if example.Legal[0] < 27 {
			m.BidCount++
			m.BidLoss += loss
		} else {
			m.CardCount++
			m.CardLoss += loss
		}
	}
	m.Loss /= float64(max(1, m.Count))
	m.TeacherKL /= float64(max(1, m.Count))
	m.Accuracy /= float64(max(1, m.Count))
	m.BidLoss /= float64(max(1, m.BidCount))
	m.CardLoss /= float64(max(1, m.CardCount))
	return m
}
