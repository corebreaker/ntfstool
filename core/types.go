package core

import (
	"fmt"
	"time"

	"essai/ntfstool/core/dataio"
)

type Usn uint64
type Char uint16

type Boolean byte

func (b Boolean) Value() bool {
	return b != BOOL_FALSE
}

const (
	BOOL_FALSE Boolean = iota
	BOOL_TRUE
)

type FileRecordData [976]byte

func (FileRecordData) String() string {
	return "<Datas>"
}

type Byte byte

func (self Byte) String() string {
	return fmt.Sprintf("%02x", byte(self))
}

type FileReferenceNumber uint64

func (frn FileReferenceNumber) IsNull() bool {
	return frn == 0
}

func (frn FileReferenceNumber) GetSequenceNumber() uint16 {
	return uint16(frn >> 48)
}

func (frn FileReferenceNumber) GetFileIndex() dataio.FileIndex {
	return dataio.FileIndex(frn & 0xFFFFFFFFFFFF)
}

func (frn FileReferenceNumber) String() string {
	if frn.IsNull() {
		return "{NO FILE}"
	}

	return fmt.Sprintf("%04X / %s", frn.GetSequenceNumber(), frn.GetFileIndex())
}

func MakeFileReferenceNumber(seq uint16, index dataio.FileIndex) FileReferenceNumber {
	return FileReferenceNumber((uint64(seq) << 48) | uint64(index))
}

type ClusterNumber uint64

func (self ClusterNumber) GetPosition(d *DiskIO) int64 {
	return d.GetOffset() + (int64(self) * 8)
}

func (self ClusterNumber) String() string {
	v := uint64(self)

	if v == 0 {
		return "<Null cluster>"
	}

	return fmt.Sprintf("%016X (%d)", v, v)
}

type Timestamp uint64

func (self Timestamp) Decode() (sec, nano int64) {
	val := uint64(self)

	return int64((val / 10000000) - 11644473600), int64(val%10000000) * 100
}

func (self Timestamp) Time() time.Time {
	return time.Unix(self.Decode())
}

func (self Timestamp) String() string {
	if self == 0 {
		return "<No time>"
	}

	return fmt.Sprint(self.Time())
}
