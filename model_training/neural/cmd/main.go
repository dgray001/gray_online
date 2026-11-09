package main

import (
	"fiddlesticks.live/neural/rl"
	"fmt"
	"os"
)

func main() {
	if err := rl.Command(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
