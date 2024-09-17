package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 3 {
		printError("Must specify input and output file paths")
		return
	}

	in_path := "inputs/" + os.Args[1] + ".yml"
	in_file, e := os.Open(in_path)
	errorCheck(e, "Input file not found: %s", in_path)
	scanner := bufio.NewScanner(in_file)
	scanner.Split(bufio.ScanLines)
	q := []InputObject{}
	q = append(q, createInput())
	for scanner.Scan() {
		if len(q) < 1 {
			break
		}
		line_split := strings.SplitN(scanner.Text(), ":", 2)
		if len(line_split) < 2 {
			continue
		}
		q = q[0].addLine(q, line_split[0], line_split[1])
	}

	e = in_file.Close()
	errorCheckSoft(e, "Failed to close input file")
}

func errorCheckSoft(e error, msg string, a ...any) {
	if e == nil {
		return
	}
	printError(msg, a...)
}

func errorCheck(e error, msg string, a ...any) {
	if e == nil {
		return
	}
	printError(msg, a...)
	panic("")
}

func printError(msg string, a ...any) {
	s := fmt.Sprintf("Error: "+msg, a...)
	fmt.Fprintln(os.Stderr, s)
}
