package core

import "fmt"

type DirFlag uint32

const (
	DFLAG_SMALL_DIR DirFlag = iota
	DFLAG_LARGE_DIR
)

type DirEntryFlag uint32

func (self DirEntryFlag) String() string {
	if self == DEFLAG_NONE {
		return "NONE"
	}

	res := ""
	if (self & DEFLAG_HAS_TRAILING) != DEFLAG_NONE {
		res += " | HAS_TRAILING"
	}

	if (self & DEFLAG_LAST_ENTRY) != DEFLAG_NONE {
		res += " | LAST_ENTRY"
	}

	if res == "" {
		return fmt.Sprintf("UNKNOWN: %08X", uint32(self))
	}

	return res[3:]
}

const (
	DEFLAG_NONE DirEntryFlag = iota
	DEFLAG_HAS_TRAILING
	DEFLAG_LAST_ENTRY
)

type DirectoryIndex struct {
	EntriesOffset    uint32
	IndexBlockLength uint32
	AllocatedSize    uint32
	Flags            DirFlag
}

type IndexBlockHeader struct {
	RecordHeader
	IndexBlockVcn  ClusterNumber
	DirectoryIndex DirectoryIndex
}

func (self *IndexBlockHeader) PrefixSize() int {
	return 4096 - StructSize(self)
}

func (self *IndexBlockHeader) Entries(data []byte) (map[int]*DirectoryEntryExtendedHeader, error) {
	pos := StructSize(self) - StructSize(self.DirectoryIndex) + int(self.DirectoryIndex.EntriesOffset)
	sz := len(data)
	if sz > 4096 {
		sz = 4096
	}

	res := make(map[int]*DirectoryEntryExtendedHeader)

	var entry DirectoryEntryHeader

	for ; (entry.Flags & DEFLAG_LAST_ENTRY) == DEFLAG_NONE; pos += int(entry.Length) {
		if pos > sz {
			break
		}

		buffer := data[pos:]
		if err := Read(buffer, &entry); err != nil {
			return nil, err
		}

		if (entry.Length == 0) || (entry.FilenameLength > 255) {
			break
		}

		item, err := entry.ExtendsHeaderFromData(buffer)
		if err != nil {
			return nil, err
		}

		res[pos] = item
	}

	return res, nil
}

type DirectoryEntryHeader struct {
	FileReferenceNumber FileReferenceNumber
	Length              uint16
	AttributeLength     uint16
	Flags               DirEntryFlag
	ParentFileRefNum    FileReferenceNumber
	CreationTime        Timestamp
	LastModifiedTime    Timestamp
	MFTRecordChangeTime Timestamp
	LastAccessTime      Timestamp
	PhysicalSize        uint64
	LogicalSize         uint64
	FileFlags           FileAttrFlag
	ExtendedAttributes  uint32
	FilenameLength      uint8
	FilenameType        uint8
}

func (self *DirectoryEntryHeader) ExtendsHeader() *DirectoryEntryExtendedHeader {
	return &DirectoryEntryExtendedHeader{
		DirectoryEntryHeader: *self,
	}
}

func (self *DirectoryEntryHeader) ExtendsHeaderWithVcn(vcn ClusterNumber) *DirectoryEntryExtendedHeader {
	return &DirectoryEntryExtendedHeader{
		DirectoryEntryHeader: *self,
		Vcn:                  vcn,
	}
}

func (self *DirectoryEntryHeader) ExtendsHeaderFromData(data []byte) (*DirectoryEntryExtendedHeader, error) {
	res := self.ExtendsHeader()
	if (self.Flags & DEFLAG_HAS_TRAILING) != DEFLAG_NONE {
		if err := Read(data, res); err != nil {
			return nil, err
		}
	}

	return res, nil
}

func (self *DirectoryEntryHeader) MakeEntry(block_offset, entry_offset uint) *DirectoryEntry {
	return &DirectoryEntry{
		DirectoryEntryExtendedHeader: DirectoryEntryExtendedHeader{
			DirectoryEntryHeader: *self,
		},
		BlockOffset: block_offset,
		EntryOffset: entry_offset,
	}
}

func (self *DirectoryEntryHeader) DecodeFilename(data []byte) string {
	if (self.FilenameLength == 0) || (self.FileReferenceNumber == 0) {
		return ""
	}

	return DecodeString(data[StructSize(self):], int(self.FilenameLength))
}

type DirectoryEntryExtendedHeader struct {
	DirectoryEntryHeader
	Vcn ClusterNumber // VCN in IndexAllocation of earlier entries
}

func (self *DirectoryEntryExtendedHeader) MakeEntry(block_offset, entry_offset uint) *DirectoryEntry {
	return &DirectoryEntry{
		DirectoryEntryExtendedHeader: *self,
		BlockOffset:                  block_offset,
		EntryOffset:                  entry_offset,
	}
}

type DirectoryEntry struct {
	DirectoryEntryExtendedHeader
	Index       *DirectoryIndex
	Name        string
	BlockOffset uint
	EntryOffset uint
}
