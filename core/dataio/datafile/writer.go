package datafile

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"essai/ntfstool/core"
	"essai/ntfstool/core/dataio"
	"essai/ntfstool/core/dataio/codec"
)

type DataWriter struct {
	tDataContainer

	writer *codec.Encoder
}

func (self *DataWriter) write_record(rec IDataRecord) error {
	res := make([]IDataRecord, 1, 1)
	res[0] = rec

	return WrapError(self.writer.Encode(res))
}

func (self *DataWriter) Close() (err error) {
	if err := self.check(); err != nil {
		return err
	}

	defer func() {
		if self.file != nil {
			res_err := self.file.Close()
			if err == nil {
				err = WrapError(res_err)
			}
		}

		self.file, self.infos.indexes, self.infos.headers, self.writer = nil, nil, nil, nil
	}()

	pos, err := self.get_pos()
	if err != nil {
		return err
	}

	self.infos.indexes = append(self.infos.indexes, [2]int64{-1, pos})

	if err := self.write_record(new(tNullRecord)); err != nil {
		return err
	}

	pos, err = self.get_pos()
	if err != nil {
		return err
	}

	if err := self.writer.Encode(self.infos.count); err != nil {
		return WrapError(err)
	}

	if err := self.writer.Encode(self.infos.headers); err != nil {
		return WrapError(err)
	}

	if err := self.writer.Encode(self.infos.indexes); err != nil {
		return WrapError(err)
	}

	return WrapError(binary.Write(self.file, binary.BigEndian, pos))
}

func (self *DataWriter) Write(rec IDataRecord) (err error) {
	defer func() {
		if err == nil {
			err = Recover()
		}

		if err == nil {
			self.infos.count++
		}
	}()

	if err := self.check(); err != nil {
		return err
	}

	pos, err := self.get_pos()
	if err != nil {
		return err
	}

	self.infos.indexes = append(self.infos.indexes, [2]int64{rec.GetPosition(), pos})

	return self.write_record(rec)
}

func OpenDataWriter(filename, format_name string) (*DataWriter, error) {
	f, err := core.OpenFile(filename, OPEN_WRONLY)
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
		return nil, WrapError(fmt.Errorf("Unknown file format: %s", format_name))
	}

	res := &DataWriter{
		tDataContainer: tDataContainer{
			file:   file,
			format: format,
		},
		writer: codec.MakeEncoder(file, format.registry),
	}

	if _, err := file.Write(format.signature); err != nil {
		return nil, WrapError(err)
	}

	old_pos, err := res.get_pos()
	if err != nil {
		return nil, err
	}

	res.infos.headers = make([]int16, len(format.headers))
	for i, header := range format.headers {
		if err := res.write_record(header); err != nil {
			return nil, err
		}

		pos, err := res.get_pos()
		if err != nil {
			return nil, err
		}

		res.infos.headers[i] = int16(pos - old_pos)
		fmt.Println("W:", res.infos.headers[i], old_pos, pos)
		old_pos = pos
	}

	return res, nil
}
