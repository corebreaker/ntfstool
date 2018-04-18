package core

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
)

var (
	stringer_iface = reflect.TypeOf((*fmt.Stringer)(nil)).Elem()
)

func StructSize(value interface{}) int {
	return binary.Size(value)
}

func is_nil(v reflect.Value) bool {
	defer DiscardPanic()

	return (!v.IsValid()) || v.IsNil()
}

func print_value(indent, prefix string, w io.Writer, v reflect.Value) {
	if is_nil(v) {
		fmt.Fprintln(w, indent, prefix+"<nil>")

		return
	}

	if v.Type().Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if is_nil(v) {
		fmt.Fprintln(w, indent, prefix+"<nil>")

		return
	}

	t := v.Type()

	switch t.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			sz := StringSize(prefix)
			padding := ""
			if sz > 0 {
				padding = strings.Repeat(" ", sz)
			}

			f := t.Field(i)
			val := v.Field(i)

			ftyp := f.Type
			if (ftyp.Kind() == reflect.Interface) && (!val.IsNil()) {
				val = val.Elem()
				ftyp = val.Type()
			}

			switch ftyp.Kind() {
			case reflect.Ptr, reflect.Interface:
				if !val.IsNil() {
					ftyp = ftyp.Elem()
				}
			}

			switch {
			case f.Anonymous || (ftyp.Kind() == reflect.Struct):
				fv := val

				if fv.Type().Kind() == reflect.Ptr {
					fv = fv.Elem()
				}

				if (!f.Anonymous) || (fv.Type().Size() > 0) {
					fmt.Fprintln(w, indent, prefix+f.Name, ":")
					print_value(indent+padding+"   ", "", w, val)
				}

			case f.Type.Implements(stringer_iface):
				fmt.Fprintln(w, indent, prefix+f.Name, ":", val.Interface().(fmt.Stringer).String())

			case f.Type.Kind() == reflect.Slice:
				fmt.Fprintln(w, indent, prefix+f.Name, ":")
				sz := val.Len()
				for i := 0; i < sz; i++ {
					print_value(indent+padding+"   ", "- ", w, val.Index(i))
				}

			case f.Type.Kind() == reflect.Map:
				fmt.Fprintln(w, indent, prefix+f.Name, ":")
				print_value(indent+padding+"   ", "", w, val)

			default:
				fmt.Fprintln(w, indent, prefix+f.Name, ":", val.Interface())
			}

			prefix = padding
		}

	case reflect.Map:
		for _, key := range v.MapKeys() {
			sz := StringSize(prefix)
			padding := ""
			if sz > 0 {
				padding = strings.Repeat(" ", sz)
			}

			val := v.MapIndex(key)
			t := val.Type()
			switch {
			case t.Kind() == reflect.Struct:
				fmt.Fprintln(w, indent, fmt.Sprint(prefix, key), "=>")
				print_value(indent+padding+"   ", "", w, val)

			case t.Implements(stringer_iface):
				fmt.Fprintln(w, indent, fmt.Sprint(prefix, key), "=>", val.Interface().(fmt.Stringer).String())

			case t.Kind() == reflect.Slice:
				fmt.Fprintln(w, indent, fmt.Sprint(prefix, key), "=>")
				sz := val.Len()
				for i := 0; i < sz; i++ {
					print_value(indent+padding+"   ", "- ", w, val.Index(i))
				}

			case t.Kind() == reflect.Map:
				fmt.Fprintln(w, indent, fmt.Sprint(prefix, key), "=>")
				print_value(indent+padding+"   ", "", w, val)

			default:
				fmt.Fprintln(w, indent, fmt.Sprint(prefix, key), "=>", val.Interface())
			}

			prefix = padding
		}

	default:
		fmt.Fprintln(w, indent, prefix+fmt.Sprint(v.Interface()))
	}
}

func FprintStruct(w io.Writer, value interface{}) {
	if value == nil {
		fmt.Fprintln(w, "   <nil>")

		return
	}

	print_value("  ", "", w, reflect.ValueOf(value))
}

func PrintStruct(value interface{}) {
	FprintStruct(os.Stdout, value)
}
