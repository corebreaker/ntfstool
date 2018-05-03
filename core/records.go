package core

import (
	"bytes"
	"fmt"
	"io"

	"github.com/corebreaker/ntfstool/core/data"
)

type RecordType uint32

const (
	RECTYP_NONE RecordType = 0x00000000
	RECTYP_FILE RecordType = 0x454C4946 // 'FILE'
	RECTYP_INDX RecordType = 0x58444E49 // 'INDX'
	RECTYP_BAAD RecordType = 0x44414142 // 'BAAD'
	RECTYP_HOLE RecordType = 0x454C4F48 // 'HOLE'
	RECTYP_CHKD RecordType = 0x444B4843 // 'CHKD'
)

func (self RecordType) IsGood() bool {
	return record_types[self]
}

func (self RecordType) String() string {
	if !self.IsGood() {
		return ""
	}

	var buf bytes.Buffer

	if err := Write(&buf, self); err != nil {
		return ""
	}

	return buf.String()
}

func (self RecordType) Bytes() ([]byte, error) {
	if !self.IsGood() {
		return nil, WrapError(fmt.Errorf("Bad record type: %08x", uint32(self)))
	}

	var buf bytes.Buffer

	if err := Write(&buf, self); err != nil {
		return nil, WrapError(err)
	}

	return buf.Bytes(), nil
}

func RecordTypeFromString(s string) RecordType {
	return RecordTypeFromBytes([]byte(s))
}

func RecordTypeFromBytes(s []byte) RecordType {
	if len(s) < 4 {
		return RECTYP_NONE
	}

	var res RecordType

	if err := Read(s[:4], &res); (err != nil) || (!res.IsGood()) {
		return RECTYP_NONE
	}

	return res
}

type FileFlag uint16

const (
	FFLAG_NONE FileFlag = iota
	FFLAG_IN_USE
	FFLAG_DIRECTORY
)

func (self FileFlag) String() string {
	if self == FFLAG_NONE {
		return "NONE"
	}

	res := ""
	if (self & FFLAG_IN_USE) != FFLAG_NONE {
		res += " | IN_USE"
	}

	if (self & FFLAG_DIRECTORY) != FFLAG_NONE {
		res += " | DIRECTORY"
	}

	if res == "" {
		return fmt.Sprintf("UNKNOWN: %08X", uint16(self))
	}

	return res[3:]
}

var (
	record_types = map[RecordType]bool{
		RECTYP_FILE: true,
		RECTYP_INDX: true,
		RECTYP_BAAD: true,
		RECTYP_HOLE: true,
		RECTYP_CHKD: true,
	}
)

type RecordHeader struct {
	Type      RecordType
	UsaOffset uint16
	UsaCount  uint16
	Usn       Usn
}

type FileRecord struct {
	RecordHeader

	SequenceNumber      uint16
	LinkCount           uint16
	AttributesOffset    uint16
	Flags               FileFlag
	BytesInUse          uint32
	BytesAllocated      uint32
	BaseFileRecord      uint64
	NextAttributeNumber uint16
	Reserved            uint16
	MftRecordNumber     uint32
	Data                FileRecordData
}

func (self *FileRecord) IsDir() bool {
	return (self.Flags & FFLAG_DIRECTORY) != FFLAG_NONE
}

func (self *FileRecord) PrefixSize() int {
	return StructSize(self) - len(self.Data)
}

func (self *FileRecord) FileRef() data.FileRef {
	return data.MakeFileRef(self.SequenceNumber, data.FileIndex(self.MftRecordNumber))
}

func (self *FileRecord) make_attribute(offset int, header *AttributeHeader) (*AttributeDesc, error) {
	name := ""
	if header.NameLength != 0 {
		name = DecodeString(self.Data[(offset+int(header.NameOffset)):], int(header.NameLength))
	}

	var desc interface{}

	if header.NonResident != BOOL_FALSE {
		attr := new(NonResidentAttribute)

		if err := Read(self.Data[offset:], attr); err != nil {
			return nil, err
		}

		header = &attr.AttributeHeader
		desc = attr
	} else {
		attr := new(ResidentAttribute)

		if err := Read(self.Data[offset:], attr); err != nil {
			return nil, err
		}

		header = &attr.AttributeHeader
		desc = attr
	}

	res := &AttributeDesc{
		Record: self,
		Header: header,
		Index:  offset,
		Name:   name,
		Desc:   desc,
	}

	return res, nil
}

