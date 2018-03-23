package datafile

import (
	"encoding/binary"
	"fmt"
	"os"

	"essai/ntfstool/core"
	"essai/ntfstool/core/dataio"
	"essai/ntfstool/core/dataio/codec"
)

type DataWriter struct {
	tDataContainer

	writer *codec.Encoder
}

func (self *DataWriter) write_record(rec dataio.IDataRecord) error {
	_, err := self.writer.Encode(rec)

	return err
}

func (self *DataWriter) Close() (err error) {
	if err := self.check(); err != nil {
		return err
	}

	defer func() {
		if self.file != nil {
			res_err := self.file.Close()
			if err == nil {
				err = core.WrapError(res_err)
			}
		}

		self.file, self.desc.Indexes, self.desc.Headers, self.writer = nil, nil, nil, nil
	}()

	pos, err := self.get_pos()
	if err != nil {
		return err
	}

	self.desc.Indexes = append(self.desc.Indexes, tFileIndex{Logical: -1, Physical: pos})

	if err := self.write_record(new(tNullRecord)); err != nil {
		return err
	}

	self.desc.Position, err = self.get_pos()
	if err != nil {
		return err
	}

	if err := self.write_record(self.desc.GetRecord()); err != nil {
		return err
	}

	return core.WrapError(binary.Write(self.file, binary.BigEndian, self.desc.Position))
}

func (self *DataWriter) Write(rec dataio.IDataRecord) (err error) {
	defer func() {
		if err == nil {
			err = core.Recover()
		}

		if err == nil {
			self.desc.Count++
		}
	}()

	if err := self.check(); err != nil {
		return err
	}

	pos, err := self.get_pos()
	if err != nil {
		return err
	}

	self.desc.Indexes = append(self.desc.Indexes, tFileIndex{Logical: rec.GetPosition(), Physical: pos})

	return self.write_record(rec)
}

func OpenDataWriter(filename, format_name string) (*DataWriter, error) {
	f, err := core.OpenFile(filename, core.OPEN_WRONLY)
	if err != nil {
		return nil, err
	}

	res, err := MakeDataWriter(f, format_name)
	if err != nil {
		f.Close()
	}

	return res, err
}

func MakeDataWriter(file *os.File, format_name string) (*DataWriter, error) {
	format, ok := file_formats[format_name]
	if !ok {
		return nil, core.WrapError(fmt.Errorf("Unknown file format: %s", format_name))
	}

	res := &DataWriter{
		tDataContainer: tDataContainer{
			file:   file,
			format: format,
		},
		writer: codec.MakeEncoder(file, format.registry),
	}

	if _, err := file.Write(format.signature); err != nil {
		return nil, core.WrapError(err)
	}

	old_pos, err := res.get_pos()
	if err != nil {
		return nil, err
	}

	res.desc.Headers = make([]int16, len(format.headers))
	for i, header := range format.headers {
		if err := res.write_record(header); err != nil {
			return nil, err
		}

		pos, err := res.get_pos()
		if err != nil {
			return nil, err
		}

		res.desc.Headers[i] = int16(pos - old_pos)
		old_pos = pos
	}

	return res, nil
}
