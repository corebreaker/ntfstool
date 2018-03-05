//+build !windows

package core

import (
    "os"
    "syscall"
)

func DupFile(file *os.File) (*os.File, error) {
    fd, err := syscall.Dup(int(file.Fd()))
    if err != nil {
        return nil, WrapError(err)
    }

    return os.NewFile(uintptr(fd), file.Name()), nil
}
