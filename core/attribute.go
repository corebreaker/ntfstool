package core

import (
    "bytes"
    "encoding/binary"
    "errors"
    "fmt"
    "reflect"
)

type tAttributeTypeInfo struct {
    name    string
    go_type reflect.Type
}

type AttributeType uint32

func (self AttributeType) String() string {
    res, ok := attr_types[self]
    if !ok {
        return fmt.Sprintf("UNKNOWN: %08X", uint32(self))
    }

    return res.name
}

func (self AttributeType) NewAttribute() interface{} {
    typ, ok := attr_types[self]
    if (!ok) || (typ == nil) || (typ.go_type == nil) {
        return nil
    }

    return reflect.New(typ.go_type).Interface()
}

func (self AttributeType) IsGood() bool {
    _, ok := attr_types[self]

    return ok
}

const (
    ATTR_NONE                  AttributeType = 0x00000000
    ATTR_STANDARD_INFORMATION  AttributeType = 0x00000010
    ATTR_ATTRIBUTE_LIST        AttributeType = 0x00000020
    ATTR_FILE_NAME             AttributeType = 0x00000030
    ATTR_OBJECT_ID             AttributeType = 0x00000040
    ATTR_SECURITY_DESCRIPTOR   AttributeType = 0x00000050
    ATTR_VOLUME_NAME           AttributeType = 0x00000060
    ATTR_VOLUME_INFORMATION    AttributeType = 0x00000070
    ATTR_DATA                  AttributeType = 0x00000080
    ATTR_INDEX_ROOT            AttributeType = 0x00000090
    ATTR_INDEX_ALLOCATION      AttributeType = 0x000000A0
    ATTR_BITMAP                AttributeType = 0x000000B0
    ATTR_REPARSE_POINT         AttributeType = 0x000000C0
    ATTR_EA_INFORMATION        AttributeType = 0x000000D0
    ATTR_EA                    AttributeType = 0x000000E0
    ATTR_PROPERTY_SET          AttributeType = 0x000000F0
    ATTR_LOGGED_UTILITY_STREAM AttributeType = 0x00000100
    ATTR_END_OF_ATTRIBUTES     AttributeType = 0xFFFFFFFF
)

type AttributeTypeFilter map[AttributeType]bool

func MakeAttributeTypeFilter(first AttributeType, others []AttributeType) AttributeTypeFilter {
    filter := map[AttributeType]bool{first: true}
    for _, t := range others {
        filter[t] = true
    }

    return filter
}

type AttrFlag uint16

const (
    AFLAG_NONE AttrFlag = iota
    AFLAG_COMPRESSED
)

type ResidentAttrFlag uint16

const (
    RAFLAG_NONE ResidentAttrFlag = iota
    RAFLAG_INDEXED
)

type FileAttrFlag uint32

func (self FileAttrFlag) String() string {
    if self == FAFLAG_NONE {
        return "NONE"
    }

    res := ""
    for flag, name := range fattr_flags {
        if (self & flag) != FAFLAG_NONE {
            res += " | " + name
        }
    }

    if res == "" {
        return fmt.Sprintf("UNKNOWN: %08X", uint32(self))
    }

    return res[3:]
}

const (
    FAFLAG_NONE                FileAttrFlag = 0x00000
    FAFLAG_READONLY            FileAttrFlag = 0x00001
    FAFLAG_HIDDEN              FileAttrFlag = 0x00002
    FAFLAG_SYSTEM              FileAttrFlag = 0x00004
    FAFLAG_DIRECTORY           FileAttrFlag = 0x00010
    FAFLAG_ARCHIVE             FileAttrFlag = 0x00020
    FAFLAG_DEVICE              FileAttrFlag = 0x00040
    FAFLAG_NORMAL              FileAttrFlag = 0x00080
    FAFLAG_TEMPORARY           FileAttrFlag = 0x00100
    FAFLAG_SPARSE_FILE         FileAttrFlag = 0x00200
    FAFLAG_REPARSE_POINT       FileAttrFlag = 0x00400
    FAFLAG_COMPRESSED          FileAttrFlag = 0x00800
    FAFLAG_OFFLINE             FileAttrFlag = 0x01000
    FAFLAG_NOT_CONTENT_INDEXED FileAttrFlag = 0x02000
    FAFLAG_ENCRYPTED           FileAttrFlag = 0x04000
    FAFLAG_INTEGRITY_STREAM    FileAttrFlag = 0x08000
    FAFLAG_VIRTUAL             FileAttrFlag = 0x10000
    FAFLAG_NO_SCRUB_DATA       FileAttrFlag = 0x20000
)

