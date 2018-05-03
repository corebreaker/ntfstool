package core

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"io"
	"unicode/utf16"

	"github.com/pborman/uuid"
)

func Read(buffer []byte, data interface{}) error {
	sz := StructSize(data)
	if len(buffer) < sz {
		src := buffer
		buffer = make([]byte, sz)
		copy(buffer, src)
	}

	return WrapError(binary.Read(bytes.NewReader(buffer), binary.LittleEndian, data))
}

func Write(writer io.Writer, data interface{}) error {
	return WrapError(binary.Write(writer, binary.LittleEndian, data))
}

func DecodeInt(b []byte) int64 {
	if len(b) == 0 {
		return 0
	}

	var mem [8]byte
	var res int64

	buffer := mem[:]

	copy(buffer, b)
	Read(buffer, &res)

	return res
}

func DecodeString(b []byte, sz int) string {
	str_sz := (len(b) + 1) / 2
	if (0 >= sz) || (sz > str_sz) {
		sz = str_sz
	}

	buffer := make([]byte, sz*2)
	copy(buffer, b)

	str16 := make([]uint16, sz)

	Read(buffer, str16)

	return string(utf16.Decode(str16))
}

func StringSize(s string) int {
	return len([]rune(s))
}

func NewFileId() string {
	id := uuid.NewRandom()

	res := make([]byte, len(id)*2)
	hex.Encode(res, id)

	return string(res)
}
