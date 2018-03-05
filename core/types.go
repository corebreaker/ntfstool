package core

import (
    "fmt"
    "time"
)

type Usn uint64
type Char uint16

type Boolean byte

const (
    BOOL_FALSE Boolean = iota
    BOOL_TRUE
)

type FileRecordData [982]byte

func (FileRecordData) String() string {
    return "<Datas>"
}

type Byte byte

func (self Byte) String() string {
    return fmt.Sprintf("%02x", byte(self))
}

type FileReferenceNumber uint64

func (self FileReferenceNumber) GetSequenceNumber() uint16 {
    return uint16(self >> 48)
}

func (self FileReferenceNumber) GetFileIndex() int64 {
    return int64(self & 0xFFFFFFFFFFFF)
}

func (self FileReferenceNumber) String() string {
    idx := self.GetFileIndex()

    return fmt.Sprintf("%04X / %012X (%d)", self.GetSequenceNumber(), idx, idx)
}

type ClusterNumber uint64

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
