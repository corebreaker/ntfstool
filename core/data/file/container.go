package file

import (
	"fmt"
	"os"

	"essai/ntfstool/core"
	"essai/ntfstool/core/data"
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

func (self *tDataContainer) GetIndexCount() int {
	return len(self.desc.Indexes)
}

func (self *tDataContainer) GetCounts() map[data.IDataRecord]int {
	res := make(map[data.IDataRecord]int)
	for _, k := range self.format.headers {
		res[k] = int(self.desc.Counts[k.GetEncodingCode()])
	}

	return res
}

func (self *tDataContainer) GetFormatName() string {
	return self.format.name
}

func (self *tDataContainer) Offsets() []int64 {
	res := make([]int64, len(self.desc.Indexes))

	for i, v := range self.desc.Indexes {
		res[i] = v.Physical
	}

	return res
}
