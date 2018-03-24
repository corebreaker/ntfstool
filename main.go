package main

import (
	"fmt"
	"os"
	"runtime"

	ntfs "essai/ntfstool/core"
)

func work() error {
	if len(os.Args) <= 1 {
		return do_help(nil)
	}

	cpu_count := (runtime.NumCPU() + 1) / 2
	if cpu_count == 0 {
		cpu_count++
	}

	runtime.GOMAXPROCS(cpu_count)

	part := ntfs.GetPartition()
	fmt.Println(fmt.Sprintf("Part: [%s]", part))
	if len(os.Args) <= 2 {
		ntfs.PrintBoot(part)

		return nil
	}

	action_arg := &tActionArg{
		partition: part,
		_args:     ntfs.GetArgs(),
	}

	defer ntfs.DeferedCall(action_arg.Close)

	for _, action := range actions {
		ret, err := run_action(action, action_arg)
		if err != nil {
			return err
		}

		if ret {
			return nil
		}
	}

	return nil
}

func main() {
	ntfs.CheckedMain(work)
}
