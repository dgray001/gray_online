package goldset

import "math/rand"

type reservoir struct {
	Items  []Item
	Seen   int
	Limit  int
	Random *rand.Rand
}

func (r *reservoir) next() int {
	if r.Limit == 0 {
		return -1
	}
	r.Seen++
	if len(r.Items) < r.Limit {
		r.Items = append(r.Items, Item{})
		return len(r.Items) - 1
	}
	index := r.Random.Intn(r.Seen)
	if index >= r.Limit {
		return -1
	}
	return index
}
