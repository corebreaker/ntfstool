package codec

import (
	"fmt"
	"io"

	"github.com/DeDiS/protobuf"

	"essai/ntfstool/core"
)

type Encoder struct {
	writer   io.Writer
	registry *Registry
}

func (self *Encoder) Encode(val interface{}) (int, error) {
	t := get_value_type(val)
	name, ok := self.registry.backward[t]
	if !ok {
		return 0, core.WrapError(fmt.Errorf("Type `%s` is not registered", t))
	}

	buf, err := protobuf.Encode(val)
	if err != nil {
		return 0, core.WrapError(err)
	}

	blen := len(buf)
	infos := tEntryHeader{
		Size: uint32(blen),
		Type: name,
	}

	header, err := protobuf.Encode(&infos)
	if err != nil {
		return 0, core.WrapError(err)
	}

	hlen := len(header)
	out := make([]byte, 1+hlen+blen)

	out[0] = byte(hlen - 1)
	copy(out[1:(hlen+1)], header)
	copy(out[(hlen+1):], buf)

	olen, err := self.writer.Write(out)
	if err != nil {
		return 0, core.WrapError(err)
	}

	return olen, nil
}

func MakeEncoder(writer io.Writer, registry *Registry) *Encoder {
	return &Encoder{
		writer:   writer,
		registry: registry,
	}
}
