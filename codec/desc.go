package codec

import (
	"bufio"
	"io"
	"reflect"
)

type iTypeDesc interface {
	type_name() string
	desc_name() string

	read_desc(*bufio.Reader) error
	write_desc(*bufio.Writer) error
	make_desc(typ reflect.Type)

	read_value(*bufio.Reader) (*reflect.Value, error)
	write_value(*bufio.Writer, reflect.Value) error
}

type tDescBase struct {
	_type_name string
	_desc_name string
}

func (db *tDescBase) type_name() string { return db._type_name }
func (db *tDescBase) desc_name() string { return db._desc_name }

func make_desc(typ reflect.Type) iTypeDesc {
}

func parse_desc(typ reflect.Type) iTypeDesc {
	name := typ.Name()
	res, ok := desc_registry[name]
	if ok {
		return res
	}

	res = make_desc(typ)
	desc_registry[name] = res

	return res
}

func read_desc_typename(r io.Reader) (iTypeDesc, error) {
	return nil, nil
}

func write_desc_typename(w io.Writer, desc iTypeDesc) error {
	return nil
}

func read_desc(r io.Reader) (iTypeDesc, error) {
	return nil, nil
}

func write_desc(w io.Writer, desc iTypeDesc) error {
	return nil
}
