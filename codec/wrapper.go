package codec

type ValueWrapper struct {
    TypeID string
    Value  interface{}
}

func make_wrapper_from_value(value interface{}) (*ValueWrapper, error) {
    t := get_value_type(value)

    return &ValueWrapper{
        TypeID: backward_registry[t],
        Value:  value,
    }
}
