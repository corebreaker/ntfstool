package datafile

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

const (
	COMMON_DATA_FILEFORMAT_NAME = "<All>"
	SIGNATURE_LENGTH            = 16
)

type BaseDataRecord struct{}

func (*BaseDataRecord) IsNull() bool            { return false }
func (*BaseDataRecord) GetError() error         { return nil }
func (*BaseDataRecord) GetPosition() int64      { return 0 }
func (*BaseDataRecord) GetEncodingCode() string { return "" }
func (*BaseDataRecord) String() string          { return "{NONE}" }

type tNullRecord struct {
	BaseDataRecord

	zero int64
}

func (*tNullRecord) IsNull() bool            { return true }
func (*tNullRecord) GetError() error         { return nil }
func (*tNullRecord) GetPosition() int64      { return 0 }
func (*tNullRecord) GetEncodingCode() string { return "Z" }
func (*tNullRecord) String() string          { return "{NULL}" }
func (*tNullRecord) Print()                  { fmt.Println("{NULL}") }

type tFileFormat struct {
	name      string
	signature []byte
	headers   []IDataRecord
}

func (self *tFileFormat) String() string {
	return fmt.Sprint("Name:", self.name, "/ Headers:", self.headers)
}

var (
	file_formats    map[string]*tFileFormat = make(map[string]*tFileFormat)
	file_signatures map[string]*tFileFormat = make(map[string]*tFileFormat)
)

func RegisterFileFormat(name, signature string, headers ...IDataRecord) {
	var common_headers []IDataRecord

	if name != COMMON_DATA_FILEFORMAT_NAME {
		ref, ok := file_formats[COMMON_DATA_FILEFORMAT_NAME]
		if !ok {
			RegisterFileFormat(COMMON_DATA_FILEFORMAT_NAME, "", new(tNullRecord))
			ref = file_formats[COMMON_DATA_FILEFORMAT_NAME]
		}

		common_headers = ref.headers
	}

	sz := len(common_headers)
	res_headers := make([]IDataRecord, sz+len(headers))
	copy(res_headers, common_headers)
	copy(res_headers[sz:], headers)

	res := &tFileFormat{
		name:      name,
		headers:   res_headers,
		signature: make([]byte, SIGNATURE_LENGTH),
	}

	FillBuffer(res.signature, ' ')
	copy(res.signature, signature)

	for _, header := range headers {
		gob.RegisterName(header.GetEncodingCode(), header)
	}

	file_formats[name] = res
	file_signatures[string(res.signature)] = res
}

type tDataContainer struct {
	infos struct {
		count   int
		headers []int16
		indexes [][2]int64
	}

	format *tFileFormat
	file   *os.File
}

func (self *tDataContainer) get_pos() (int64, error) {
	res, err := self.file.Seek(0, os.SEEK_CUR)
	if err != nil {
		return 0, WrapError(err)
	}

	return res, nil
}

func (self *tDataContainer) check() error {
	if self.file == nil {
		return WrapError(fmt.Errorf("File closed"))
	}

	return nil
}

func (self *tDataContainer) GetCount() int {
	return self.infos.count
}

func (self *tDataContainer) GetFormatName() string {
	return self.format.name
}

type tIndex struct {
	start  int64
	length int64
}

type DataReader struct {
	tDataContainer

	positions map[int64]*tIndex
	buffer    *Buffer
}

func (self *DataReader) Close() error {
	if err := self.check(); err != nil {
		return err
	}

	defer func() {
		self.file, self.infos.indexes, self.infos.headers = nil, nil, nil
	}()

	return WrapError(self.file.Close())
}

func (self *DataReader) ReadRecord(position int64) (IDataRecord, error) {
	if err := self.check(); err != nil {
		return nil, err
	}

	idx, ok := self.positions[position]
	if !ok {
		return nil, WrapError(fmt.Errorf("Unknown position", position))
	}

	self.buffer.Reset()
	if _, err := self.file.ReadAt(self.buffer.Get(int(idx.length)), idx.start); err != nil {
		return nil, WrapError(err)
	}

	return ReadRecord(self.buffer)
}

func (self *DataReader) GetRecordAt(index int) (IDataRecord, error) {
	if err := self.check(); err != nil {
		return nil, err
	}

	if (0 > index) || (index >= self.infos.count) {
		return nil, WrapError(fmt.Errorf("Bad index %d (limit= %d)", index, self.infos.count))
	}

	return self.ReadRecord(self.infos.indexes[index][0])
}

func (self *DataReader) InitStream(stream IDataStream) error {
	if err := self.check(); err != nil {
		return err
	}

	file, err := DupFile(self.file)
	if err != nil {
		return err
	}

	if _, err := file.Seek(SIGNATURE_LENGTH, os.SEEK_SET); err != nil {
		return WrapError(err)
	}

	reader := gob.NewDecoder(file)

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
			record, err := ReadRecord(reader)
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
	f, err := OpenFile(filename, OPEN_RDONLY)
	if err != nil {
		return nil, WrapError(err)
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
			return nil, WrapError(fmt.Errorf("Unknown file format: %s", format_name))
		}
	}

	signature := make([]byte, SIGNATURE_LENGTH)

	if _, err := file.Read(signature); err != nil {
		return nil, WrapError(err)
	}

	switch format_name {
	case "", COMMON_DATA_FILEFORMAT_NAME:
		var ok bool

		format, ok = file_signatures[string(signature)]
		if !ok {
			return nil, WrapError(fmt.Errorf("Unknown file format: %s", format_name))
		}

	default:
		if string(signature) != string(format.signature) {
			return nil, WrapError(fmt.Errorf("Bad file format"))
		}
	}

	if _, err := file.Seek(-8, os.SEEK_END); err != nil {
		return nil, WrapError(err)
	}

	var position int64

	if err := binary.Read(file, binary.BigEndian, &position); err != nil {
		return nil, WrapError(err)
	}

	if _, err := file.Seek(position, os.SEEK_SET); err != nil {
		return nil, WrapError(err)
	}

	reader := gob.NewDecoder(file)

	res := &DataReader{
		tDataContainer: tDataContainer{
			format: format,
			file:   file,
		},
		positions: make(map[int64]*tIndex),
	}

	if err := reader.Decode(&res.infos.count); err != nil {
		return nil, WrapError(err)
	}

	if err := reader.Decode(&res.infos.headers); err != nil {
		return nil, WrapError(err)
	}

	if err := reader.Decode(&res.infos.indexes); err != nil {
		return nil, WrapError(err)
	}

	if _, err := file.Seek(SIGNATURE_LENGTH, os.SEEK_SET); err != nil {
		return nil, WrapError(err)
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
			return nil, WrapError(err)
		}

		p2, _ := res.get_pos()
		fmt.Println("L:", length, p1, p2)

		if _, err := ReadRecord(res.buffer); err != nil {
			return nil, err
		}
	}

	return res, nil
}

type DataWriter struct {
	tDataContainer

	writer *gob.Encoder
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
	f, err := OpenFile(filename, OPEN_WRONLY)
	if err != nil {
		return nil, WrapError(err)
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
		writer: gob.NewEncoder(file),
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
