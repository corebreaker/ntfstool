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

type DataReader struct {
	tDataContainer

	positions map[int64]*tIndex
}

func (self *DataReader) Close() error {
	if err := self.check(); err != nil {
		return err
	}

	defer func() {
		self.file, self.infos.indexes, self.infos.headers = nil, nil, nil
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

	if (0 > index) || (index >= self.infos.count) {
		return nil, core.WrapError(fmt.Errorf("Bad index %d (limit= %d)", index, self.infos.count))
	}

	return self.ReadRecord(self.infos.indexes[index][0])
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

	reader := codec.MakeDecoder(self.file, self.format.registry)

	for _ = range self.infos.headers {
		if _, err := ReadRecord(reader); err != nil {
			return WrapError(err)
		}
	}

	close_stream := func() (err error) {
		defer func() {
			recover()
			err = file.Close()
		}()

		stream.Close()

		return
	}

	go func() {
		defer DeferedCall(close_stream)

		for {
			record, err := core.ReadRecord(reader)
			if err != nil {
				if err != io.EOF {
					stream.SendError(err)
				}

				return
			}

			if !record.IsNull() {
				stream.SendRecord(record)
			}
		}
	}()

	return nil
}

func OpenDataReader(filename, format_name string) (*DataReader, error) {
	f, err := core.OpenFile(filename, OPEN_RDONLY)
	if err != nil {
		return nil, err
	}

	defer f.Close()

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
			return nil, WrapError(fmt.Errorf("Bad file format"))
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
			format: format,
			file:   file,
		},
		positions: make(map[int64]*tIndex),
	}

	if err := reader.Decode(&res.infos.count); err != nil {
		return nil, core.WrapError(err)
	}

	if err := reader.Decode(&res.infos.headers); err != nil {
		return nil, core.WrapError(err)
	}

	if err := reader.Decode(&res.infos.indexes); err != nil {
		return nil, core.WrapError(err)
	}

	if _, err := file.Seek(SIGNATURE_LENGTH, os.SEEK_SET); err != nil {
		return nil, core.WrapError(err)
	}

	prev := new(tIndex)
	max_len := int64(0)

	for _, l := range res.infos.headers {
		length := int64(l)
		if max_len < length {
			max_len = length
		}
	}

	for _, pos := range res.infos.indexes {
		idx := &tIndex{
			start: pos[1],
		}

		if idx.start < 0 {
			idx.start = position
		}

		start := pos[0]

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

	res.buffer = MakeBuffer(int(max_len))

	for _, l := range res.infos.headers {
		length := int64(l)
		p1, _ := res.get_pos()

		res.buffer.Reset()
		if _, err := file.Read(res.buffer.Get(int(length))); err != nil {
			return nil, core.WrapError(err)
		}

		p2, _ := res.get_pos()
		fmt.Println("L:", length, p1, p2)

		if _, err := core.ReadRecord(res.buffer); err != nil {
			return nil, err
		}
	}

	return res, nil
}
