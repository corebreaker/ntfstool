package inspect

import (
	"fmt"
	"os"
	"reflect"
	"sort"

	"github.com/corebreaker/ntfstool/core"
	"github.com/corebreaker/ntfstool/core/data"
	datafile "github.com/corebreaker/ntfstool/core/data/file"

	"github.com/DeDiS/protobuf"
)

const STATE_FORMAT_NAME = "States"

type StateRecordType uint8

const (
	STATE_RECORD_TYPE_NONE StateRecordType = iota
	STATE_RECORD_TYPE_ERROR
	STATE_RECORD_TYPE_FILE
	STATE_RECORD_TYPE_INDEX
	STATE_RECORD_TYPE_MFT
)

type IStateRecord interface {
	data.IDataRecord

	GetType() StateRecordType
	GetHeader() *core.RecordHeader
	GetMftId() string
	SetMft(state *StateMft)
	GetAttributeDesc(attr *StateAttribute) (*core.AttributeDesc, error)
	GetAttribute(pos int64) *StateAttribute
	GetAttributes(attr core.AttributeType, others ...core.AttributeType) []*StateAttribute
	Init(disk *core.DiskIO) (bool, error)
}

type tNoneState struct {
	datafile.BaseDataRecord
}

func (self *tNoneState) GetType() StateRecordType           { return STATE_RECORD_TYPE_NONE }
func (self *tNoneState) GetHeader() *core.RecordHeader      { return nil }
func (self *tNoneState) GetMftId() string                   { return "" }
func (self *tNoneState) SetMft(*StateMft)                   {}
func (self *tNoneState) Init(*core.DiskIO) (bool, error)    { return true, nil }
func (self *tNoneState) GetAttribute(int64) *StateAttribute { return nil }

func (self *tNoneState) GetAttributeDesc(*StateAttribute) (*core.AttributeDesc, error) {
	return nil, core.WrapError(fmt.Errorf("This state has no attribute"))
}

func (self *tNoneState) GetAttributes(core.AttributeType, ...core.AttributeType) []*StateAttribute {
	return nil
}

type tStateError struct {
	tNoneState

	err error
}

func (self *tStateError) GetType() StateRecordType { return STATE_RECORD_TYPE_ERROR }
func (self *tStateError) GetError() error          { return self.err }
func (self *tStateError) Print()                   { fmt.Println("Error:", self.err) }

type StateBase struct {
	tNoneState

	Position int64
	MftId    string
}

func (self *StateBase) GetPosition() int64     { return self.Position }
func (self *StateBase) GetMftId() string       { return self.MftId }
func (self *StateBase) SetMft(state *StateMft) { self.MftId = state.MftId }
func (self *StateBase) Print()                 { fmt.Println(self) }

func (self *StateBase) String() string {
	return fmt.Sprintf("{STATEBASE: MFTID=%s Position=%d}", self.MftId, self.Position)
}

type StateAttribute struct {
	BasePosition   int64
	RecordPosition int64
	Header         core.AttributeHeader
	RunList        core.RunList
}

type StateFileRecord struct {
	StateBase

	Header     core.FileRecord
	Reference  data.FileRef
	Parent     data.FileRef
	Name       string
	Names      []string
	Attributes []*StateAttribute
}

func (self *StateFileRecord) GetEncodingCode() string       { return "F" }
func (self *StateFileRecord) GetLabel() string              { return "MFT File Records" }
func (self *StateFileRecord) GetType() StateRecordType      { return STATE_RECORD_TYPE_FILE }
func (self *StateFileRecord) GetHeader() *core.RecordHeader { return &self.Header.RecordHeader }
func (self *StateFileRecord) HasName() bool                 { return true }
func (self *StateFileRecord) GetName() string               { return self.Name }
func (self *StateFileRecord) SetName(name string)           { self.Name = name }
func (self *StateFileRecord) IsDir() bool                   { return self.Header.IsDir() }
func (self *StateFileRecord) GetParent() data.FileRef       { return self.Parent }
func (self *StateFileRecord) Print()                        { fmt.Println("[FILE]"); core.PrintStruct(self) }

