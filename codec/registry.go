package codec

import (
    "reflect"
)

var (
    foreward_registry map[string]reflect.Type = make(map[string]reflect.Type)
    backward_registry map[reflect.Type]string = make(map[reflect.Type]string)
)

func get_value_type(value interface{}) reflect.Type {
    res := reflect.ValueOf(value).Type()
    if res.Kind() == reflect.Ptr {
        res = t.Elem()
    }

    return res
}

func UnRegisterName(name string) {
    delete(foreward_registry, name)
}

func RegisterName(name string, value interface{}) {
    t := get_value_type(value)

    foreward_registry[name] = t
    backward_registry[t] = name
}

func Register(value interface{}) {
    t := get_value_type(value)
    name := t.Name()

    foreward_registry[name] = t
    backward_registry[t] = name
}
