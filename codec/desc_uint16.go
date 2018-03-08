package codec

import (
	"encoding/binary"
	"io"
	"reflect"

	"essai/ntfstool/core"
)

type tDescUint16 struct {
	tDescBase
}

func (d *tDescUint16) read_desc(r io.Reader) error  { return nil }
func (d *tDescUint16) write_desc(w io.Writer) error { return nil }
func (d *tDescUint16) make_desc(typ reflect.Type)   {}

func (d *tDescUint16) read_value(r io.Reader) (*reflect.Value, error) {
	var v uint16

	if err := binary.Read(r, binary.BigEndian, &v); err != nil {
		return nil, core.WrapError(err)
	}

	return &reflect.ValueOf(v), nil
}

func (d *tDescUint16) write_value(w io.Writer, v reflect.Value) error {
	return core.WrapError(binary.Write(w, binary.BigEndian, uint16(v.Uint())))
}
