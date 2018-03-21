package datafile

import (
	"fmt"
	"os"

	"essai/ntfstool/core"
)

type tDataContainer struct {
	infos struct {
		count   int
		headers []int16
		indexes [][2]int64
	}

	format *tFileFormat
	file   *os.File
}

func (self *tDataContainer) get_pos() (int64, error) {
	res, err := self.file.Seek(0, os.SEEK_CUR)
	if err != nil {
		return 0, core.WrapError(err)
	}

	return res, nil
}

func (self *tDataContainer) check() error {
	if self.file == nil {
		return core.WrapError(fmt.Errorf("File closed"))
	}

	return nil
}

func (self *tDataContainer) GetCount() int {
	return self.infos.count
}

func (self *tDataContainer) GetFormatName() string {
	return self.format.name
}
