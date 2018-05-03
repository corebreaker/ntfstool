package codec

import (
	"encoding/binary"
	"io"
	"reflect"

	"essai/ntfstool/core"
)

type tDescBool struct {
	tDescBase
}

func (d *tDescBool) read_desc(r io.Reader) error  { return nil }
func (d *tDescBool) write_desc(w io.Writer) error { return nil }
func (d *tDescBool) make_desc(typ reflect.Type)   {}

func (d *tDescBool) read_value(r io.Reader) (*reflect.Value, error) {
	var v bool

	if err := binary.Read(r, binary.BigEndian, &v); err != nil {
		return nil, core.WrapError(err)
	}

	return &reflect.ValueOf(v), nil
}

func (d *tDescBool) write_value(w io.Writer, v reflect.Value) error {
	return core.WrapError(binary.Write(w, binary.BigEndian, v.Bool()))
}
