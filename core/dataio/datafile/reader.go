package datafile

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"essai/ntfstool/core"
	"essai/ntfstool/core/dataio"
	"essai/ntfstool/core/dataio/buffer"
	"essai/ntfstool/core/dataio/codec"
)

type DataReader struct {
	tDataContainer

	positions map[int64]*tIndex
	buffer    *buffer.Buffer
}

func (self *DataReader) Close() error {
	if err := self.check(); err != nil {
		return err
	}

	defer func() {
		self.file, self.desc.Indexes, self.desc.Headers = nil, nil, nil
	}()

	return core.WrapError(self.file.Close())
}

func (self *DataReader) ReadRecord(position int64) (dataio.IDataRecord, error) {
	if err := self.check(); err != nil {
		return nil, err
	}

	idx, ok := self.positions[position]
	if !ok {
		return nil, core.WrapError(fmt.Errorf("Unknown position", position))
	}

	self.buffer.Reset()
	if _, err := self.file.ReadAt(self.buffer.Get(int(idx.length)), idx.start); err != nil {
		return nil, core.WrapError(err)
	}

	return core.ReadRecord(self.buffer)
}

func (self *DataReader) GetRecordAt(index int) (dataio.IDataRecord, error) {
	if err := self.check(); err != nil {
		return nil, err
	}

	if (0 > index) || (index >= int(self.desc.Count)) {
		return nil, core.WrapError(fmt.Errorf("Bad index %d (limit= %d)", index, self.desc.Count))
	}

	return self.ReadRecord(self.desc.Indexes[index].Logical)
}

func (self *DataReader) InitStream(stream dataio.IDataStream) error {
	if err := self.check(); err != nil {
		return err
	}

	file, err := DupFile(self.file)
	if err != nil {
		return err
	}

	if _, err := file.Seek(SIGNATURE_LENGTH, os.SEEK_SET); err != nil {
		return core.WrapError(err)
	}

	reader := codec.MakeDecoder(file, self.format.registry)
	decoder := reader.ToCoreDecoder()

	for _ = range self.desc.Headers {
		if _, err := core.ReadRecord(decoder); err != nil {
			return err
		}
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

		cnt := self.desc.Count
		for i := uint32(0); i < cnt; i++ {
			record, err := core.ReadRecord(decoder)
			if err != nil {
				if err != io.EOF {
					stream.SendError(err)
				}

				return
			}

			stream.SendRecord(record)
		}
	}()

	return nil
}

func (self *DataReader) IsClosed() bool {
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
	case "", COMMON_DATA_FILEFORMAT_NAME:
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
	case "", COMMON_DATA_FILEFORMAT_NAME:
		var ok bool

		format, ok = file_signatures[string(signature)]
		if !ok {
			return nil, core.WrapError(fmt.Errorf("Unknown file format: %s", format_name))
		}

	default:
		if string(signature) != string(format.signature) {
			return nil, core.WrapError(fmt.Errorf("Bad file format"))
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

	var desc_record tFileDescRecord

	if _, err := reader.Decode(&desc_record); err != nil {
		return nil, err
	}

	if _, err := file.Seek(SIGNATURE_LENGTH, os.SEEK_SET); err != nil {
		return nil, core.WrapError(err)
	}

	res := &DataReader{
		tDataContainer: tDataContainer{
			format: format,
			file:   file,
		},
		positions: make(map[int64]*tIndex),
	}

	res.desc.FromRecord(&desc_record)

	prev := new(tIndex)
	max_len := int64(0)

	for _, l := range res.desc.Headers {
		length := int64(l)
		if max_len < length {
			max_len = length
		}
	}

	for _, pos := range res.desc.Indexes {
		idx := &tIndex{
			start: pos.Physical,
		}

		if idx.start < 0 {
			idx.start = position
		}

		start := pos.Logical

		if start != 0 {
			res.positions[start] = idx
		}

		length := idx.start - prev.start
		if max_len < length {
			max_len = length
		}

		prev.length = length
		prev = idx
	}

	if prev.start != position {
		length := position - prev.start
		if max_len < length {
			max_len = length
		}

		prev.length = length
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
