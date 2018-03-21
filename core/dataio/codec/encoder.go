package codec

import (
	"bytes"
	"fmt"
	"io"
	"reflect"

	"github.com/DeDiS/protobuf"

	"essai/ntfstool/core"
)

type Encoder struct {
	writer   io.Writer
	registry *Registry
}

func (self *Encoder) Encode(val interface{}) (int, error) {
	v := normalize_value(val)
	if v.Kind() != reflect.Struct {
		return 0, core.WrapError(fmt.Errorf("Value should have a struct type"))
	}

	t, ok := self.registry.backward[v.Type()]
	if !ok {
		return 0, core.WrapError(fmt.Errorf("Type `%s` is not registered", v.Type()))
	}

	valbuf, err := protobuf.Encode(v.Interface())
	if err != nil {
		return 0, core.WrapError(err)
	}

	header := &tEntryHeader{
		Size: uint32(len(valbuf)),
		Type: t,
	}

	headbuf, err := protobuf.Encode(header)
	if err != nil {
		return 0, core.WrapError(err)
	}

	var out bytes.Buffer

	if err := out.WriteByte(byte(len(headbuf) - 1)); err != nil {
		return 0, core.WrapError(err)
	}

	if _, err := out.Write(headbuf); err != nil {
		return 0, core.WrapError(err)
	}

	if _, err := out.Write(valbuf); err != nil {
		return 0, core.WrapError(err)
	}

	sz, err := self.writer.Write(out.Bytes())
	if err != nil {
		return 0, core.WrapError(err)
	}

	return sz, nil
}

func MakeEncoder(writer io.Writer, registry *Registry) *Encoder {
	return &Encoder{
		writer:   writer,
		registry: registry.sub_registry(),
	}
}
