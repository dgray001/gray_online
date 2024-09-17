package utils

import (
	"fmt"
	"os"
)

func ErrorCheckSoft(e error, msg string, a ...any) {
	if e == nil {
		return
	}
	a = append(a, e.Error())
	PrintError(msg+": %s", a...)
}

func ErrorCheck(e error, msg string, a ...any) {
	ErrorCheckSoft(e, msg, a...)
	if e != nil {
		panic("")
	}
}

func PrintError(msg string, a ...any) {
	s := fmt.Sprintf("Error: "+msg, a...)
	fmt.Fprintln(os.Stderr, s)
}
