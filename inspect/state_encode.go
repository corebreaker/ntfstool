package inspect

import (
	"essai/ntfstool/core"
)

type tRecordHeader struct {
	Type      core.RecordType
	UsaOffset uint32
	UsaCount  uint32
	Usn       core.Usn
}

func (self *tRecordHeader) from(src *core.RecordHeader) *tRecordHeader {
	*self = tRecordHeader{
		Type:      src.Type,
		UsaOffset: uint32(src.UsaOffset),
		UsaCount:  uint32(src.UsaCount),
		Usn:       src.Usn,
	}

	return self
}

func (self *tRecordHeader) to(dest *core.RecordHeader) *core.RecordHeader {
	*dest = core.RecordHeader{
		Type:      self.Type,
		UsaOffset: uint16(self.UsaOffset),
		UsaCount:  uint16(self.UsaCount),
		Usn:       self.Usn,
	}

	return dest
}

type tFileRecord struct {
	tRecordHeader

	SequenceNumber      uint32
	LinkCount           uint32
	AttributesOffset    uint32
	Flags               uint32
	BytesInUse          uint32
	BytesAllocated      uint32
	BaseFileRecord      uint64
	NextAttributeNumber uint32
	Data                core.FileRecordData
}

func (self *tFileRecord) from(src *core.FileRecord) *tFileRecord {
	*self = tFileRecord{
		SequenceNumber:      uint32(src.SequenceNumber),
		LinkCount:           uint32(src.LinkCount),
		AttributesOffset:    uint32(src.AttributesOffset),
		Flags:               uint32(src.Flags),
		BytesInUse:          src.BytesInUse,
		BytesAllocated:      src.BytesAllocated,
		BaseFileRecord:      src.BaseFileRecord,
		NextAttributeNumber: uint32(src.NextAttributeNumber),
		Data:                src.Data,
	}

	self.tRecordHeader.from(&src.RecordHeader)

	return self
}

func (self *tFileRecord) to(dest *core.FileRecord) *core.FileRecord {
	*dest = core.FileRecord{
		SequenceNumber:      uint16(self.SequenceNumber),
		LinkCount:           uint16(self.LinkCount),
		AttributesOffset:    uint16(self.AttributesOffset),
		Flags:               core.FileFlag(self.Flags),
		BytesInUse:          self.BytesInUse,
		BytesAllocated:      self.BytesAllocated,
		BaseFileRecord:      self.BaseFileRecord,
		NextAttributeNumber: uint16(self.NextAttributeNumber),
		Data:                self.Data,
	}

	self.tRecordHeader.to(&dest.RecordHeader)

	return dest
}

type tStateBase struct {
	Position int64
	MftId    string
}

func (self *tStateBase) from(src *StateBase) *tStateBase {
	*self = tStateBase{
		Position: src.Position,
		MftId:    src.MftId,
	}

	return self
}

func (self *tStateBase) to(dest *StateBase) *StateBase {
	*dest = StateBase{
		Position: self.Position,
		MftId:    self.MftId,
	}

	return dest
}

type tIndexBlockHeader struct {
	tRecordHeader

	IndexBlockVcn  core.ClusterNumber
	DirectoryIndex core.DirectoryIndex
}

func (self *tIndexBlockHeader) from(src *core.IndexBlockHeader) *tIndexBlockHeader {
	*self = tIndexBlockHeader{
		IndexBlockVcn:  src.IndexBlockVcn,
		DirectoryIndex: src.DirectoryIndex,
	}

	self.tRecordHeader.from(&src.RecordHeader)

	return self
}

func (self *tIndexBlockHeader) to(dest *core.IndexBlockHeader) *core.IndexBlockHeader {
	*dest = core.IndexBlockHeader{
		IndexBlockVcn:  self.IndexBlockVcn,
		DirectoryIndex: self.DirectoryIndex,
	}

	self.tRecordHeader.to(&dest.RecordHeader)

	return dest
}

type tStateFile struct {
	tStateBase

	Parent    core.FileReferenceNumber
	Reference core.FileReferenceNumber
}

func (self *tStateFile) from(src *StateFile) *tStateFile {
	*self = tStateFile{
		Parent:    src.Parent,
		Reference: src.Reference,
	}

	self.tStateBase.from(&src.StateBase)

	return self
}

func (self *tStateFile) to(dest *StateFile) *StateFile {
	*dest = StateFile{
		Parent:    self.Parent,
		Reference: self.Reference,
	}

	self.tStateBase.to(&dest.StateBase)

	return dest
}