func (self *StateFileRecord) Init(disk *core.DiskIO) (bool, error) {
	disk.SetOffset(self.Position)
	err := disk.ReadStruct(0, &self.Header)
	if err != nil {
		return false, err
	}

	rec := &self.Header
	header := &rec.RecordHeader

	if (header.Type != core.RECTYP_FILE) || (header.UsaCount > 3) || ((header.UsaOffset + (header.UsaCount * 2)) >= 1024) {
		return false, nil
	}

	if (rec.BytesAllocated > 1024) || (rec.BytesInUse > 1024) || (rec.AttributesOffset >= 1024) {
		return false, nil
	}

	if ((rec.Flags & core.FFLAG_IN_USE) == core.FFLAG_NONE) || (int(rec.AttributesOffset) < self.Header.PrefixSize()) {
		return false, nil
	}

	min_sz := uint32(rec.PrefixSize())
	if (rec.BytesAllocated < min_sz) || (rec.BytesInUse < min_sz) {
		return false, nil
	}

	attributes, err := rec.GetAttributes(true)
	if err != nil {
		return false, err
	}

	sz := len(attributes)
	if sz == 0 {
		return false, nil
	}

	self.Attributes = make([]*StateAttribute, sz)

	indexes := make([]int, 0)
	for idx := range attributes {
		indexes = append(indexes, idx)
	}

	sort.Ints(indexes)
	for i, idx := range indexes {
		position := int64(idx)
		attr := attributes[idx]

		desc, err := rec.MakeAttributeFromHeader(attr)
		if err != nil {
			return false, err
		}

		self.Attributes[i] = &StateAttribute{
			BasePosition:   position + self.Position + int64(rec.PrefixSize()),
			RecordPosition: position,
			Header:         *attr,
			RunList:        desc.GetRunList(),
		}
	}

	return true, nil
}

func (self *StateFileRecord) GetAttributeDesc(attr *StateAttribute) (*core.AttributeDesc, error) {
	return self.Header.MakeAttributeFromHeader(&attr.Header)
}

func (self *StateFileRecord) GetAttribute(pos int64) *StateAttribute {
	for _, a := range self.Attributes {
		if a.RecordPosition == pos {
			return a
		}
	}

	return nil
}

func (self *StateFileRecord) GetAttributes(attr core.AttributeType, others ...core.AttributeType) []*StateAttribute {
	list := makeAttributeList(attr, others)

	for _, attr := range self.Attributes {
		list.add(attr)
	}

	return list.sort()
}

func (self *StateFileRecord) String() string {
	const msg = "{%s at %d [MFT:%s] [REF:%s] [Parent:%s]}"

	return fmt.Sprintf(msg, self.Name, self.Position, self.MftId, self.Reference, self.Parent)
}

func (self *StateFileRecord) MarshalBinary() ([]byte, error) {
	res, err := protobuf.Encode(new(tStateFileRecord).from(self))
	if err != nil {
		return nil, core.WrapError(err)
	}

	return res, nil
}

func (self *StateFileRecord) UnmarshalBinary(data []byte) error {
	var rec tStateFileRecord

	if err := protobuf.Decode(data, &rec); err != nil {
		return core.WrapError(err)
	}

	rec.to(self)

	return nil
}

type StateDirEntry struct {
	BasePosition   int64
	RecordPosition int64
	Parent         data.FileRef
	Header         core.DirectoryEntryExtendedHeader
	Name           string
}

type StateIndexRecord struct {
	StateBase

	RecordRef data.FileRef
	Header    core.IndexBlockHeader
	Entries   []*StateDirEntry
}

func (self *StateIndexRecord) GetEncodingCode() string       { return "I" }
func (self *StateIndexRecord) GetLabel() string              { return "Indexes" }
func (self *StateIndexRecord) GetType() StateRecordType      { return STATE_RECORD_TYPE_INDEX }
func (self *StateIndexRecord) GetHeader() *core.RecordHeader { return &self.Header.RecordHeader }
func (self *StateIndexRecord) Print()                        { fmt.Println("[INDEX]"); core.PrintStruct(self) }

