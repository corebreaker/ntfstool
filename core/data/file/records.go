package file

import (
	"fmt"

	"essai/ntfstool/core/data"
)

type BaseDataRecord struct{}

func (*BaseDataRecord) IsNull() bool            { return false }
func (*BaseDataRecord) HasName() bool           { return false }
func (*BaseDataRecord) GetError() error         { return nil }
func (*BaseDataRecord) GetPosition() int64      { return 0 }
func (*BaseDataRecord) GetEncodingCode() string { return "" }
func (*BaseDataRecord) String() string          { return "{NONE}" }
func (*BaseDataRecord) GetName() string         { return "" }
func (*BaseDataRecord) GetLabel() string        { return "Other Records" }
func (*BaseDataRecord) GetParent() data.FileRef { return 0 }
func (*BaseDataRecord) GetIndex() int           { return 0 }
func (*BaseDataRecord) SetIndex(int)            {}

type tNullRecord struct {
	BaseDataRecord

	Zero bool
}

func (*tNullRecord) IsNull() bool            { return true }
func (*tNullRecord) GetError() error         { return nil }
func (*tNullRecord) GetPosition() int64      { return 0 }
func (*tNullRecord) GetEncodingCode() string { return "$Z" }
func (*tNullRecord) GetLabel() string        { return "Null Record" }
func (*tNullRecord) String() string          { return "{NULL}" }
func (*tNullRecord) Print()                  { fmt.Println("{NULL}") }

type tErrorRecord struct {
	BaseDataRecord

	err error
}

func (r *tErrorRecord) GetError() error { return r.err }
func (r *tErrorRecord) String() string  { return r.err.Error() }
func (r *tErrorRecord) Error() string   { return r.err.Error() }
func (r *tErrorRecord) Print()          { fmt.Println(r.err) }
