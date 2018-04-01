package datafile

import (
	"fmt"
	"os"

	"essai/ntfstool/core"
	"essai/ntfstool/core/dataio"
)

type tDataContainer struct {
	desc   tFileDesc
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
	return int(self.desc.Count)
}

func (self *tDataContainer) GetCounts() map[dataio.IDataRecord]int {
	res := make(map[dataio.IDataRecord]int)
	for _, k := range self.format.headers {
		res[k] = int(self.desc.Counts[k.GetEncodingCode()])
	}

	return res
}

func (self *tDataContainer) GetFormatName() string {
	return self.format.name
}
