package core

import (
	"fmt"
	"time"
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

type DataZone []byte

func (dz DataZone) String() string {
	return fmt.Sprintf("<Data:%d>", len(dz))
}
