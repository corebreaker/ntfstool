package main

import (
	"fmt"
	"os"
	"os/signal"

	ntfs "github.com/corebreaker/ntfstool/core"
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

			fmt.Println("  -", ntfs.GetSource(x))
		}

		fmt.Println()

		ntfs.PrintError(_stk[0])

		os.Exit(1)
	}()
}

func addstep(msg string, a ...interface{}) {
	copy(_stk[1:], _stk[:])
	_stk[0] = ntfs.WrapError(fmt.Errorf(msg, a...))
}
