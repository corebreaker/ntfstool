package file

import (
	"fmt"

	"essai/ntfstool/core"

	"github.com/DeDiS/protobuf"
)

type tFileIndex struct {
	Logical  int64
	Physical int64
}

func (fidx *tFileIndex) String() string {
	return fmt.Sprintf("{FILE-IDX: LOG=%d PHYS=%d}", fidx.Logical, fidx.Physical)
}

type tFileDescRecord struct {
	Count    uint32
	Counts   map[string]uint32
	Headers  []int32
	Indexes  []*tFileIndex
	Sep      string
	Position int64
}

type tFileDesc struct {
	BaseDataRecord

	Trailer  bool
	Count    uint32
	Counts   map[string]uint32
	Headers  []int16
	Indexes  []*tFileIndex
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

	sep := ""
	if d.Trailer {
		sep = "FILE-END"
	}

	res, err := protobuf.Encode(&tFileDescRecord{
		Count:    d.Count,
		Counts:   d.Counts,
		Headers:  headers,
		Indexes:  d.Indexes,
		Sep:      sep,
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
		Trailer:  rec.Sep == "FILE-END",
		Count:    rec.Count,
		Counts:   rec.Counts,
		Headers:  headers,
		Indexes:  rec.Indexes,
		Position: rec.Position,
	}

	return nil
}
