//+build !windows

package datafile

import (
	"os"
	"syscall"

	"essai/ntfstool/core"
)

func DupFile(file *os.File) (*os.File, error) {
	fd, err := syscall.Dup(int(file.Fd()))
	if err != nil {
		return nil, core.WrapError(err)
	}

	return os.NewFile(uintptr(fd), file.Name()), nil
}
