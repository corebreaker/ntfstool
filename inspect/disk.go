package inspect

import (
	"github.com/corebreaker/ntfstool/core"
	"github.com/corebreaker/ntfstool/core/data"
)

type NtfsDisk struct {
	disk      *core.DiskIO
	mft_rl    core.RunList
	mft_shift int64
}

func (self *NtfsDisk) fill_runlist() error {
	var mft core.FileRecord

	if err := self.disk.ReadStruct((self.mft_shift+511)/512, &mft); err != nil {
		return err
	}

	if mft.Type != core.RECTYP_FILE {
		return nil
	}

	data_attrs := mft.GetAttributeFilteredList(core.ATTR_DATA)
	if (data_attrs == nil) || (len(data_attrs) == 0) {
		return nil
	}

	data_attr, err := mft.MakeAttributeFromOffset(data_attrs[0])
	if err != nil {
		return err
	}

	self.mft_rl = data_attr.GetRunList()

	return nil
}

func (self *NtfsDisk) get_file_sector(index int64) int64 {
	if index == 0 {
		return (self.mft_shift + 511) / 512
	}

	if self.mft_rl != nil {
		start := int64(0)
		for _, run := range self.mft_rl {
			end := start + (run.Count * 4)

			if (start <= index) && (index < end) {
				return (int64(run.Start) * 8) + ((index - start) * 2)
			}

			start = end
		}
	}

	return index * 2
}

func (self *NtfsDisk) FindIndex(position int64) data.FileIndex {
	fpos := data.FileIndex((position + 1023) / 1024)
	if self.mft_rl == nil {
		return fpos
	}

	vidx := data.FileIndex(0)
	for _, run := range self.mft_rl {
		start, end := data.FileIndex(run.Start)*4, data.FileIndex(run.GetNext())*4

		if (start <= fpos) && (fpos < end) {
			return vidx + (fpos - start)
		}

		vidx += data.FileIndex(run.Count) * 4
	}

	return data.FileIndex(0)
}

func (self *NtfsDisk) GetDisk() *core.DiskIO {
	return self.disk.Shift(0)
}

func (self *NtfsDisk) SetStart(start int64) error {
	self.disk.SetOffset(start)

	return self.fill_runlist()
}

func (self *NtfsDisk) SetMftShift(shift int64) error {
	self.mft_shift = shift

	return self.fill_runlist()
}

func (self *NtfsDisk) ReadRecordHeader(index int64, header *core.RecordHeader) error {
	return self.disk.ReadStruct(((self.mft_shift+511)/512)+(index*2), header)
}

func (self *NtfsDisk) ReadRecordHeaderFromRef(ref data.FileRef, header *core.RecordHeader) error {
	return self.ReadRecordHeader(int64(ref.GetFileIndex()), header)
}

func (self *NtfsDisk) ReadFileRecord(index int64, record *core.FileRecord) error {
	return self.disk.ReadStruct(self.get_file_sector(index), record)
}

func (self *NtfsDisk) ReadFileRecordFromRef(ref data.FileRef, record *core.FileRecord) error {
	return self.ReadFileRecord(int64(ref.GetFileIndex()), record)
}

func (self *NtfsDisk) GetAttributeValue(desc *core.AttributeDesc, read bool) (*core.AttributeValue, error) {
	if read {
		return desc.GetValue(self.disk)
	}

	return desc.GetValue(nil)
}

func (self *NtfsDisk) GetFileRecordFilename(record *core.FileRecord) (string, error) {
	return record.GetFilename(self.disk)
}

func (self *NtfsDisk) InitState(state IStateRecord) (bool, error) {
	return state.Init(self.disk)
}

func (self *NtfsDisk) Close() error {
	if self != nil {
		defer func() {
			self.disk = nil
		}()

		if self.disk != nil {
			return self.disk.Close()
		}
	}

	return nil
}

func OpenNtfsDisk(name string, mft_shift int64) (*NtfsDisk, error) {
	data, err := core.OpenDisk(name)
	if err != nil {
		return nil, err
	}

	res := &NtfsDisk{
		disk:      data,
		mft_shift: mft_shift,
	}

	if err := res.fill_runlist(); err != nil {
		return nil, err
	}

	return res, nil
}
