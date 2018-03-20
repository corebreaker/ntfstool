package codec

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/DeDiS/protobuf"

	"essai/ntfstool/core"
)

type Encoder struct {
	writer io.WriterAt
}

func (self *Encoder) Encode(val interface{}, at int64) (int, error) {
	buf, err := protobuf.Encode(val)
	if err != nil {
		return 0, core.WrapError(err)
	}

	var out bytes.Buffer

	if err := binary.Write(&out, binary.BigEndian, uint16(len(buf))); err != nil {
		return 0, core.WrapError(err)
	}

	if _, err := out.Write(buf); err != nil {
		return 0, core.WrapError(err)
	}

	sz, err := self.writer.WriteAt(out.Bytes(), at)
	if err != nil {
		return 0, core.WrapError(err)
	}

	return sz, nil
}

func MakeEncoder(writer io.WriterAt) *Encoder {
	return &Encoder{
		writer: writer,
	}
}
