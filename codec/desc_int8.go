package codec

import (
	"encoding/binary"
	"io"
	"reflect"

	"essai/ntfstool/core"
)

type tDescInt8 struct {
	tDescBase
}

func (d *tDescInt8) read_desc(r io.Reader) error  { return nil }
func (d *tDescInt8) write_desc(w io.Writer) error { return nil }
func (d *tDescInt8) make_desc(typ reflect.Type)   {}

func (d *tDescInt8) read_value(r io.Reader) (*reflect.Value, error) {
	var v int8

	if err := binary.Read(r, binary.BigEndian, &v); err != nil {
		return nil, core.WrapError(err)
	}

	return &reflect.ValueOf(v), nil
}

func (d *tDescInt8) write_value(w io.Writer, v reflect.Value) error {
	return core.WrapError(binary.Write(w, binary.BigEndian, int8(v.Int())))
}
