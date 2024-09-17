package main

import (
	"bufio"
	"os"
	"strings"

	"fiddlesticks.live/parser"
	"fiddlesticks.live/utils"
)

func main() {
	if len(os.Args) < 2 {
		utils.PrintError("Must specify input file path")
		return
	}

	in_path := "inputs/" + os.Args[1] + ".yml"
	in_file, e := os.Open(in_path)
	utils.ErrorCheck(e, "Input file not found for path %s", in_path)
	scanner := bufio.NewScanner(in_file)
	scanner.Split(bufio.ScanLines)

	q := make([]parser.InputObject, 0, 5)
	input := parser.CreateInput()
	q = append(q, input)
	for scanner.Scan() {
		line_split := strings.SplitN(scanner.Text(), ":", 2)
		if len(line_split) < 2 {
			continue
		}
		k := strings.TrimSpace(line_split[0])
		if k == "End" {
			q = q[:len(q)-1]
			continue
		}
		q = q[len(q)-1].AddLine(q, k, strings.TrimSpace(line_split[1]))
	}
	input.Print()
	utils.ErrorCheck(input.Validate(), "Input validation error")
	utils.ErrorCheckSoft(in_file.Close(), "Failed to close input file")

	utils.ErrorCheckSoft(input.Run(), "Run errored")
}
