package codec

type tRecord struct {
	Value interface{}
}

type tEntryHeader struct {
	Size uint32
	Type string
}