type tDirectoryEntryHeader struct {
	FileReferenceNumber core.FileReferenceNumber
	Length              uint32
	AttributeLength     uint32
	Flags               core.DirEntryFlag
	ParentFileRefNum    core.FileReferenceNumber
	CreationTime        core.Timestamp
	LastModifiedTime    core.Timestamp
	MFTRecordChangeTime core.Timestamp
	LastAccessTime      core.Timestamp
	PhysicalSize        uint64
	LogicalSize         uint64
	FileFlags           core.FileAttrFlag
	ExtendedAttributes  uint32
	FilenameLength      uint32
	FilenameType        uint32
}

func (self *tDirectoryEntryHeader) from(src *core.DirectoryEntryHeader) *tDirectoryEntryHeader {
	*self = tDirectoryEntryHeader{
		FileReferenceNumber: src.FileReferenceNumber,
		Length:              uint32(src.Length),
		AttributeLength:     uint32(src.AttributeLength),
		Flags:               src.Flags,
		ParentFileRefNum:    src.ParentFileRefNum,
		CreationTime:        src.CreationTime,
		LastModifiedTime:    src.LastModifiedTime,
		MFTRecordChangeTime: src.MFTRecordChangeTime,
		LastAccessTime:      src.LastAccessTime,
		PhysicalSize:        src.PhysicalSize,
		LogicalSize:         src.LogicalSize,
		FileFlags:           src.FileFlags,
		ExtendedAttributes:  src.ExtendedAttributes,
		FilenameLength:      uint32(src.FilenameLength),
		FilenameType:        uint32(src.FilenameType),
	}

	return self
}

func (self *tDirectoryEntryHeader) to(dest *core.DirectoryEntryHeader) *core.DirectoryEntryHeader {
	*dest = core.DirectoryEntryHeader{
		FileReferenceNumber: self.FileReferenceNumber,
		Length:              uint16(self.Length),
		AttributeLength:     uint16(self.AttributeLength),
		Flags:               self.Flags,
		ParentFileRefNum:    self.ParentFileRefNum,
		CreationTime:        self.CreationTime,
		LastModifiedTime:    self.LastModifiedTime,
		MFTRecordChangeTime: self.MFTRecordChangeTime,
		LastAccessTime:      self.LastAccessTime,
		PhysicalSize:        self.PhysicalSize,
		LogicalSize:         self.LogicalSize,
		FileFlags:           self.FileFlags,
		ExtendedAttributes:  self.ExtendedAttributes,
		FilenameLength:      uint8(self.FilenameLength),
		FilenameType:        uint8(self.FilenameType),
	}

	return dest
}

type tDirectoryEntryExtendedHeader struct {
	tDirectoryEntryHeader

	Vcn core.ClusterNumber
}

func (self *tDirectoryEntryExtendedHeader) from(src *core.DirectoryEntryExtendedHeader) *tDirectoryEntryExtendedHeader {
	*self = tDirectoryEntryExtendedHeader{
		Vcn: src.Vcn,
	}

	self.tDirectoryEntryHeader.from(&src.DirectoryEntryHeader)

	return self
}

func (h *tDirectoryEntryExtendedHeader) to(r *core.DirectoryEntryExtendedHeader) *core.DirectoryEntryExtendedHeader {
	*r = core.DirectoryEntryExtendedHeader{
		Vcn: h.Vcn,
	}

	h.tDirectoryEntryHeader.to(&r.DirectoryEntryHeader)

	return r
}

type tAttributeHeader struct {
	AttributeType   core.AttributeType
	Length          uint32
	NonResident     bool
	NameLength      uint32
	NameOffset      uint32
	Flags           uint32
	AttributeNumber uint32
}

func (self *tAttributeHeader) from(src *core.AttributeHeader) *tAttributeHeader {
	*self = tAttributeHeader{
		AttributeType:   src.AttributeType,
		Length:          src.Length,
		NonResident:     src.NonResident != core.BOOL_FALSE,
		NameLength:      uint32(src.NameLength),
		NameOffset:      uint32(src.NameOffset),
		Flags:           uint32(src.Flags),
		AttributeNumber: uint32(src.AttributeNumber),
	}

	return self
}

func (self *tAttributeHeader) to(dest *core.AttributeHeader) *core.AttributeHeader {
	var non_resident core.Boolean

	if self.NonResident {
		non_resident = core.BOOL_FALSE
	} else {
		non_resident = core.BOOL_TRUE
	}

	*dest = core.AttributeHeader{
		AttributeType:   self.AttributeType,
		Length:          self.Length,
		NonResident:     non_resident,
		NameLength:      uint8(self.NameLength),
		NameOffset:      uint16(self.NameOffset),
		Flags:           core.AttrFlag(self.Flags),
		AttributeNumber: uint16(self.AttributeNumber),
	}

	return dest
}

