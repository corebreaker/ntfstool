package core

import (
	"os"
	"reflect"
)

var (
	header = reflect.TypeOf(RecordHeader{})
)

type tSharedFile struct {
	file           *os.File
	instance_count uint
}

func (self *tSharedFile) inc() *tSharedFile {
	self.instance_count++

	return self
}

func (self *tSharedFile) close() error {
	self.instance_count--
	if self.instance_count != 0 {
		return nil
	}

	return WrapError(self.file.Close())
}

type DiskIO struct {
	in     *tSharedFile
	buffer [512]byte
	offset int64
}

func (self *DiskIO) GetOffset() int64 {
	return self.offset
}

func (self *DiskIO) SetOffset(offset int64) {
	self.offset = offset
}

func (self *DiskIO) Shift(offset int64) *DiskIO {
	return &DiskIO{
		in:     self.in.inc(),
		offset: self.offset + offset,
	}
}

func (self *DiskIO) ReadSector(position int64, data []byte) error {
	buffer := data
	to_copy := false

	buf_sz := len(buffer)
	if buf_sz > 512 {
		buffer = buffer[:512]
	} else {
		to_copy = buf_sz < 512
		if to_copy {
			buffer = self.buffer[:]
		}
	}

	_, err := self.in.file.ReadAt(buffer, self.offset+(position*512))

	if (err == nil) && to_copy {
		copy(data, buffer)
	}

	return WrapError(err)
}

func (self *DiskIO) ReadSectors(position, count int64, data []byte) error {
	buffer := data
	to_copy := false

	buf_sz := len(buffer)
	read_sz := int(count * 512)
	if buf_sz > read_sz {
		buffer = buffer[:read_sz]
	} else {
		to_copy = buf_sz < 512
		if to_copy {
			buffer = self.buffer[:]
		}
	}

	_, err := self.in.file.ReadAt(buffer, self.offset+(position*512))

	if (err == nil) && to_copy {
		copy(data, buffer)
	}

	return WrapError(err)
}

func (self *DiskIO) ReadCluster(position int64, data []byte) error {
	return self.ReadSectors(position*8, 8, data)
}

func (self *DiskIO) ReadClusters(position, count int64, data []byte) error {
	return self.ReadSectors(position*8, 8*count, data)
}

func (self *DiskIO) ReadStruct(position int64, ptr interface{}) error {
	sz := int64(StructSize(ptr))
	buffer := make([]byte, sz)

	if err := self.ReadSectors(position, (sz+511)/512, buffer); err != nil {
		return err
	}

	first := reflect.ValueOf(ptr).Elem().Type().Field(0)
	if first.Anonymous && (first.Type == header) {
		var h RecordHeader

		if err := Read(buffer, &h); err != nil {
			return WrapError(err)
		}

		if (h.UsaCount <= 3) && ((h.UsaOffset + (h.UsaCount * 2)) < 1024) {
			idx := int64(h.UsaOffset + 2)
			pos := int64(510) // 255 * 2
			for i := uint16(1); i < h.UsaCount; i++ {
				if (pos >= sz) || ((idx + 2) >= sz) {
					break
				}

				copy(buffer[pos:], buffer[idx:(idx+2)])
				idx += 2
				pos += 512
			}
		}
	}

	return WrapError(Read(buffer, ptr))
}

func (self *DiskIO) Close() error {
	if self == nil {
		return nil
	}

	return self.in.close()
}

func OpenDisk(name string) (*DiskIO, error) {
	file, err := os.Open(name)
	if err != nil {
		return nil, WrapError(err)
	}

	res := &DiskIO{
		in: &tSharedFile{
			file:           file,
			instance_count: 1,
		},
	}

	return res, nil
}
