package datafile

type BaseDataRecord struct{}

func (*BaseDataRecord) IsNull() bool            { return false }
func (*BaseDataRecord) GetError() error         { return nil }
func (*BaseDataRecord) GetPosition() int64      { return 0 }
func (*BaseDataRecord) GetEncodingCode() string { return "" }
func (*BaseDataRecord) String() string          { return "{NONE}" }

type tNullRecord struct {
	BaseDataRecord

	//zero int64
}

func (*tNullRecord) IsNull() bool            { return true }
func (*tNullRecord) GetError() error         { return nil }
func (*tNullRecord) GetPosition() int64      { return 0 }
func (*tNullRecord) GetEncodingCode() string { return "Z" }
func (*tNullRecord) String() string          { return "{NULL}" }
func (*tNullRecord) Print()                  { fmt.Println("{NULL}") }
