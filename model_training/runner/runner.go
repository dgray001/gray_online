package runner

import (
	"fmt"
	"time"
)

func (i *Input) Run() error {
	time_start := time.Now().UTC()
	fmt.Println("")
	fmt.Println("Begin Run")
	for _, g := range i.games {
		e := g.run()
		if e != nil {
			return e
		}
	}
	for _, b := range i.benchmarks {
		e := b.run()
		if e != nil {
			return e
		}
	}
	for _, b := range i.standard_benchmarks {
		e := b.run()
		if e != nil {
			return e
		}
	}
	for _, g := range i.genetic_algorithms {
		e := g.run()
		if e != nil {
			return e
		}
	}
	for _, p := range i.parameter_testers {
		e := p.run()
		if e != nil {
			return e
		}
	}
	time_dif := time.Now().UTC().Sub(time_start)
	fmt.Println("\nEnd Run")
	fmt.Println("Total Time:", time_dif.String())
	return nil
}
