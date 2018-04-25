package extract

import (
	"fmt"

	"essai/ntfstool/core"
	"essai/ntfstool/core/data"
	datafile "essai/ntfstool/core/data/file"
)

type File struct {
	datafile.BaseDataRecord

	FileRef   data.FileRef
	ParentRef data.FileRef
	Id        string
	Mft       string
	Parent    string
	ParentIdx int64
	Index     int64
	Position  int64
	Origin    int64
	Size      uint64
	Name      string
	RunList   core.RunList
}

func (self *File) IsRoot() bool              { return (len(self.Parent) == 0) || (self.Parent == self.Id) }
func (self *File) IsFile() bool              { return len(self.RunList) > 0 }
func (self *File) IsDir() bool               { return len(self.RunList) == 0 }
func (self *File) HasName() bool             { return true }
func (self *File) GetEncodingCode() string   { return "N" }
func (self *File) GetFile() *File            { return self }
func (self *File) GetId() string             { return self.Id }
func (self *File) GetPosition() int64        { return self.Position }
func (self *File) GetName() string           { return self.Name }
func (self *File) GetLabel() string          { return "Files Nodes" }
func (self *File) GetParent() data.FileRef   { return self.ParentRef }
func (self *File) GetIndex() int             { return int(self.Index) }
func (self *File) SetIndex(idx int)          { self.Index = int64(idx) }
func (self *File) Print()                    { core.PrintStruct(self) }
func (self *File) setParentIndex(idx *Index) { self.ParentIdx = idx.IdMap[self.Parent] }

func (self *File) String() string {
	const msg = "{%s at %d [MFT:%s; REF:%s; Parent:%s]}"

	return fmt.Sprintf(msg, self.Name, self.Position, self.Mft, self.FileRef, self.ParentRef)
}
