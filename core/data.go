package core

import "fmt"

type IDataRecord interface {
	fmt.Stringer

	IsNull() bool
	GetError() error
	GetPosition() int64
	GetEncodingCode() string
	Print()
}

type IDataStream interface {
	Close() error
	SendRecord(IDataRecord)
	SendError(error)
}