type tStateAttribute struct {
	BasePosition   int64
	RecordPosition int64
	Header         tAttributeHeader
	RunList        core.RunList
}

func (self *tStateAttribute) from(src *StateAttribute) *tStateAttribute {
	*self = tStateAttribute{
		BasePosition:   src.BasePosition,
		RecordPosition: src.RecordPosition,
		RunList:        src.RunList,
	}

	self.Header.from(&src.Header)

	return self
}

func (self *tStateAttribute) to(dest *StateAttribute) *StateAttribute {
	*dest = StateAttribute{
		BasePosition:   self.BasePosition,
		RecordPosition: self.RecordPosition,
		RunList:        self.RunList,
	}

	self.Header.to(&dest.Header)

	return dest
}

type tStateDirEntry struct {
	BasePosition   int64
	RecordPosition int64
	Parent         core.FileReferenceNumber
	Header         tDirectoryEntryExtendedHeader
	Name           string
}

func (self *tStateDirEntry) from(src *StateDirEntry) *tStateDirEntry {
	*self = tStateDirEntry{
		BasePosition:   src.BasePosition,
		RecordPosition: src.RecordPosition,
		Parent:         src.Parent,
		Name:           src.Name,
	}

	self.Header.from(&src.Header)

	return self
}

func (self *tStateDirEntry) to(dest *StateDirEntry) *StateDirEntry {
	*dest = StateDirEntry{
		BasePosition:   self.BasePosition,
		RecordPosition: self.RecordPosition,
		Parent:         self.Parent,
		Name:           self.Name,
	}

	self.Header.to(&dest.Header)

	return dest
}

type tStateIndexRecord struct {
	tStateFile

	Header  tIndexBlockHeader
	Entries []*tStateDirEntry
}

func (self *tStateIndexRecord) from(src *StateIndexRecord) *tStateIndexRecord {
	entries := make([]*tStateDirEntry, len(src.Entries))
	for i, entry := range src.Entries {
		entries[i] = new(tStateDirEntry).from(entry)
	}

	self.Entries = entries
	self.tStateFile.from(&src.StateFile)
	self.Header.from(&src.Header)

	return self
}

func (self *tStateIndexRecord) to(dest *StateIndexRecord) *StateIndexRecord {
	entries := make([]*StateDirEntry, len(self.Entries))
	for i, entry := range self.Entries {
		entries[i] = entry.to(new(StateDirEntry))
	}

	dest.Entries = entries
	self.tStateFile.to(&dest.StateFile)
	self.Header.to(&dest.Header)

	return dest
}

type tStateFileRecord struct {
	tStateFile

	Header     tFileRecord
	Name       string
	Names      []string
	Attributes []*tStateAttribute
}

func (self *tStateFileRecord) from(src *StateFileRecord) *tStateFileRecord {
	attributes := make([]*tStateAttribute, len(src.Attributes))
	for i, attr := range src.Attributes {
		attributes[i] = new(tStateAttribute).from(attr)
	}

	*self = tStateFileRecord{
		Name:       src.Name,
		Names:      src.Names,
		Attributes: attributes,
	}

	self.Header.from(&src.Header)
	self.tStateFile.from(&src.StateFile)

	return self
}

func (self *tStateFileRecord) to(dest *StateFileRecord) *StateFileRecord {
	attributes := make([]*StateAttribute, len(self.Attributes))
	for i, attr := range self.Attributes {
		attributes[i] = attr.to(new(StateAttribute))
	}

	*dest = StateFileRecord{
		Name:       self.Name,
		Names:      self.Names,
		Attributes: attributes,
	}

	self.Header.to(&dest.Header)
	self.tStateFile.to(&dest.StateFile)

	return dest
}

type tStateMft struct {
	tStateBase

	Header     tFileRecord
	RunList    core.RunList
	PartOrigin int64
}

func (self *tStateMft) from(src *StateMft) *tStateMft {
	*self = tStateMft{
		RunList:    src.RunList,
		PartOrigin: src.PartOrigin,
	}

	self.tStateBase.from(&src.StateBase)
	self.Header.from(&src.Header)

	return self
}

func (self *tStateMft) to(dest *StateMft) *StateMft {
	*dest = StateMft{
		RunList:    self.RunList,
		PartOrigin: self.PartOrigin,
	}

	self.tStateBase.to(&dest.StateBase)
	self.Header.to(&dest.Header)

	return dest
}
