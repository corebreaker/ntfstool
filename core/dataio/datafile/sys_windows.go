//+build windows

package datafile

import (
	"os"
)

type tSysErr string

func (self tSysErr) Error() string { return self }

func DupFile(f *os.File) (*os.File, error) {
	res, err := os.Open(f.Name())
	if err != nil {
		return nil, WrapError(err)
	}

	return res, nil
}
