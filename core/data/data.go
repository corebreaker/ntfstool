package data

import "fmt"

type FileIndex uint64

func (fi FileIndex) String() string {
	return fmt.Sprintf("%012X (%d)", uint64(fi), uint64(fi))
}

type FileRef uint64

func (frn FileRef) IsNull() bool {
	return frn == 0
}

func (frn FileRef) GetSequenceNumber() uint16 {
	return uint16(frn >> 48)
}

func (frn FileRef) GetFileIndex() FileIndex {
	return FileIndex(frn & 0xFFFFFFFFFFFF)
}

func (frn FileRef) String() string {
	if frn.IsNull() {
		return "{NO FILE}"
	}

	return fmt.Sprintf("%04X / %s", frn.GetSequenceNumber(), frn.GetFileIndex())
}

func MakeFileRef(seq uint16, index FileIndex) FileRef {
	return FileRef((uint64(seq) << 48) | uint64(index))
}

type IDataRecord interface {
	fmt.Stringer

	IsNull() bool
	HasName() bool
	GetError() error
	GetPosition() int64
	GetEncodingCode() string
	GetName() string
	GetParent() FileRef
	GetLabel() string
	Print()
}

type IDataStream interface {
	Close() error
	SendRecord(uint, int64, IDataRecord)
	SendError(error)
}
