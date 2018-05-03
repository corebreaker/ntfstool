package codec

import (
	"encoding/binary"
	"io"
	"reflect"

	"essai/ntfstool/core"
)

type tDescUint64 struct {
	tDescBase
}

func (d *tDescUint64) read_desc(r io.Reader) error  { return nil }
func (d *tDescUint64) write_desc(w io.Writer) error { return nil }
func (d *tDescUint64) make_desc(typ reflect.Type)   {}

func (d *tDescUint64) read_value(r io.Reader) (*reflect.Value, error) {
	var v uint64

	if err := binary.Read(r, binary.BigEndian, &v); err != nil {
		return nil, core.WrapError(err)
	}

	return &reflect.ValueOf(v), nil
}

func (d *tDescUint64) write_value(w io.Writer, v reflect.Value) error {
	return core.WrapError(binary.Write(w, binary.BigEndian, v.Uint()))
}