func (self *FileRecord) MakeAttributeFromHeader(header *AttributeHeader) (*AttributeDesc, error) {
	offset := int(self.AttributesOffset) - self.PrefixSize()

	for {
		var attr AttributeHeader

		if err := Read(self.Data[offset:], &attr); err != nil {
			return nil, err
		}

		if attr.AttributeType == ATTR_END_OF_ATTRIBUTES {
			return nil, WrapError(fmt.Errorf("Attribute doesn't exists (type= %08x)", uint32(attr.AttributeType)))
		}

		if (attr.AttributeType == header.AttributeType) && (attr.AttributeNumber == header.AttributeNumber) {
			break
		}

		offset += int(attr.Length)
	}

	return self.make_attribute(offset, header)
}

func (self *FileRecord) MakeAttributeFromOffset(offset int) (*AttributeDesc, error) {
	attr := new(AttributeHeader)

	if err := Read(self.Data[offset:], attr); err != nil {
		return nil, err
	}

	if offset < 0 {
		offset += int(self.AttributesOffset)
	}

	return self.make_attribute(offset, attr)
}

func (self *FileRecord) GetAttributes(filter bool) (map[int]*AttributeHeader, error) {
	attributes := make(map[int]*AttributeHeader)

	idx := int(self.AttributesOffset) - self.PrefixSize()
	for {
		attr := new(AttributeHeader)

		if (0 > idx) || (idx >= 1024) {
			return nil, nil
		}

		err := Read(self.Data[idx:], attr)
		if err != nil {
			if GetSource(err) == io.ErrUnexpectedEOF {
				if err := Read(self.Data[idx:], &attr.AttributeType); err != nil {
					return nil, err
				}
			} else {
				return nil, err
			}
		}

		if attr.AttributeType == ATTR_END_OF_ATTRIBUTES {
			break
		}

		if err != nil {
			return nil, err
		}

		if filter && ((!attr.AttributeType.IsGood()) || (attr.NameOffset >= 1024)) {
			return nil, nil
		}

		attributes[idx] = attr

		idx += int(attr.Length)
	}

	return attributes, nil
}

func (self *FileRecord) GetAttributeFilteredList(attr_type AttributeType, other_types ...AttributeType) []int {
	filter := MakeAttributeTypeFilter(attr_type, other_types)

	var attr AttributeHeader
	var res []int

	idx := int(self.AttributesOffset) - self.PrefixSize()
	for {
		Read(self.Data[idx:], &attr)

		t := attr.AttributeType
		if t == ATTR_END_OF_ATTRIBUTES {
			break
		}

		if filter[t] {
			res = append(res, idx)
		}

		idx += int(attr.Length)
	}

	return res
}

func (self *FileRecord) GetFilename(disk *DiskIO) (string, error) {
	if self.Type != RECTYP_FILE {
		return "<No file>", nil
	}

	idx := int(self.AttributesOffset) - self.PrefixSize()

	last_val := (*AttributeValue)(nil)

attr_loop:
	for {
		var attr AttributeHeader

		if err := Read(self.Data[idx:], &attr); err != nil {
			return "", err
		}

		pos := idx
		idx += int(attr.Length)

		switch attr.AttributeType {
		case ATTR_END_OF_ATTRIBUTES:
			break attr_loop

		case ATTR_FILE_NAME:
			desc, err := self.make_attribute(pos, &attr)
			if err != nil {
				return "", err
			}

			val, err := desc.GetValue(disk)
			if err != nil {
				if IsEof(err) {
					return "<No file>", nil
				}

				return "", err
			}

			last_val = val

			if !val.IsLongName() {
				continue
			}

			return val.GetFilename(), nil
		}
	}

	if last_val != nil {
		return last_val.GetFilename(), nil
	}

	return "", nil
}