type NameType uint8

func (self NameType) String() string {
    res, ok := name_types[self]
    if !ok {
        return "NONE"
    }

    return res
}

const (
    NANE_TYPE_NONE NameType = iota
    NAME_TYPE_LONG
    NAME_TYPE_SHORT
    NAME_TYPE_SMALL
)

func typeof(v interface{}) reflect.Type {
    if v == nil {
        return nil
    }

    res := reflect.TypeOf(v)
    if res.Kind() == reflect.Ptr {
        res = res.Elem()
    }

    return res
}

var (
    attr_types = map[AttributeType]*tAttributeTypeInfo{
        ATTR_STANDARD_INFORMATION: &tAttributeTypeInfo{
            name:    "STANDARD_INFORMATION",
            go_type: typeof((*StandardInformationAttribute)(nil)),
        },

        ATTR_ATTRIBUTE_LIST: &tAttributeTypeInfo{
            name:    "ATTRIBUTE_LIST",
            go_type: typeof((*AttributeListAttribute)(nil)),
        },

        ATTR_FILE_NAME: &tAttributeTypeInfo{
            name:    "FILE_NAME",
            go_type: typeof((*FilenameAttribute)(nil)),
        },

        ATTR_OBJECT_ID: &tAttributeTypeInfo{
            name:    "OBJECT_ID",
            go_type: typeof(nil),
        },

        ATTR_SECURITY_DESCRIPTOR: &tAttributeTypeInfo{
            name:    "SECURITY_DESCRIPTOR",
            go_type: typeof(nil),
        },

        ATTR_VOLUME_NAME: &tAttributeTypeInfo{
            name:    "VOLUME_NAME",
            go_type: typeof(nil),
        },

        ATTR_VOLUME_INFORMATION: &tAttributeTypeInfo{
            name:    "VOLUME_INFORMATION",
            go_type: typeof(nil),
        },

        ATTR_DATA: &tAttributeTypeInfo{
            name:    "DATA",
            go_type: typeof(nil),
        },

        ATTR_INDEX_ROOT: &tAttributeTypeInfo{
            name:    "INDEX_ROOT",
            go_type: typeof((*IndexRootAttribute)(nil)),
        },

        ATTR_INDEX_ALLOCATION: &tAttributeTypeInfo{
            name:    "INDEX_ALLOCATION",
            go_type: typeof((*IndexBlockHeader)(nil)),
        },

        ATTR_BITMAP: &tAttributeTypeInfo{
            name:    "BITMAP",
            go_type: typeof(nil),
        },

        ATTR_REPARSE_POINT: &tAttributeTypeInfo{
            name:    "REPARSE_POINT",
            go_type: typeof((*ReparsePointAttribute)(nil)),
        },

        ATTR_EA_INFORMATION: &tAttributeTypeInfo{
            name:    "EA_INFORMATION",
            go_type: typeof((*EaInformationAttribute)(nil)),
        },

        ATTR_EA: &tAttributeTypeInfo{
            name:    "EA",
            go_type: typeof((*EaAttribute)(nil)),
        },

        ATTR_PROPERTY_SET: &tAttributeTypeInfo{
            name:    "PROPERTY_SET",
            go_type: typeof(nil),
        },

        ATTR_LOGGED_UTILITY_STREAM: &tAttributeTypeInfo{
            name:    "LOGGED_UTILITY_STREAM",
            go_type: typeof(nil),
        },

        ATTR_END_OF_ATTRIBUTES: &tAttributeTypeInfo{
            name:    "END_OF_ATTRIBUTES",
            go_type: typeof(nil),
        },
    }

    fattr_flags = map[FileAttrFlag]string{
        FAFLAG_READONLY:            "READONLY",
        FAFLAG_HIDDEN:              "HIDDEN",
        FAFLAG_SYSTEM:              "SYSTEM",
        FAFLAG_DIRECTORY:           "DIRECTORY",
        FAFLAG_ARCHIVE:             "ARCHIVE",
        FAFLAG_DEVICE:              "DEVICE",
        FAFLAG_NORMAL:              "NORMAL",
        FAFLAG_TEMPORARY:           "TEMPORARY",
        FAFLAG_SPARSE_FILE:         "SPARSE_FILE",
        FAFLAG_REPARSE_POINT:       "REPARSE_POINT",
        FAFLAG_COMPRESSED:          "COMPRESSED",
        FAFLAG_OFFLINE:             "OFFLINE",
        FAFLAG_NOT_CONTENT_INDEXED: "NOT_CONTENT_INDEXED",
        FAFLAG_ENCRYPTED:           "ENCRYPTED",
        FAFLAG_INTEGRITY_STREAM:    "INTEGRITY_STREAM",
        FAFLAG_VIRTUAL:             "VIRTUAL",
        FAFLAG_NO_SCRUB_DATA:       "NO_SCRUB_DATA",
    }

    name_types = map[NameType]string{
        NAME_TYPE_LONG:  "LONG",
        NAME_TYPE_SHORT: "SHORT",
        NAME_TYPE_SMALL: "SMALL",
    }
)

