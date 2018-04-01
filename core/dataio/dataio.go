package dataio

import "fmt"

type FileIndex uint64

func (fi FileIndex) String() string {
	return fmt.Sprintf("%012X (%d)", uint64(fi), uint64(fi))
}

type IDataRecord interface {
	fmt.Stringer

	IsNull() bool
	HasName() bool
	GetError() error
	GetPosition() int64
	GetEncodingCode() string
	GetName() string
	GetParentIndex() FileIndex
	GetLabel() string
	Print()
}

type IDataStream interface {
	Close() error
	SendRecord(uint, IDataRecord)
	SendError(error)
}
