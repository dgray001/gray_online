package main

import (
	"fiddlesticks.live/goldset"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	port := flag.Int("port", 7077, "localhost port")
	foundation := flag.Bool("add-foundation", false, "add the curated 30-item foundation batch and exit")
	data := flag.String("data", "goldset/data", "goldset directory")
	web := flag.String("web", "goldset/web", "frontend directory")
	images := flag.String("images", "../backend/static/images", "card images directory")
	flag.Parse()
	store, err := goldset.OpenStore(*data)
	if err != nil {
		return err
	}
	if *foundation {
		count, err := goldset.AddFoundationBatch(store)
		if err != nil {
			return err
		}
		fmt.Printf("Added %d curated foundation situations.\n", count)
		return nil
	}
	if len(store.Queue()) == 0 {
		items, err := goldset.Generate(goldset.GenerateRequest{Count: 20, Players: 0, MinHand: 1, MaxHand: 3, Kind: "both", RoundPoints: 10, TrickPoints: 1, Seed: 20261109})
		if err != nil {
			return err
		}
		if _, err := store.Add(items); err != nil {
			return err
		}
	}
	address := fmt.Sprintf("127.0.0.1:%d", *port)
	fmt.Printf("Goldset: http://localhost:%d\nData: %s\n", *port, *data)
	server := &http.Server{Addr: address, Handler: goldset.Handler(store, *web, *images), ReadHeaderTimeout: 5 * time.Second}
	return server.ListenAndServe()
}