type AttributeHeader struct {
    AttributeType   AttributeType
    Length          uint32
    NonResident     Boolean
    NameLength      uint8
    NameOffset      uint16
    Flags           AttrFlag
    AttributeNumber uint16
}

type ResidentAttribute struct {
    AttributeHeader
    ValueLength uint32
    ValueOffset uint16
    Flags       ResidentAttrFlag
}

type NonResidentAttribute struct {
    AttributeHeader
    LowVcn              ClusterNumber
    HighVcn             ClusterNumber
    RunArrayOffset      uint16
    CompressionUnit     uint8
    AlignmentOrReserved [5]uint8
    AllocatedSize       uint64
    DataSize            uint64
    InitializedSize     uint64
    CompressedSize      uint64
}

type StandardInformationAttribute struct {
    CreationTime                 Timestamp
    ChangeTime                   Timestamp
    LastWriteTime                Timestamp
    LastAccessTime               Timestamp
    FileAttributes               uint32
    AlignmentOrReservedOrUnknown [3]uint32
    QuotaId                      uint32
    SecurityId                   uint32
    QuotaCharge                  uint64
    USN                          Usn
}

type AttributeListAttribute struct {
    AttributeType       AttributeType
    Length              uint16
    NameLength          uint8
    NameOffset          uint8
    LowVcn              ClusterNumber
    FileReferenceNumber uint64
    AttributeNumber     uint16
    AlignmentOrReserved [3]uint16
}

type IndexRootAttribute struct {
    Type                  AttributeType
    CollationRule         uint32
    BytesPerIndexBlock    uint32
    ClustersPerIndexBlock uint32
    DirectoryIndex        DirectoryIndex
}

type FilenameAttribute struct {
    DirectoryFileReferenceNumber FileReferenceNumber
    CreationTime                 Timestamp
    ChangeTime                   Timestamp
    LastWriteTime                Timestamp
    LastAccessTime               Timestamp
    AllocatedSize                uint64
    DataSize                     uint64
    FileAttributes               FileAttrFlag
    AlignmentOrReserved          uint32
    NameLength                   uint8
    NameType                     NameType
    // Name                      [1]Char
}

type ReparsePointAttribute struct {
    ReparseTag        uint32
    ReparseDataLength uint16
    Reserved          uint16
    ReparseData       [1]byte
}

type EaInformationAttribute struct {
    EaLength      uint32
    EaQueryLength uint32
}

