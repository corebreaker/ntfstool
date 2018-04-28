package extract

import (
	"fmt"
	"os"
	"reflect"

	"essai/ntfstool/core"
	datafile "essai/ntfstool/core/data/file"
)

type FileModifier struct {
	writer   *datafile.DataModifier
	index    *Index
	modified bool
}

func (self *FileModifier) getIndexWithId(id string) (int, error) {
	res, found := self.index.IdMap[id]
	if !found {
		return 0, core.WrapError(fmt.Errorf("ID `%s` don't exist", id))
	}

	return int(res), nil
}

func (self *FileModifier) Close() (err error) {
	defer func() {
		e := self.writer.Close()
		if err == nil {
			err = e
		}
	}()

	if !self.modified {
		return nil
	}

	return self.writer.SetRecordAt(0, self.index)
}

func (self *FileModifier) GetCount() int {
	return self.writer.GetCount()
}

func (self *FileModifier) Write(rec IFile) (err error) {
	defer func() {
		if err == nil {
			self.index.IdMap[rec.GetId()] = int64(rec.GetIndex())
			self.modified = true
		}
	}()

	return self.writer.Write(rec)
}

func (self *FileModifier) SetRecordAt(index int, rec IFile) error {
	old, err := self.GetRecordAt(index)
	if err != nil {
		return err
	}

	defer func() {
		if err == nil {
			idx := self.index.IdMap

			delete(idx, old.GetId())
			idx[rec.GetId()] = int64(rec.GetIndex())

			self.modified = true
		}
	}()

	return self.writer.SetRecordAt(index, rec)
}

func (self *FileModifier) SetRecordWithId(id string, rec IFile) error {
	idx, err := self.getIndexWithId(id)
	if err != nil {
		return err
	}

	return self.SetRecordAt(idx, rec)
}

func (self *FileModifier) ReadRecord(position int64) (IFile, error) {
	rec, err := self.writer.ReadRecord(position)
	if err != nil {
		return nil, err
	}

	res, ok := rec.(IFile)
	if !ok {
		v := reflect.ValueOf(rec)
		if v.Kind() == reflect.Interface {
			v = v.Elem()
		}

		return nil, core.WrapError(fmt.Errorf("Bad record type: %s", v.Type()))
	}

	return res, nil
}

func (self *FileModifier) GetRecordAt(index int) (IFile, error) {
	rec, err := self.writer.GetRecordAt(index)
	if err != nil {
		return nil, err
	}

	res, ok := rec.(IFile)
	if !ok {
		v := reflect.ValueOf(rec)
		if v.Kind() == reflect.Interface {
			v = v.Elem()
		}

		return nil, core.WrapError(fmt.Errorf("Bad record type: %s", v.Type()))
	}

	return res, nil
}

func (self *FileModifier) GetRecordWithId(id string) (IFile, error) {
	idx, err := self.getIndexWithId(id)
	if err != nil {
		return nil, err
	}

	return self.GetRecordAt(idx)
}

func (self *FileModifier) DelRecordAt(index int) error {
	fmt.Println("<<<<<<< I=", index, " // ", len(self.index.IdMap), self.writer.GetCount())
	old, err := self.GetRecordAt(index)
	if err != nil {
		return err
	}

	defer func() {
		if err == nil {
			delete(self.index.IdMap, old.GetId())

			idmap := self.index.IdMap
			ref_idx := int64(index)
			for id, rec_idx := range idmap {
				if rec_idx > ref_idx {
					idmap[id]--
				}
			}

			self.modified = true
		}
	}()

	return self.writer.DelRecordAt(index)
}

func (self *FileModifier) DelRecordWithId(id string) error {
	idx, err := self.getIndexWithId(id)
	if err != nil {
		return err
	}

	return self.DelRecordAt(idx)
}

func (self *FileModifier) MakeFileStream() (FileStream, error) {
	res := make(chan IFileStreamItem)

	if err := self.writer.InitStreamFrom(&tFileStream{res}, 0); err != nil {
		return nil, err
	}

	return FileStream(res), nil
}

func OpenFileModifier(filename string) (*FileModifier, error) {
	f, err := core.OpenFile(filename, core.OPEN_WRONLY)
	if err != nil {
		return nil, core.WrapError(err)
	}

	return MakeFileModifier(f)
}

func MakeFileModifier(file *os.File) (*FileModifier, error) {
	writer, err := datafile.MakeDataModifier(file, FILENODES_FORMAT_NAME)
	if err != nil {
		return nil, err
	}

	record, err := writer.GetRecordAt(0)
	if err != nil {
		return nil, err
	}

	index, ok := record.(*Index)
	if !ok {
		return nil, core.WrapError(fmt.Errorf("Bad file format; No index record found"))
	}

	res := &FileModifier{
		writer: writer,
		index:  index,
	}

	return res, nil
}
