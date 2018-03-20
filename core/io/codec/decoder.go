package codec

import (
	"fmt"
	"io"
	"reflect"

	"github.com/DeDiS/protobuf"

	"essai/ntfstool/core"
)

type Decoder struct {
	reader   io.ReaderAt
	registry *Registry
}

func (self *Decoder) Decode(at int64, val interface{}) (int, error) {
	v := reflect.ValueOf(val)
	if v.Kind() != reflect.Ptr {
		return 0, core.WrapError(fmt.Errorf("value must be a pointer"))
	}

	var szbuf [1]byte

	n1, err := self.reader.ReadAt(szbuf[:], at)
	if err != nil {
		return 0, core.WrapError(err)
	}

	at += int64(n1)
	headbuf := make([]byte, szbuf[0])

	n2, err := self.reader.ReadAt(headbuf, at)
	if err != nil {
		return 0, core.WrapError(err)
	}

	var header tEntryHeader

	if err := protobuf.Decode(headbuf, &header); err != nil {
		return 0, core.WrapError(err)
	}

	at += int64(n2)
	valbuf := make([]byte, header.Size)

	t, ok := self.registry.foreward[header.Type]
	if !ok {
		return 0, core.WrapError(fmt.Errorf("Unknown decoded type: `%s`", header.Type))
	}

	n3, err := self.reader.ReadAt(valbuf, at)
	if err != nil {
		return 0, core.WrapError(err)
	}

	res := reflect.New(t)

	if err := protobuf.Decode(valbuf, res.Interface()); err != nil {
		return 0, core.WrapError(err)
	}

	func() {
		defer func() {
			errv := recover()
			if errv == nil {
				return
			}

			erri, ok := errv.(error)
			if !ok {
				erri = fmt.Errorf("Error: %s", errv)
			}

			err = core.WrapError(erri)
		}()

		v.Elem().Set(res.Elem())
	}()

	if err != nil {
		return 0, err
	}

	return n1 + n2 + n3, nil
}

func MakeDecoder(reader io.ReaderAt, registry *Registry) *Decoder {
	return &Decoder{
		reader:   reader,
		registry: registry.sub_registry(),
	}
}
