package codec

import (
	"encoding/binary"
	"io"
	"reflect"

	"essai/ntfstool/core"
)

type tDescUint struct {
	tDescBase
}

func (d *tDescUint) read_desc(r io.Reader) error  { return nil }
func (d *tDescUint) write_desc(w io.Writer) error { return nil }
func (d *tDescUint) make_desc(typ reflect.Type)   {}

func (d *tDescUint) read_value(r io.Reader) (*reflect.Value, error) {
	var v uint64

	if err := binary.Read(r, binary.BigEndian, &v); err != nil {
		return nil, core.WrapError(err)
	}

	return &reflect.ValueOf(uint(v)), nil
}

func (d *tDescUint) write_value(w io.Writer, v reflect.Value) error {
	return core.WrapError(binary.Write(w, binary.BigEndian, v.Uint()))
}
