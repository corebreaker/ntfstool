package extract

import (
	"fmt"

	datafile "github.com/corebreaker/ntfstool/core/data/file"
)

type Index struct {
	datafile.BaseDataRecord

	IdMap map[string]int64
}

func (*Index) IsRoot() bool            { return false }
func (*Index) IsFile() bool            { return false }
func (*Index) IsDir() bool             { return false }
func (*Index) GetFile() *File          { return nil }
func (*Index) GetId() string           { return "" }
func (*Index) GetIndex() int           { return 0 }
func (*Index) GetEncodingCode() string { return "I" }
func (*Index) setParentIndex(*Index)   {}
func (*Index) SetName(string)          {}
func (idx *Index) Print()              { fmt.Println(idx) }
func (idx *Index) String() string      { return fmt.Sprintf("{File indexes:%d}", len(idx.IdMap)) }
func (idx *Index) addFile(f *File)     { idx.IdMap[f.Id] = f.Index }
func (idx *Index) addNode(n *Node)     { idx.addFile(n.File) }
