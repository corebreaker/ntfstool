package codec

import (
	"fmt"
	"io"
	"reflect"

	"github.com/DeDiS/protobuf"

	"essai/ntfstool/core"
)

type Decoder struct {
	reader   io.Reader
	registry *Registry
}

func (self *Decoder) Decode() (interface{}, error) {
	var size_buf [1]byte

	if _, err := self.reader.Read(size_buf[:]); err != nil {
		return nil, core.WrapError(err)
	}

	header_buf := make([]byte, int(size_buf[0]+1))
	if _, err := self.reader.Read(header_buf); err != nil {
		return nil, core.WrapError(err)
	}

	var header tEntryHeader

	if err := protobuf.Decode(header_buf, &header); err != nil {
		return nil, core.WrapError(err)
	}

	t, ok := self.registry.foreward[header.Type]
	if !ok {
		return nil, core.WrapError(fmt.Errorf("Unknown type with name = `%s`", header.Type))
	}

	buf := make([]byte, header.Size)
	if _, err := self.reader.Read(buf); err != nil {
		return nil, core.WrapError(err)
	}

	res := reflect.New(t)

	if err := protobuf.Decode(buf, res.Interface()); err != nil {
		return nil, core.WrapError(err)
	}

	return res, nil
}

func MakeDecoder(reader io.Reader, registry *Registry) *Decoder {
	return &Decoder{
		reader:   reader,
		registry: registry,
	}
}
