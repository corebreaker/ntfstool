//+build windows

package datafile

import (
	"os"

	"essai/ntfstool/core"
)

type tSysErr string

func (self tSysErr) Error() string { return self }

func DupFile(f *os.File) (*os.File, error) {
	res, err := os.Open(f.Name())
	if err != nil {
		return nil, core.WrapError(err)
	}

	return res, nil
}
