package inspect

import (
	"fmt"
	"os"
	"reflect"
	"sort"

	"essai/ntfstool/core"
	"essai/ntfstool/core/dataio"
	"essai/ntfstool/core/dataio/datafile"

	"github.com/DeDiS/protobuf"
	"github.com/pborman/uuid"
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
	dataio.IDataRecord

	GetType() StateRecordType
	GetHeader() *core.RecordHeader
	GetMftId() string
	SetMft(state *StateMft)
	Init(disk *core.DiskIO) (bool, error)
}

type tNoneState struct {
	datafile.BaseDataRecord
}

func (self *tNoneState) GetType() StateRecordType             { return STATE_RECORD_TYPE_NONE }
func (self *tNoneState) GetHeader() *core.RecordHeader        { return nil }
func (self *tNoneState) GetMftId() string                     { return "" }
func (self *tNoneState) SetMft(state *StateMft)               {}
func (self *tNoneState) Init(disk *core.DiskIO) (bool, error) { return true, nil }

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

type StateFile struct {
	StateBase

	Parent    core.FileReferenceNumber
	Reference core.FileReferenceNumber
}

func (self *StateFile) String() string {
	const msg = "{STATFILE: MFTID=%s Parent=%s Ref=%s Pos=%s}"

	return fmt.Sprintf(msg, self.MftId, self.Parent, self.Reference, self.Position)
}

func (self *StateFile) Print() {
	fmt.Println(self)
}

type StateAttribute struct {
	BasePosition   int64
	RecordPosition int64
	Header         core.AttributeHeader
	RunList        core.RunList
}

type StateFileRecord struct {
	StateFile

	Header     core.FileRecord
	Name       string
	Names      []string
	Attributes []*StateAttribute
}

func (self *StateFileRecord) GetEncodingCode() string       { return "F" }
func (self *StateFileRecord) GetType() StateRecordType      { return STATE_RECORD_TYPE_FILE }
func (self *StateFileRecord) GetHeader() *core.RecordHeader { return &self.Header.RecordHeader }
func (self *StateFileRecord) Print()                        { fmt.Println("[FILE]"); core.PrintStruct(self) }

func (self *StateFileRecord) Init(disk *core.DiskIO) (bool, error) {
	disk.SetOffset(self.Position)
	err := disk.ReadStruct(0, &self.Header)
	if err != nil {
		return false, err
	}

	record := &self.Header
	header := &record.RecordHeader

	if (header.Type != core.RECTYP_FILE) || (header.UsaCount > 3) || ((header.UsaOffset + (header.UsaCount * 2)) >= 1024) {
		return false, nil
	}

	if (record.BytesAllocated > 1024) || (record.BytesInUse > 1024) || (record.AttributesOffset >= 1024) {
		return false, nil
	}

	if (record.Flags & core.FFLAG_IN_USE) == core.FFLAG_NONE {
		return false, nil
	}

	min_sz := uint32(record.PrefixSize())
	if (record.BytesAllocated < min_sz) || (record.BytesInUse < min_sz) {
		return false, nil
	}

	attributes, err := record.GetAttributes(true)
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

		desc, err := record.MakeAttributeFromHeader(attr)
		if err != nil {
			return false, err
		}

		self.Attributes[i] = &StateAttribute{
			BasePosition:   position + self.Position + int64(record.PrefixSize()),
			RecordPosition: position,
			Header:         *attr,
			RunList:        desc.GetRunList(),
		}
	}

	return true, nil
}

func (self *StateFileRecord) GetAttributes(attr_type core.AttributeType, other_types ...core.AttributeType) []*StateAttribute {
	filter := core.MakeAttributeTypeFilter(attr_type, other_types)

	res := make([]*StateAttribute, 0)
	for _, attr := range self.Attributes {
		if filter[attr.Header.AttributeType] {
			res = append(res, attr)
		}
	}

	return res
}

func (self *StateFileRecord) String() string {
	msg := "{%s at %d [MFT:%s] [REF:%s] [Parent:%s]}"
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
	Parent         core.FileReferenceNumber
	Header         core.DirectoryEntryExtendedHeader
	Name           string
}

type StateIndexRecord struct {
	StateFile

	Header  core.IndexBlockHeader
	Entries []*StateDirEntry
}

func (self *StateIndexRecord) GetEncodingCode() string       { return "I" }
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
	header := &record.RecordHeader

	if (header.Type != core.RECTYP_FILE) || (header.UsaCount > 9) || ((header.UsaOffset + (header.UsaCount * 2)) >= 4096) {
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

	return false, nil
}

func (self *StateIndexRecord) String() string {
	msg := "{<Index> at %d [MFT:%s] [REF:%s] [Parent:%s]}"
	return fmt.Sprintf(msg, self.Position, self.MftId, self.Reference, self.Parent)
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
func (self *StateMft) GetType() StateRecordType      { return STATE_RECORD_TYPE_MFT }
func (self *StateMft) GetHeader() *core.RecordHeader { return &self.Header.RecordHeader }
func (self *StateMft) Print()                        { fmt.Println("[MFT]"); core.PrintStruct(self) }

func (self *StateMft) Init(disk *core.DiskIO) (bool, error) {
	if self.MftId == "" {
		self.MftId = uuid.New()
	}

	return true, nil
}

func (self *StateMft) GetReference(position int64) core.FileReferenceNumber {
	part_pos := position - self.PartOrigin
	if (part_pos % 1024) != 0 {
		return core.FileReferenceNumber(0)
	}

	lcn := core.ClusterNumber(part_pos / 4096)
	offset := (part_pos % 4096) / 1024

	vcn := int64(0)
	for _, run := range self.RunList {
		if !run.Zero {
			start := run.Start

			if (start <= lcn) && (lcn < run.GetLast()) {
				return core.FileReferenceNumber(((vcn + int64(lcn-start)) * 4) + offset)
			}
		}

		vcn += run.Count
	}

	return core.FileReferenceNumber(0)
}

func (self *StateMft) IsMft(disk *NtfsDisk) (bool, error) {
	if (self.PartOrigin + (int64(self.RunList[0].Start) * 4096)) != self.Position {
		return false, nil
	}

	return true, nil
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

type StateStream <-chan IStateRecord

func (self StateStream) Close() error {
	defer func() {
		recover()
	}()

	reflect.ValueOf(self).Close()

	return nil
}

type tStateStream struct {
	stream chan IStateRecord
}

func (self *tStateStream) Close() error {
	defer func() {
		recover()
	}()

	close(self.stream)

	return nil
}

func (self *tStateStream) SendRecord(_ uint, rec dataio.IDataRecord) {
	self.stream <- rec.(IStateRecord)
}

func (self *tStateStream) SendError(err error) {
	self.stream <- &tStateError{err: err}
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
	res := make(chan IStateRecord)

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
