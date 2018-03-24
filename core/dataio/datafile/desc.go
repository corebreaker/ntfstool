package datafile

import (
	"fmt"

	"essai/ntfstool/core"

	"github.com/DeDiS/protobuf"
)

type tFileIndex struct {
	Logical  int64
	Physical int64
}

type tFileDescRecord struct {
	Count    uint32
	Headers  []int32
	Indexes  []tFileIndex
	Sep      string
	Position int64
}

type tFileDesc struct {
	BaseDataRecord

	Count    uint32
	Headers  []int16
	Indexes  []tFileIndex
	Position int64
}

func (d *tFileDesc) GetPosition() int64      { return d.Position }
func (d *tFileDesc) GetEncodingCode() string { return "$FILEDESC" }
func (d *tFileDesc) String() string          { return fmt.Sprintf("{DESC:%d}", d.GetPosition()) }
func (d *tFileDesc) Print()                  { fmt.Println(d.String()) }

func (d *tFileDesc) MarshalBinary() (data []byte, err error) {
	headers := make([]int32, len(d.Headers))

	for i, h := range d.Headers {
		headers[i] = int32(h)
	}

	res, err := protobuf.Encode(&tFileDescRecord{
		Count:    d.Count,
		Headers:  headers,
		Indexes:  d.Indexes,
		Sep:      "FILE-END",
		Position: d.Position,
	})

	if err != nil {
		return nil, core.WrapError(err)
	}

	return res, nil
}

func (d *tFileDesc) UnmarshalBinary(data []byte) error {
	var rec tFileDescRecord

	if err := protobuf.Decode(data, &rec); err != nil {
		return core.WrapError(err)
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

	return nil
}
