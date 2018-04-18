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

type DataModifier struct {
	tDataContainer

	position  int64
	positions map[int64]int
	writer    *codec.Encoder
	buffer    *buffer.Buffer
}

func (self *DataModifier) write_record(rec data.IDataRecord) error {
	_, err := self.file.Seek(self.position, os.SEEK_SET)
	if err != nil {
		return core.WrapError(err)
	}

	sz, err := self.writer.Encode(rec)
	if err != nil {
		return err
	}

	self.position += int64(sz)
	bufsz := self.buffer.GetSize()
	if sz > bufsz {
		self.buffer.SetSize(bufsz)
	}

	return nil
}

func (self *DataModifier) Close() (err error) {
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

		self.desc.Indexes, self.desc.Headers, self.desc.Counts = nil, nil, nil
		self.file, self.positions, self.buffer, self.writer = nil, nil, nil, nil
	}()

	if err := self.write_record(new(tNullRecord)); err != nil {
		return err
	}

	if self.position < self.desc.Position {
		self.position = self.desc.Position
	} else {
		self.desc.Position = self.position
	}

	if err := self.write_record(&self.desc); err != nil {
		return err
	}

	return core.WrapError(binary.Write(self.file, binary.BigEndian, self.desc.Position))
}

func (self *DataModifier) Write(rec data.IDataRecord) (err error) {
	defer core.Recover(func(e error) {
		if e == nil {
			if err == nil {
				code := rec.GetEncodingCode()

				self.desc.Count++
				self.desc.Counts[code]++

				pos := rec.GetPosition()
				if pos > 0 {
					self.positions[pos] = len(self.desc.Indexes)
				}
			}
		} else {
			e = err
		}
	})

	if err := self.check(); err != nil {
		return err
	}

	filepos := self.position

	defer core.Recover(func(e error) {
		if e == nil {
			if err == nil {
				last := len(self.desc.Indexes) - 1
				self.desc.Indexes = append(self.desc.Indexes, self.desc.Indexes[last])
				self.desc.Indexes[last] = &tFileIndex{
					Logical:  rec.GetPosition(),
					Physical: filepos,
				}

				self.desc.Indexes[last+1].Physical = self.position
			}
		} else {
			e = err
		}
	})

	return self.write_record(rec)
}

func (self *DataModifier) SetRecordAt(index int, rec data.IDataRecord) (err error) {
	if err := self.check(); err != nil {
		return err
	}

	if (0 > index) || (index >= int(self.desc.Count)) {
		return core.WrapError(fmt.Errorf("Bad index %d (limit= %d)", index, self.desc.Count))
	}

	old_start := self.desc.Indexes[index].Physical
	old_size := self.desc.Indexes[index+1].Physical - old_start

	self.buffer.Reset()
	if _, err := self.file.ReadAt(self.buffer.Get(int(old_size)), old_start); err != nil {
		return core.WrapError(err)
	}

	old, err := core.ReadRecord(self.buffer)
	if err != nil {
		return err
	}

	trailer_start := self.position

	if err := self.write_record(rec); err != nil {
		return err
	}

	trailer_end := self.position
	rec_size := trailer_end - trailer_start

	rec_buf := self.buffer.Get(int(rec_size))

	if _, err := self.file.ReadAt(rec_buf, trailer_start); err != nil {
		return core.WrapError(err)
	}

	var move_buf [65536]byte

	bufsz := int64(len(move_buf))
	move_end := old_start + rec_size

	for pos := trailer_end; pos > move_end; pos -= bufsz {
		dest := pos - bufsz
		src := dest - rec_size
		if src < old_start {
			src = old_start
			dest = src + rec_size
		}

		b := move_buf[:(pos - dest)]

		if _, err := self.file.ReadAt(b, src); err != nil {
			return core.WrapError(err)
		}

		if _, err := self.file.WriteAt(b, dest); err != nil {
			return core.WrapError(err)
		}
	}

	if _, err := self.file.WriteAt(rec_buf, old_start); err != nil {
		return core.WrapError(err)
	}

	self.desc.Counts[old.GetEncodingCode()]--
	self.desc.Counts[rec.GetEncodingCode()]++

	pos := old.GetPosition()
	if pos > 0 {
		delete(self.positions, pos)
	}

	pos = rec.GetPosition()
	if pos > 0 {
		self.positions[pos] = index
	}

	self.desc.Indexes[index].Logical = pos

	delta := rec_size - old_size
	if delta != 0 {
		for _, idx := range self.desc.Indexes[index:] {
			idx.Physical += delta
		}
	}

	self.position = trailer_start + delta

	return nil
}

func (self *DataModifier) ReadRecord(position int64) (data.IDataRecord, error) {
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

func (self *DataModifier) GetRecordAt(index int) (data.IDataRecord, error) {
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

func (self *DataModifier) InitStream(stream data.IDataStream) error {
	return self.InitStreamFrom(stream, 0)
}

func (self *DataModifier) InitStreamFrom(stream data.IDataStream, index int64) error {
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

func (self *DataModifier) IsClosed() bool {
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

func OpenDataModifier(filename, format_name string) (*DataModifier, error) {
	f, err := core.OpenFile(filename, core.OPEN_RDWR)
	if err != nil {
		return nil, err
	}

	res, err := MakeDataModifier(f, format_name)
	if err != nil {
		f.Close()
	}

	return res, err
}

func MakeDataModifier(file *os.File, format_name string) (*DataModifier, error) {
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

	res := &DataModifier{
		tDataContainer: tDataContainer{
			desc: tFileDesc{
				Counts: make(map[string]uint32),
			},
			format: format,
			file:   file,
		},
		positions: make(map[int64]int),
		writer:    codec.MakeEncoder(file, format.registry),
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

	res.position = res.desc.Indexes[res.desc.Count].Physical
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
