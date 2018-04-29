package file

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/corebreaker/ntfstool/core"
	"github.com/corebreaker/ntfstool/core/data"
	"github.com/corebreaker/ntfstool/core/data/buffer"
	"github.com/corebreaker/ntfstool/core/data/codec"
)

type DataReader struct {
	tDataContainer

	positions map[int64]int
	buffer    *buffer.Buffer
}

func (self *DataReader) Close() error {
	if err := self.check(); err != nil {
		return err
	}

	defer func() {
		desc := self.desc
		self.file, self.positions, self.buffer, desc.Indexes, desc.Headers, desc.Counts = nil, nil, nil, nil, nil, nil
	}()

	return core.WrapError(self.file.Close())
}

func (self *DataReader) ReadRecord(position int64) (data.IDataRecord, error) {
	if err := self.check(); err != nil {
		return nil, err
	}

	idx_pos, ok := self.positions[position]
	if !ok {
		return nil, core.WrapError(fmt.Errorf("Unknown position: %d", position))
	}

	if uint32(idx_pos) >= self.desc.Count {
		return nil, core.WrapError(fmt.Errorf("Bad index range: %d", idx_pos))
	}

	start := self.desc.Indexes[idx_pos].Physical
	end := self.desc.Indexes[idx_pos+1].Physical

	self.buffer.Reset()
	if _, err := self.file.ReadAt(self.buffer.Get(int(end-start)), start); err != nil {
		return nil, core.WrapError(err)
	}

	return core.ReadRecord(self.buffer)
}

func (self *DataReader) GetRecordAt(index int) (data.IDataRecord, error) {
	if err := self.check(); err != nil {
		return nil, err
	}

	if (0 > index) || (index >= int(self.desc.Count)) {
		return nil, core.WrapError(fmt.Errorf("Bad index %d (limit= %d)", index, self.desc.Count))
	}

	start := self.desc.Indexes[index].Physical
	length := int(self.desc.Indexes[index+1].Physical - start)

	self.buffer.Reset()
	if _, err := self.file.ReadAt(self.buffer.Get(length), start); err != nil {
		return nil, core.WrapError(err)
	}

	return core.ReadRecord(self.buffer)
}

func (self *DataReader) InitStream(stream data.IDataStream) error {
	return self.InitStreamFrom(stream, 0)
}

func (self *DataReader) InitStreamFrom(stream data.IDataStream, index int64) error {
	if err := self.check(); err != nil {
		return err
	}

	file, err := DupFile(self.file)
	if err != nil {
		return err
	}

	cnt := int64(self.desc.Count)
	if (0 > index) || (index >= cnt) {
		return core.WrapError(fmt.Errorf("Bad index %d (limit= %d)", index, cnt))
	}

	reader := codec.MakeDecoder(file, self.format.registry)
	decoder := reader.ToCoreDecoder()

	if _, err := file.Seek(SIGNATURE_LENGTH, os.SEEK_SET); err != nil {
		return core.WrapError(err)
	}

	for _ = range self.desc.Headers {
		if _, err := core.ReadRecord(decoder); err != nil {
			return err
		}
	}

	if _, err := file.Seek(self.desc.Indexes[index].Physical, os.SEEK_SET); err != nil {
		return core.WrapError(err)
	}

	close_stream := func() (err error) {
		defer func() {
			defer func() {
				e := file.Close()
				if err == nil {
					err = core.WrapError(e)
				}
			}()

			verr := recover()
			if verr != nil {
				e, ok := verr.(error)
				if !ok {
					e = fmt.Errorf("Error: %s", e)
				}

				err = core.WrapError(e)
			}
		}()

		return stream.Close()
	}

	go func() {
		defer core.DeferedCall(close_stream)

		for i := index; i < cnt; i++ {
			pos, err := file.Seek(0, os.SEEK_CUR)
			if err != nil {
				if err != io.EOF {
					stream.SendError(core.WrapError(err))
				}

				return
			}

			record, err := core.ReadRecord(decoder)
			if err != nil {
				if core.GetSource(err) != io.EOF {
					stream.SendError(err)
				}

				return
			}

			stream.SendRecord(uint(i), pos, record)
		}
	}()

	return nil
}

func (self *DataReader) IsClosed() bool {
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

func OpenDataReader(filename, format_name string) (*DataReader, error) {
	f, err := core.OpenFile(filename, core.OPEN_RDONLY)
	if err != nil {
		return nil, err
	}

	return MakeDataReader(f, format_name)
}

func MakeDataReader(file *os.File, format_name string) (*DataReader, error) {
	var format *tFileFormat

	switch format_name {
	case "", ANY_FILEFORMAT:
	default:
		var ok bool

		format, ok = file_formats[format_name]
		if !ok {
			return nil, core.WrapError(fmt.Errorf("Unknown file format: %s", format_name))
		}
	}

	signature := make([]byte, SIGNATURE_LENGTH)

	if _, err := file.Read(signature); err != nil {
		return nil, core.WrapError(err)
	}

	switch format_name {
	case "", ANY_FILEFORMAT:
		var ok bool

		format, ok = file_signatures[string(signature)]
		if !ok {
			return nil, core.WrapError(fmt.Errorf("Unknown file format: %s", format_name))
		}

	default:
		if string(signature) != string(format.signature) {
			return nil, core.WrapError(fmt.Errorf("Bad file format: `%s` != `%s`", signature, format.signature))
		}
	}

	if _, err := file.Seek(-8, os.SEEK_END); err != nil {
		return nil, core.WrapError(err)
	}

	var position int64

	if err := binary.Read(file, binary.BigEndian, &position); err != nil {
		return nil, core.WrapError(err)
	}

	if _, err := file.Seek(position, os.SEEK_SET); err != nil {
		return nil, core.WrapError(err)
	}

	reader := codec.MakeDecoder(file, format.registry)

	res := &DataReader{
		tDataContainer: tDataContainer{
			desc: tFileDesc{
				Counts: make(map[string]uint32),
			},
			format: format,
			file:   file,
		},
		positions: make(map[int64]int),
	}

	if _, err := reader.Decode(&res.desc); err != nil {
		return nil, err
	}

	max_len, prev := int64(0), int64(0)

	for _, l := range res.desc.Headers {
		length := int64(l)
		if max_len < length {
			max_len = length
		}
	}

	for i, pos := range res.desc.Indexes {
		part_pos := pos.Logical
		if part_pos > 0 {
			res.positions[part_pos] = i
		}

		start := pos.Physical
		if start < 0 {
			start = position
		}

		length := prev - start
		if max_len < length {
			max_len = length
		}
	}

	if prev != position {
		length := position - prev
		if max_len < length {
			max_len = length
		}
	}

	if _, err := file.Seek(SIGNATURE_LENGTH, os.SEEK_SET); err != nil {
		return nil, core.WrapError(err)
	}

	res.buffer = buffer.MakeBuffer(int(max_len), res.format.registry)

	for _, l := range res.desc.Headers {
		length := int64(l)

		res.buffer.Reset()
		if _, err := file.Read(res.buffer.Get(int(length))); err != nil {
			return nil, core.WrapError(err)
		}

		if _, err := core.ReadRecord(res.buffer); err != nil {
			return nil, err
		}
	}

	return res, nil
}