func (self *StateIndexRecord) Init(disk *core.DiskIO) (bool, error) {
	buffer := make([]byte, 4096)

	disk.SetOffset(self.Position)
	if err := disk.ReadCluster(0, buffer); err != nil {
		return false, err
	}

	if err := core.Read(buffer, &self.Header); err != nil {
		return false, nil
	}

	record := &self.Header
	hdr := &record.RecordHeader

	if (hdr.Type != core.RECTYP_INDX) || (hdr.UsaCount > 9) || ((hdr.UsaOffset + (hdr.UsaCount * 2)) >= 4096) {
		return false, nil
	}

	if record.DirectoryIndex.EntriesOffset >= 4096 {
		return false, nil
	}

	entries, err := record.Entries(buffer)
	if err != nil {
		return false, err
	}

	sz := len(entries)
	if sz == 0 {
		return false, nil
	}

	self.Entries = make([]*StateDirEntry, sz)

	indexes := make([]int, 0)
	for idx := range entries {
		indexes = append(indexes, idx)
	}

	sort.Ints(indexes)
	for i, idx := range indexes {
		position := int64(idx)
		entry := entries[idx]

		self.Entries[i] = &StateDirEntry{
			BasePosition:   position + self.Position,
			RecordPosition: position,
			Header:         *entry,
			Parent:         entry.ParentFileRefNum,
			Name:           entry.DecodeFilename(buffer[idx:]),
		}
	}

	return true, nil
}

func (self *StateIndexRecord) String() string {
	const msg = "{<Index> at %d [MFT:%s] for file %s}"

	return fmt.Sprintf(msg, self.Position, self.MftId, self.RecordRef)
}

func (self *StateIndexRecord) MarshalBinary() ([]byte, error) {
	res, err := protobuf.Encode(new(tStateIndexRecord).from(self))
	if err != nil {
		return nil, core.WrapError(err)
	}

	return res, nil
}

func (self *StateIndexRecord) UnmarshalBinary(data []byte) error {
	var rec tStateIndexRecord

	if err := protobuf.Decode(data, &rec); err != nil {
		return core.WrapError(err)
	}

	rec.to(self)

	return nil
}

type StateMft struct {
	StateBase

	Header     core.FileRecord
	RunList    core.RunList
	PartOrigin int64
}

func (self *StateMft) GetEncodingCode() string       { return "M" }
func (self *StateMft) GetLabel() string              { return "MFT Descriptors" }
func (self *StateMft) GetType() StateRecordType      { return STATE_RECORD_TYPE_MFT }
func (self *StateMft) GetHeader() *core.RecordHeader { return &self.Header.RecordHeader }
func (self *StateMft) Print()                        { fmt.Println("[MFT]"); core.PrintStruct(self) }

func (self *StateMft) Init(disk *core.DiskIO) (bool, error) {
	if self.MftId == "" {
		self.MftId = core.NewFileId()
	}

	return true, nil
}

func (self *StateMft) GetReference(file *StateFileRecord) data.FileRef {
	fpos := file.Position - self.PartOrigin

	vidx := int64(0)
	for _, run := range self.RunList {
		start, end := int64(run.Start)*0x1000, int64(run.GetNext())*0x1000

		if (start <= fpos) && (fpos < end) {
			if uint32((vidx+(fpos-start))/1024) != file.Header.MftRecordNumber {
				return data.FileRef(0)
			}

			return file.Header.FileRef()
		}

		vidx += int64(run.Count) * 0x1000
	}

	return data.FileRef(0)
}

func (self *StateMft) IsMft() bool {
	return (self.PartOrigin + (int64(self.RunList[0].Start) * 4096)) == self.Position
}

func (self *StateMft) IsMirror(disk *NtfsDisk) (bool, error) {
	return (self.PartOrigin + (int64(self.RunList[0].Start) * 4096)) == self.Position, nil
}

func (self *StateMft) String() string {
	return fmt.Sprintf("{%s at %d}", self.MftId, self.Position)
}

func (self *StateMft) MarshalBinary() ([]byte, error) {
	res, err := protobuf.Encode(new(tStateMft).from(self))
	if err != nil {
		return nil, core.WrapError(err)
	}

	return res, nil
}

func (self *StateMft) UnmarshalBinary(data []byte) error {
	var rec tStateMft

	if err := protobuf.Decode(data, &rec); err != nil {
		return core.WrapError(err)
	}

	rec.to(self)

	return nil
}