type EaAttribute struct {
    NextEntryOffset uint32
    Flags           uint8
    EaNameLength    uint8
    EaValueLength   uint16
    // EaName       [1]Char
    // EaData       []byte
}

type AttributeDefinition struct {
    AttributeName   [64]Char
    AttributeNumber uint32
    Unknown         [2]uint32
    Flags           uint32
    MinimumSize     uint64
    MaximumSize     uint64
}

type RunEntry struct {
    Start ClusterNumber
    Count int64
    Zero  bool
}

func (self *RunEntry) GetNext() ClusterNumber {
    return self.Start + ClusterNumber(self.Count)
}

func (self *RunEntry) GetLast() ClusterNumber {
    return self.GetNext() - 1
}

func (self *RunEntry) String() string {
    zero_str := ""
    if self.Zero {
        zero_str = " (Zero)"
    }

    return fmt.Sprintf("%d - %d [ Count= %d ]%s", self.Start, self.GetLast(), self.Count, zero_str)
}

type RunList []*RunEntry

type AttributeDesc struct {
    Record  *FileRecord
    Header  *AttributeHeader
    Index   int
    Name    string
    Desc    interface{}
    RunList RunList
}

func (self *AttributeDesc) ResidentDesc() *ResidentAttribute {
    res, _ := self.Desc.(*ResidentAttribute)

    return res
}

func (self *AttributeDesc) NonResidentDesc() *NonResidentAttribute {
    res, _ := self.Desc.(*NonResidentAttribute)

    return res
}

func (self *AttributeDesc) GetSize() uint64 {
    if self.Header.NonResident != BOOL_FALSE {
        desc := self.NonResidentDesc()

        return desc.DataSize
    } else {
        desc := self.ResidentDesc()

        return uint64(desc.ValueLength)
    }
}

func (self *AttributeDesc) GetValue(io *DiskIO) (*AttributeValue, error) {
    var data []byte
    var first_lcn int64
    var size int

    switch {
    case self.Header.NonResident == BOOL_FALSE:
        attr := self.ResidentDesc()
        start := self.Index + int(attr.ValueOffset)
        end := start + int(attr.ValueLength)
        buffer := self.Record.Data[start:end]

        size = len(buffer)
        data = make([]byte, size)
        copy(data, buffer)

    case io != nil:
        attr := self.NonResidentDesc()

        size = int(attr.DataSize)
        data = make([]byte, size)

        run_datas := self.Record.Data[(self.Index + int(attr.RunArrayOffset)):]
        run_size := 1
        lcn, vcn := int64(0), int64(0)

        for i := 0; run_datas[i] != 0; i += run_size {
            first := lcn == 0
            infos := run_datas[i]
            offs_sz, cnt_sz := int(infos>>4), int(infos&0xF)
            run_size = offs_sz + cnt_sz + 1

            start := i + 1
            cnt := DecodeInt(run_datas[start:(start + cnt_sz)])
            start += cnt_sz
            if offs_sz > 0 {
                delta := DecodeInt(run_datas[start:(start + offs_sz)])
                if delta > 0 {
                    lcn += delta
                    if err := io.ReadClusters(lcn, cnt, data[vcn:]); err != nil {
                        return nil, err
                    }
                }
            }

            if first {
                first_lcn = lcn
            }

            vcn += cnt * int64(4096)
        }

    default:
        attr := self.NonResidentDesc()

        size = int(attr.DataSize)

        run_datas := self.Record.Data[(self.Index + int(attr.RunArrayOffset)):]
        infos := run_datas[0]

        offs_sz := int(infos >> 4)
        if offs_sz > 0 {
            start := 1 + int(infos&0xF)
            delta := DecodeInt(run_datas[start:(start + offs_sz)])
            if delta > 0 {
                first_lcn = delta
            }
        }
    }

    if data == nil {
        res := &AttributeValue{
            Desc:     self,
            Data:     nil,
            FirstLCN: ClusterNumber(first_lcn),
            Content:  nil,
            Size:     size,
            Value:    nil,
        }

        return res, nil
    }

    value := self.Header.AttributeType.NewAttribute()
    buffer := bytes.NewBuffer(data)

    if value != nil {
        if err := binary.Read(buffer, binary.LittleEndian, value); err != nil {
            return nil, WrapError(err)
        }
    }

    res := &AttributeValue{
        Desc:     self,
        Data:     buffer.Bytes(),
        FirstLCN: ClusterNumber(first_lcn),
        Content:  data,
        Size:     size,
        Value:    value,
    }

    return res, nil
}

