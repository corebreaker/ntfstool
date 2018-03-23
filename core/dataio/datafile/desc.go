package datafile

import (
	"fmt"

	"essai/ntfstool/core/dataio"
)

type tFileIndex struct {
	Logical  int64
	Physical int64
}

type tFileDescRecord struct {
	BaseDataRecord

	Count    uint32
	Headers  []int32
	Indexes  []tFileIndex
	Sep      string
	Position int64
}

func (d *tFileDescRecord) GetPosition() int64      { return d.Position }
func (d *tFileDescRecord) GetEncodingCode() string { return "$FILEDESC" }
func (d *tFileDescRecord) String() string          { return fmt.Sprintf("{DESC:%d}", d.GetPosition()) }
func (d *tFileDescRecord) Print()                  { fmt.Println(d.String()) }

type tFileDesc struct {
	Count    uint32
	Headers  []int16
	Indexes  []tFileIndex
	Position int64
}

func (d *tFileDesc) GetRecord() dataio.IDataRecord {
	headers := make([]int32, len(d.Headers))

	for i, h := range d.Headers {
		headers[i] = int32(h)
	}

	return &tFileDescRecord{
		Count:    d.Count,
		Headers:  headers,
		Indexes:  d.Indexes,
		Sep:      "FILE-END",
		Position: d.Position,
	}
}

func (d *tFileDesc) FromRecord(data dataio.IDataRecord) {
	rec, ok := data.(*tFileDescRecord)
	if !ok {
		return
	}

	headers := make([]int16, len(rec.Headers))

	for i, h := range rec.Headers {
		headers[i] = int16(h)
	}

	*d = tFileDesc{
		Count:    rec.Count,
		Headers:  headers,
		Indexes:  rec.Indexes,
		Position: rec.Position,
	}
}