func init() {
	datafile.RegisterFileFormat(
		STATE_FORMAT_NAME,
		"[NTFS - STATES]",
		new(StateMft),
		new(StateIndexRecord),
		new(StateFileRecord),
	)
}

type IStateStreamItem interface {
	Index() int
	Offset() int64
	Record() IStateRecord
}

type tStateStreamError struct {
	record IStateRecord
}

func (*tStateStreamError) Index() int              { return -1 }
func (*tStateStreamError) Offset() int64           { return -1 }
func (se *tStateStreamError) Record() IStateRecord { return se.record }

type tStateStreamRecord struct {
	tStateStreamError

	pos   int64
	index int
}

func (sr *tStateStreamRecord) Index() int    { return sr.index }
func (sr *tStateStreamRecord) Offset() int64 { return sr.pos }

type StateStream <-chan IStateStreamItem

func (self StateStream) Close() error {
	defer core.DiscardPanic()

	reflect.ValueOf(self).Close()

	return nil
}

type tStateStream struct {
	stream chan IStateStreamItem
}

func (self *tStateStream) Close() error {
	defer core.DiscardPanic()

	close(self.stream)

	return nil
}

func (self *tStateStream) SendRecord(i uint, pos int64, rec data.IDataRecord) {
	defer core.DiscardPanic()

	self.stream <- &tStateStreamRecord{
		tStateStreamError: tStateStreamError{rec.(IStateRecord)},
		pos:               pos,
		index:             int(i),
	}
}

func (self *tStateStream) SendError(err error) {
	defer core.DiscardPanic()

	self.stream <- &tStateStreamError{&tStateError{err: err}}
}

type StateReader struct {
	reader *datafile.DataReader
}

func (self *StateReader) Close() error {
	return self.reader.Close()
}

func (self *StateReader) GetCount() int {
	return self.reader.GetCount()
}

func (self *StateReader) GetIndexCount() int {
	return self.reader.GetIndexCount()
}

func (self *StateReader) ReadRecord(position int64) (IStateRecord, error) {
	rec, err := self.reader.ReadRecord(position)
	if err != nil {
		return nil, err
	}

	res, ok := rec.(IStateRecord)
	if !ok {
		return nil, core.WrapError(fmt.Errorf("Bad record type"))
	}

	return res, nil
}

func (self *StateReader) GetRecordAt(index int) (IStateRecord, error) {
	rec, err := self.reader.GetRecordAt(index)
	if err != nil {
		return nil, err
	}

	res, ok := rec.(IStateRecord)
	if !ok {
		return nil, core.WrapError(fmt.Errorf("Bad record type"))
	}

	return res, nil
}

func (self *StateReader) MakeStream() (StateStream, error) {
	res := make(chan IStateStreamItem)

	if err := self.reader.InitStream(&tStateStream{res}); err != nil {
		return nil, err
	}

	return StateStream(res), nil
}

func OpenStateReader(filename string) (*StateReader, error) {
	f, err := core.OpenFile(filename, core.OPEN_RDONLY)
	if err != nil {
		return nil, core.WrapError(err)
	}

	defer core.DeferedCall(f.Close)

	return MakeStateReader(f)
}

func MakeStateReader(file *os.File) (*StateReader, error) {
	reader, err := datafile.MakeDataReader(file, STATE_FORMAT_NAME)
	if err != nil {
		return nil, err
	}

	res := &StateReader{
		reader: reader,
	}

	return res, nil
}

type StateWriter struct {
	writer *datafile.DataWriter
}

func (self *StateWriter) Close() (err error) {
	return self.writer.Close()
}

func (self *StateWriter) Write(rec IStateRecord) error {
	return self.writer.Write(rec)
}

func OpenStateWriter(filename string) (*StateWriter, error) {
	f, err := core.OpenFile(filename, core.OPEN_WRONLY)
	if err != nil {
		return nil, core.WrapError(err)
	}

	return MakeStateWriter(f)
}

func MakeStateWriter(file *os.File) (*StateWriter, error) {
	writer, err := datafile.MakeDataWriter(file, STATE_FORMAT_NAME)
	if err != nil {
		return nil, err
	}

	res := &StateWriter{
		writer: writer,
	}

	return res, nil
}
