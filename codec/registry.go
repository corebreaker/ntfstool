package codec

import (
	"fmt"
	"reflect"
	"strings"
)

func get_value_type(value interface{}) reflect.Type {
	val := reflect.ValueOf(value)

loop:
	for {
		switch val.Kind() {
		case reflect.Ptr:
			val = val.Elem()

		case reflect.Interface:
			val = val.Elem()

		default:
			break loop
		}
	}

	return val.Type()
}

func get_registered_name(name string) string {
	name = strings.Replace(name, "\\", "\\\\")
	if name[0] == '#' {
		name = "\\#"
	}

	return name
}

type Registry struct {
	foreward map[string]reflect.Type
	backward map[reflect.Type]string
}

func (r *Registry) register_value(name string, value interface{}) {
	r.register(name, get_value_type(value))
}

func (r *Registry) register_type(typ reflect.Type) {
	t := get_value_type(value)
	r.register_value("#"+t.Name(), t)
}

func (r *Registry) register(name string, typ reflect.Type) {
	r.foreward[name] = typ
	r.backward[t] = name
}

func (r *Registry) unregister(name string) {
	t, ok := r.foreward[name]
	if !ok {
		return
	}

	delete(r.foreward, name)
	delete(r.backward, t)
}

func (r *Registry) RegisterName(name string, value interface{}) {
	r.register_value(get_registered_name(name), value)
}

func (r *Registry) UnRegisterName(name string) {
	r.unregister(get_registered_name(name))
}

func (r *Registry) RegisterValue(value interface{}) {
	t := get_value_type(value)
	r.register_value(t.Name(), t)
}

func (r *Registry) UnregisterValue(value interface{}) {
	t := get_value_type(value)
	name, ok := r.backward[t]
	if !ok {
		return
	}

	delete(r.foreward, name)
	delete(r.backward, t)
}

func (r *Registry) Names() []string {
	var res []string

	for name := range r.foreward {
		if name[0] != '#' {
			res = append(res, name)
		}
	}

	return res
}

func (r *Registry) sub_registry() *Registry {
	foreward := make(map[string]reflect.Type)
	backward := make(map[reflect.Type]string)

	for k, v := range r.foreward {
		foreward[k] = v
		backward[v] = k
	}

	return &Registry{
		foreward: foreward,
		backward: backward,
	}
}

func MakeRegistry() *Registry {
	return &Registry{
		foreward: make(map[string]reflect.Type),
		backward: make(map[reflect.Type]string),
	}
}
