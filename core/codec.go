package core

import (
	"fmt"
	"reflect"

    "essai/ntfstool/core/dataio"
)

type IEncoder interface {
	Encode(interface{}) error
}

func WriteRecord(encoder IEncoder, record dataio.IDataRecord) error {
	return encoder.Encode(record)
}

type IDecoder interface {
	Decode() (interface{}, error)
}

func ReadRecord(decoder IDecoder) (dataio.IDataRecord, error) {
	val, err := decoder.Decode()
	if err != nil {
		return nil, err
	}

	res, ok := val.(dataio.IDataRecord)
	if !ok {
		return nil, WrapError(fmt.Errorf("`%s` is not a data record", reflect.TypeOf(val)))
	}

	return res, nil
}
