//+build windows

package core

import (
    "os"
    //"syscall"
)

type tSysErr string

func (self tSysErr) Error() string { return self }

func DupFile(f *os.File) (*os.File, error) {
    //syscall.DuplicateHandle()
    return nil, WrapError(tSysErr("DupFile not implemented"))
}