func (self *AttributeDesc) GetRunList() RunList {
    if self.Header.NonResident == BOOL_FALSE {
        return nil
    }

    res := self.RunList
    if res == nil {
        res = make(RunList, 0)
        attr := self.NonResidentDesc()

        run_datas := self.Record.Data[(self.Index + int(attr.RunArrayOffset)):]
        run_size := 1
        lcn := ClusterNumber(0)
        last := (*RunEntry)(nil)

        for i := 0; run_datas[i] != 0; i += run_size {
            infos := run_datas[i]
            offs_sz, cnt_sz := int(infos>>4), int(infos&0xF)
            run_size = offs_sz + cnt_sz + 1

            start := i + 1
            cnt := DecodeInt(run_datas[start:(start + cnt_sz)])

            if offs_sz <= 0 {
                continue
            }

            start += cnt_sz
            delta := ClusterNumber(DecodeInt(run_datas[start:(start + offs_sz)]))
            if delta > 0 {
                lcn += delta
                last = &RunEntry{
                    Start: lcn,
                    Count: cnt,
                }

                res = append(res, last)
            } else if last != nil {
                res = append(res, &RunEntry{
                    Start: last.GetNext(),
                    Count: cnt,
                    Zero:  true,
                })
            }
        }

        self.RunList = res
    }

    return res
}

type AttributeValue struct {
    Desc     *AttributeDesc
    FirstLCN ClusterNumber
    Content  []byte
    Data     []byte
    Size     int
    Value    interface{}
}

func (self *AttributeValue) get_filename_attribute() *FilenameAttribute {
    if self.Desc.Header.AttributeType != ATTR_FILE_NAME {
        return nil
    }

    res, _ := self.Value.(*FilenameAttribute)

    return res
}

func (self *AttributeValue) GetFilename() string {
    value := self.get_filename_attribute()

    if (value == nil) || (value.NameLength == 0) {
        return ""
    }

    return DecodeString(self.Data, int(value.NameLength))
}

func (self *AttributeValue) IsLongName() bool {
    value := self.get_filename_attribute()
    if value == nil {
        return false
    }

    return (value.NameType & NAME_TYPE_LONG) != NANE_TYPE_NONE
}

func (self *AttributeValue) GetParent() FileReferenceNumber {
    value := self.get_filename_attribute()
    if value == nil {
        return FileReferenceNumber(0)
    }

    return value.DirectoryFileReferenceNumber
}

func (self *AttributeValue) GetFirstEntry() (*DirectoryEntry, error) {
    var index *DirectoryIndex

    ir_attr, ok := self.Value.(*IndexRootAttribute)
    if ok {
        index = &ir_attr.DirectoryIndex
    } else {
        ia_attr, ok := self.Value.(*IndexBlockHeader)
        if !ok {
            return nil, WrapError(errors.New("It's not an Index attribute"))
        }

        index = &ia_attr.DirectoryIndex
    }

    offset := uint(index.EntriesOffset) - uint(StructSize(index))
    res := &DirectoryEntry{EntryOffset: offset, Index: index}
    buffer := self.Data[offset:]

    if err := Read(buffer, &res.DirectoryEntryHeader); err != nil {
        return nil, WrapError(err)
    }

    if (res.Flags & DEFLAG_HAS_TRAILING) != DEFLAG_NONE {
        start := int(res.Length) - binary.Size(res.Vcn)
        if err := Read(buffer[start:], &res.Vcn); err != nil {
            return nil, WrapError(err)
        }
    }

    /*
       if ((res.Flags & DEFLAG_LAST_ENTRY) != DEFLAG_NONE) && (res.FileReferenceNumber == 0) {
           return self.GetNextEntry(res)
       }
    */

    res.Name = res.DecodeFilename(buffer)

    return res, nil
}

