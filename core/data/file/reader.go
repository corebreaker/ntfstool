package file

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"essai/ntfstool/core"
	"essai/ntfstool/core/data"
	"essai/ntfstool/core/data/buffer"
	"essai/ntfstool/core/data/codec"
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

func (self *DataReader) ReadRecord(position int64) (data.IDataRecord, error) {
	if err := self.check(); err != nil {
		return nil, err
	}

	idx, ok := self.positions[position]
	if !ok {
		return nil, core.WrapError(fmt.Errorf("Unknown position: %d", position))
	}

	self.buffer.Reset()
	if _, err := self.file.ReadAt(self.buffer.Get(int(idx.length)), idx.start); err != nil {
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
		positions: make(map[int64]*tIndex),
	}

	if _, err := reader.Decode(&res.desc); err != nil {
		return nil, err
	}

	if _, err := file.Seek(SIGNATURE_LENGTH, os.SEEK_SET); err != nil {
		return nil, core.WrapError(err)
	}

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

		if pos.Logical != 0 {
			res.positions[pos.Logical] = idx
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
