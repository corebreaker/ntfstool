package codec

import (
	"encoding/binary"
	"io"
	"reflect"

	"essai/ntfstool/core"
)

type tDescInt16 struct {
	tDescBase
}

func (d *tDescInt16) read_desc(r io.Reader) error  { return nil }
func (d *tDescInt16) write_desc(w io.Writer) error { return nil }
func (d *tDescInt16) make_desc(typ reflect.Type)   {}

func (d *tDescInt16) read_value(r io.Reader) (*reflect.Value, error) {
	var v int16

	if err := binary.Read(r, binary.BigEndian, &v); err != nil {
		return nil, core.WrapError(err)
	}

	return &reflect.ValueOf(v), nil
}

func (d *tDescInt16) write_value(w io.Writer, v reflect.Value) error {
	return core.WrapError(binary.Write(w, binary.BigEndian, int16(v.Int())))
}
