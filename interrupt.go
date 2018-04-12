package main

import (
	"fmt"
	"os"
	"os/signal"

	"essai/ntfstool/core"
)

var _stk [100]error

func init() {
	s := make(chan os.Signal)
	signal.Notify(s, os.Interrupt)

	go func() {
		<-s

		for _, x := range _stk {
			if x == nil {
				continue
			}

			fmt.Println("  -", core.GetSource(x))
		}

		fmt.Println()

		core.PrintError(_stk[0])

		os.Exit(1)
	}()
}

func addstep(msg string, a ...interface{}) {
	copy(_stk[1:], _stk[:])
	_stk[0] = core.WrapError(fmt.Errorf(msg, a...))
}
