package core

import (
    "encoding/binary"
    "fmt"
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
    defer func() {
        recover()
    }()

    return (!v.IsValid()) || v.IsNil()
}

func print_value(indent, prefix string, v reflect.Value) {
    if is_nil(v) {
        fmt.Println(indent, prefix+"<nil>")

        return
    }

    if v.Type().Kind() == reflect.Ptr {
        v = v.Elem()
    }

    if is_nil(v) {
        fmt.Println(indent, prefix+"<nil>")

        return
    }

    t := v.Type()

    if t.Kind() == reflect.Struct {
        for i := 0; i < v.NumField(); i++ {
            sz := StringSize(prefix)
            padding := ""
            if sz > 0 {
                padding = strings.Repeat(" ", sz)
            }

            f := t.Field(i)
            val := v.Field(i)
            switch {
            case f.Anonymous || (f.Type.Kind() == reflect.Struct):
                fmt.Println(indent, prefix+f.Name, ":")
                print_value(indent+padding+"   ", "", val)

            case f.Type.Implements(stringer_iface):
                fmt.Println(indent, prefix+f.Name, ":", val.Interface().(fmt.Stringer).String())

            case f.Type.Kind() == reflect.Slice:
                sz := val.Len()
                for i := 0; i < sz; i++ {
                    print_value(indent+padding+"   ", "- ", val.Index(i))
                }

            default:
                fmt.Println(indent, prefix+f.Name, ":", val.Interface())
            }

            prefix = padding
        }
    } else {
        fmt.Println(indent, prefix+fmt.Sprint(v.Interface()))
    }
}

func PrintStruct(value interface{}) {
    if value == nil {
        fmt.Println("   <nil>")

        return
    }

    print_value("  ", "", reflect.ValueOf(value))
}