func (self *AttributeValue) GetNextEntry(entry *DirectoryEntry) (*DirectoryEntry, error) {
    block_offset, entry_offset, index := entry.BlockOffset, entry.EntryOffset, entry.Index

    _, ok := self.Value.(*IndexRootAttribute)
    if ok {
        if (entry.Flags & DEFLAG_LAST_ENTRY) != DEFLAG_NONE {
            return nil, nil
        }

        entry_offset += uint(entry.Length)
    } else {
        ia_attr, ok := self.Value.(*IndexBlockHeader)
        if !ok {
            return nil, WrapError(errors.New("It's not an Index attribute"))
        }

        if (entry.Flags & DEFLAG_LAST_ENTRY) != DEFLAG_NONE {
            bias := uint(StructSize(ia_attr))

            if block_offset == 0 {
                block_offset = uint(4096) - bias
            } else {
                block_offset += 4096
            }

            if uint64(block_offset+bias) >= self.Desc.GetSize() {
                return nil, nil
            }

            ia_attr = new(IndexBlockHeader)
            buffer := self.Data[block_offset:]
            if err := Read(buffer, ia_attr); err != nil {
                if IsEof(err) {
                    return nil, nil
                }

                return nil, WrapError(err)
            }

            struct_offset := bias - uint(StructSize(ia_attr.DirectoryIndex))
            index_offset := block_offset + struct_offset

            index = &ia_attr.DirectoryIndex
            entry_offset = index_offset + uint(index.EntriesOffset)
        } else {
            entry_offset += uint(entry.Length)
        }
    }

    res := &DirectoryEntry{BlockOffset: block_offset, EntryOffset: entry_offset, Index: index}
    buffer := self.Data[entry_offset:]

    if err := Read(buffer, &res.DirectoryEntryHeader); err != nil {
        return nil, WrapError(err)
    }

    if (res.Flags & DEFLAG_HAS_TRAILING) != DEFLAG_NONE {
        start := int(res.Length) - binary.Size(res.Vcn)
        if err := Read(buffer[start:], &res.Vcn); err != nil {
            return nil, WrapError(err)
        }
    }

    /*
       if ((res.Flags & DEFLAG_LAST_ENTRY) != DEFLAG_NONE) && (res.FileReferenceNumber == 0) {
           return self.GetNextEntry(res)
       }
    */

    res.Name = res.DecodeFilename(buffer)

    return res, nil
}

func (self *AttributeValue) GetIndexBlock(entry_position uint) (interface{}, error) {
    index_root, ok := self.Value.(*IndexRootAttribute)
    if ok {
        return index_root, nil
    }

    attr, ok := self.Value.(*IndexBlockHeader)
    if !ok {
        return nil, WrapError(errors.New("It's not an Index attribute"))
    }

    block := new(IndexBlockHeader)
    *block = *attr
    offset := 0

    for i := uint(0); i < entry_position; i++ {
        if offset == 0 {
            offset = block.PrefixSize()
        } else {
            offset += 4096
        }

        if err := Read(self.Data[offset:], block); err != nil {
            return nil, WrapError(err)
        }
    }

    return block, nil
}

func (self *AttributeValue) GetIndexBlockFromEntry(entry *DirectoryEntry) (interface{}, error) {
    index_root, ok := self.Value.(*IndexRootAttribute)
    if ok {
        return index_root, nil
    }

    attr, ok := self.Value.(*IndexBlockHeader)
    if !ok {
        return nil, WrapError(errors.New("It's not an Index attribute"))
    }

    block := new(IndexBlockHeader)
    *block = *attr

    if entry.BlockOffset != 0 {
        if err := Read(self.Data[entry.BlockOffset:], block); err != nil {
            return nil, err
        }
    }

    return block, nil
}
