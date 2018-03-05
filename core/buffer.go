package core

import (
    "encoding/gob"
    "io"
)

type tBufData struct {
    buffer []byte
    offset int
}

func (self *tBufData) Read(p []byte) (n int, err error) {
    if self.offset >= len(self.buffer) {
        if len(p) == 0 {
            return
        }

        return 0, io.EOF
    }

    n = copy(p, self.buffer[self.offset:])
    self.offset += n

    return
}

type Buffer struct {
    data   *tBufData
    reader *gob.Decoder
}

func (self *Buffer) Get(size int) []byte {
    data := self.data
    if (0 >= size) || (size > len(data.buffer)) {
        return data.buffer
    }

    return data.buffer[:size]
}

func (self *Buffer) Reset() {
    self.data.offset = 0
}

func (self *Buffer) Decode(data interface{}) error {
    return WrapError(self.reader.Decode(data))
}

func MakeBuffer(size int) *Buffer {
    data := &tBufData{
        buffer: make([]byte, size),
    }

    return &Buffer{
        data:   data,
        reader: gob.NewDecoder(data),
    }
}
