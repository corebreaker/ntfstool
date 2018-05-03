package inspect

import (
    "essai/ntfstool/core"
)

type NtfsDisk struct {
    data   *core.DiskIO
    mft_io *core.DiskIO
    mft_rl core.RunList
}

func (self *NtfsDisk) fill_runlist() error {
    var mft core.FileRecord

    if err := self.mft_io.ReadStruct(0, &mft); err != nil {
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

func (self *NtfsDisk) GetDisk() *core.DiskIO {
    return self.data.Shift(0)
}

func (self *NtfsDisk) SetStart(start int64) error {
    shift := self.mft_io.GetOffset() - self.data.GetOffset()

    self.data.SetOffset(start)
    self.mft_io.SetOffset(start + shift)

    return self.fill_runlist()
}

func (self *NtfsDisk) SetMftShift(shift int64) error {
    self.mft_io.SetOffset(self.data.GetOffset() + shift)

    return self.fill_runlist()
}

func (self *NtfsDisk) ReadRecordHeader(index int64, header *core.RecordHeader) error {
    return self.mft_io.ReadStruct(index*2, header)
}

func (self *NtfsDisk) ReadRecordHeaderFromRef(ref core.FileReferenceNumber, header *core.RecordHeader) error {
    return self.ReadRecordHeader(ref.GetFileIndex(), header)
}

func (self *NtfsDisk) ReadFileRecord(index int64, record *core.FileRecord) error {
    return self.data.ReadStruct(self.get_file_sector(index), record)
}

func (self *NtfsDisk) ReadFileRecordFromRef(ref core.FileReferenceNumber, record *core.FileRecord) error {
    return self.ReadFileRecord(ref.GetFileIndex(), record)
}

func (self *NtfsDisk) GetAttributeValue(desc *core.AttributeDesc, read bool) (*core.AttributeValue, error) {
    if read {
        return desc.GetValue(self.data)
    }

    return desc.GetValue(nil)
}

func (self *NtfsDisk) GetFileRecordFilename(record *core.FileRecord) (string, error) {
    return record.GetFilename(self.data)
}

func (self *NtfsDisk) InitState(state IStateRecord) (bool, error) {
    return state.Init(self.data)
}

func (self *NtfsDisk) Close() error {
    if self != nil {
        defer func() {
            self.mft_io = nil
            self.data = nil
        }()

        if self.mft_io != nil {
            self.mft_io.Close()
        }

        if self.data != nil {
            return self.data.Close()
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
        data: data,
    }

    if mft_shift == 0 {
        res.mft_io = data.Shift(0)
    } else {
        if err := res.SetMftShift(mft_shift); err != nil {
            return nil, err
        }
    }

    return res, nil
}
