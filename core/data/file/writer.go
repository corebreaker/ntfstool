package file

import (
	"encoding/binary"
	"fmt"
	"os"

	"essai/ntfstool/core"
	"essai/ntfstool/core/data"
	"essai/ntfstool/core/data/codec"
)

type DataWriter struct {
	tDataContainer

	writer *codec.Encoder
}

func (self *DataWriter) write_record(rec data.IDataRecord) error {
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

		self.file, self.writer, self.desc.Indexes, self.desc.Headers, self.desc.Counts = nil, nil, nil, nil, nil
	}()

	pos, err := self.get_pos()
	if err != nil {
		return err
	}

	self.desc.Indexes = append(self.desc.Indexes, &tFileIndex{Logical: -1, Physical: pos})

	if err := self.write_record(new(tNullRecord)); err != nil {
		return err
	}

	self.desc.Position, err = self.get_pos()
	if err != nil {
		return err
	}

	if err := self.write_record(&self.desc); err != nil {
		return err
	}

	return core.WrapError(binary.Write(self.file, binary.BigEndian, self.desc.Position))
}

func (self *DataWriter) Write(rec data.IDataRecord) (err error) {
	defer core.Recover(func(e error) {
		if e == nil {
			if err == nil {
				code := rec.GetEncodingCode()

				self.desc.Count++
				self.desc.Counts[code]++
			}
		} else {
			e = err
		}
	})

	if err := self.check(); err != nil {
		return err
	}

	pos, err := self.get_pos()
	if err != nil {
		return err
	}

	defer core.Recover(func(e error) {
		if e == nil {
			if err == nil {
				self.desc.Indexes = append(self.desc.Indexes, &tFileIndex{Logical: rec.GetPosition(), Physical: pos})
			}
		} else {
			e = err
		}
	})

	return self.write_record(rec)
}

func (self *DataWriter) IsClosed() bool {
	if self.file == nil {
		return true
	}

	_, err := self.file.Seek(0, os.SEEK_END)
	if err == nil {
		return false
	}

	oserr, ok := err.(*os.PathError)
	if !ok {
		return false
	}

	return oserr.Err == os.ErrClosed
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
			desc: tFileDesc{
				Counts: make(map[string]uint32),
			},
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
