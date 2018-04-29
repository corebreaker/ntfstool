package extract

import (
	"os"

	"github.com/corebreaker/ntfstool/core"
	datafile "github.com/corebreaker/ntfstool/core/data/file"
)

type FileWriter struct {
	writer *datafile.DataWriter
}

func (self *FileWriter) Close() (err error) {
	return self.writer.Close()
}

func (self *FileWriter) Write(rec IFile) error {
	return self.writer.Write(rec)
}

func (self *FileWriter) WriteTree(t *Tree, progress func(cur, tot int)) error {
	if progress == nil {
		progress = func(int, int) {}
	}

	index := &Index{
		IdMap: make(map[string]int64),
	}

	list := []IFile{index}

	type tListHelper struct {
		add func(node *Node)
	}

	var helper tListHelper

	helper.add = func(n *Node) {
		f := n.File

		idx := int64(len(list))
		list = append(list, f)
		f.Index = idx
		index.IdMap[f.Id] = idx

		for _, child := range n.Children {
			helper.add(child)
		}
	}

	for _, root := range t.Roots {
		helper.add(root)
	}

	cnt := len(list)
	for i, f := range list {
		progress(i, cnt)
		f.setParentIndex(index)

		if err := self.Write(f); err != nil {
			return err
		}
	}

	return nil
}

func OpenFileWriter(filename string) (*FileWriter, error) {
	f, err := core.OpenFile(filename, core.OPEN_WRONLY)
	if err != nil {
		return nil, core.WrapError(err)
	}

	return MakeFileWriter(f)
}

func MakeFileWriter(file *os.File) (*FileWriter, error) {
	writer, err := datafile.MakeDataWriter(file, FILENODES_FORMAT_NAME)
	if err != nil {
		return nil, err
	}

	res := &FileWriter{
		writer: writer,
	}

	return res, nil
}
