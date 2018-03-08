package core

import (
	"fmt"
	"reflect"
)

type IEncoder interface {
	Encode(interface{}) error
}

func WriteRecord(encoder IEncoder, record IDataRecord) error {
	return encoder.Encode(record)
}

type IDecoder interface {
	Decode() (interface{}, error)
}

func ReadRecord(decoder IDecoder) (IDataRecord, error) {
	val, err := decoder.Decode()
	if err != nil {
		return nil, err
	}

	res, ok := val.(IDataRecord)
	if !ok {
		return nil, WrapError(fmt.Errorf("`%s` is not a data record", reflect.TypeOf(val)))
	}

	return res, nil
}
