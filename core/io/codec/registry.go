package codec

import (
	"reflect"
	"strings"
)

func normalize_value(value interface{}) reflect.Value {
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

	return val
}

func get_value_type(value interface{}) reflect.Type {
	return normalize_value(value).Type()
}

func get_registered_name(name string) string {
	name = strings.Replace(name, "\\", "\\\\", -1)
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

func (r *Registry) register(name string, typ reflect.Type) {
	r.foreward[name] = typ
	r.backward[typ] = name
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
	r.register_value("#"+t.Name(), t)
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
