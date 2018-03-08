package codec

import (
	"encoding/binary"
	"io"
	"reflect"

	"essai/ntfstool/core"
)

type tDescUintptr struct {
	tDescBase
}

func (d *tDescUintptr) read_desc(r io.Reader) error  { return nil }
func (d *tDescUintptr) write_desc(w io.Writer) error { return nil }
func (d *tDescUintptr) make_desc(typ reflect.Type)   {}

func (d *tDescUintptr) read_value(r io.Reader) (*reflect.Value, error) {
	var v uint64

	if err := binary.Read(r, binary.BigEndian, &v); err != nil {
		return nil, core.WrapError(err)
	}

	return &reflect.ValueOf(uintptr(v)), nil
}

func (d *tDescUintptr) write_value(w io.Writer, v reflect.Value) error {
	return core.WrapError(binary.Write(w, binary.BigEndian, uintptr(v.Uint())))
}
