package codec

import (
	"fmt"
	"io"
	"reflect"

	"github.com/DeDiS/protobuf"

	"essai/ntfstool/core"
)

type tDecoder struct {
	dec *Decoder
}

func (self *tDecoder) Decode() (interface{}, error) {
	var res interface{}

	if _, err := self.dec.Decode(&res); err != nil {
		return nil, err
	}

	return res, nil
}

type Decoder struct {
	reader   io.Reader
	registry *Registry
}

func (self *Decoder) ToCoreDecoder() core.IDecoder {
	return &tDecoder{self}
}

func (self *Decoder) Decode(val interface{}) (int, error) {
	v := reflect.ValueOf(val)
	if v.Kind() != reflect.Ptr {
		return 0, core.WrapError(fmt.Errorf("value must be a pointer"))
	}

	var szbuf [1]byte

	n1, err := self.reader.Read(szbuf[:])
	if err != nil {
		return 0, core.WrapError(err)
	}

	headbuf := make([]byte, szbuf[0]+1)

	n2, err := self.reader.Read(headbuf)
	if err != nil {
		return 0, core.WrapError(err)
	}

	var header tEntryHeader

	if err := protobuf.Decode(headbuf, &header); err != nil {
		return 0, core.WrapError(err)
	}

	valbuf := make([]byte, header.Size)

	t, ok := self.registry.foreward[header.Type]
	if !ok {
		return 0, core.WrapError(fmt.Errorf("Unknown decoded type: `%s`", header.Type))
	}

	n3, err := self.reader.Read(valbuf)
	if err != nil {
		return 0, core.WrapError(err)
	}

    rec := &tRecord{
        Value: reflect.New(t).Interface(),
    }

	if err := protobuf.Decode(valbuf, rec); err != nil {
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

		v.Elem().Set(reflect.ValueOf(rec.Value).Elem())
	}()

	if err != nil {
		return 0, err
	}

	return n1 + n2 + n3, nil
}

func MakeDecoder(reader io.Reader, registry *Registry) *Decoder {
	return &Decoder{
		reader:   reader,
		registry: registry.SubRegistry(),
	}
}
