package codec

import (
	"encoding/binary"
	"io"
	"reflect"

	"essai/ntfstool/core"
)

type tDescInt struct {
	tDescBase
}

func (d *tDescInt) read_desc(r io.Reader) error  { return nil }
func (d *tDescInt) write_desc(w io.Writer) error { return nil }
func (d *tDescInt) make_desc(typ reflect.Type)   {}

func (d *tDescInt) read_value(r io.Reader) (*reflect.Value, error) {
	var v int64

	if err := binary.Read(r, binary.BigEndian, &v); err != nil {
		return nil, core.WrapError(err)
	}

	return &reflect.ValueOf(int(v)), nil
}

func (d *tDescInt) write_value(w io.Writer, v reflect.Value) error {
	return core.WrapError(binary.Write(w, binary.BigEndian, v.Int()))
}
