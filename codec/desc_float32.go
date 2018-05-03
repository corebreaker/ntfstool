package codec

import (
	"encoding/binary"
	"io"
	"reflect"

	"essai/ntfstool/core"
)

type tDescFloat32 struct {
	tDescBase
}

func (d *tDescFloat32) read_desc(r io.Reader) error  { return nil }
func (d *tDescFloat32) write_desc(w io.Writer) error { return nil }
func (d *tDescFloat32) make_desc(typ reflect.Type)   {}

func (d *tDescFloat32) read_value(r io.Reader) (*reflect.Value, error) {
	var v float32

	if err := binary.Read(r, binary.BigEndian, &v); err != nil {
		return nil, core.WrapError(err)
	}

	return &reflect.ValueOf(v), nil
}

func (d *tDescFloat32) write_value(w io.Writer, v reflect.Value) error {
	return core.WrapError(binary.Write(w, binary.BigEndian, float32(v.Float())))
}
