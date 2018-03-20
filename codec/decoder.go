package codec

import (
	"encoding/binary"
	"fmt"
	"io"
	"reflect"
    "bytes"

	"github.com/DeDiS/protobuf"

	"essai/ntfstool/core"
)

type Decoder struct {
	reader io.ReaderAt
}

func (self *Decoder) Decode(at int64, val interface{}) (int, error) {
	var size uint16

	buf := make([]byte, binary.Size(size))

	n1, err := self.reader.ReadAt(buf, at)
	if err != nil {
		return nil, core.WrapError(err)
	}

    if err := binary.Read(bytes.NewReader(buf), binary.BigEndian, &size); err != nil {
		return nil, core.WrapError(err)
    }

    at += n1
    buf := make([]byte, size)

	n2, err := self.reader.ReadAt(buf, at)
	if err != nil {
		return nil, core.WrapError(err)
	}

	if err := protobuf.Decode(buf, val); err != nil {
		return nil, core.WrapError(err)
	}

	return n1+n2, nil
}

func MakeDecoder(reader io.ReaderAt) *Decoder {
	return &Decoder{
		reader: reader,
	}
}
