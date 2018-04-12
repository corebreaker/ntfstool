package core

import (
	"fmt"
	"reflect"

	"essai/ntfstool/core/data"
)

var recordType = reflect.TypeOf((*data.IDataRecord)(nil)).Elem()

type IEncoder interface {
	Encode(interface{}) error
}

func WriteRecord(encoder IEncoder, record data.IDataRecord) error {
	return encoder.Encode(record)
}

type IDecoder interface {
	Decode() (interface{}, error)
}

func ReadRecord(decoder IDecoder) (data.IDataRecord, error) {
	val, err := decoder.Decode()
	if err != nil {
		return nil, err
	}

	v := reflect.ValueOf(val)
	t := v.Type()
	if (!t.Implements(recordType)) && (t.Kind() != reflect.Ptr) {
		if v.CanAddr() {
			val = v.Addr().Interface()
		} else {
			p := reflect.New(t)
			p.Elem().Set(v)

			val = p.Interface()
		}
	}

	res, ok := val.(data.IDataRecord)
	if !ok {
		return nil, WrapError(fmt.Errorf("`%s` is not a data record", reflect.TypeOf(val)))
	}

	return res, nil
}
