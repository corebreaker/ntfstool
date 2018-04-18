package extract

import (
	"os"

	"essai/ntfstool/core"
	"essai/ntfstool/core/data"
	datafile "essai/ntfstool/core/data/file"
)

type FileModifier struct {
	writer *datafile.DataModifier
}

func (self *FileModifier) Close() (err error) {
	return self.writer.Close()
}

func (self *FileModifier) GetCount() int {
	return self.writer.GetCount()
}

func (self *FileModifier) Write(rec IFile) error {
	return self.writer.Write(rec)
}

func (self *FileModifier) SetRecordAt(index int, rec data.IDataRecord) error {
	return self.writer.SetRecordAt(index, rec)
}

func (self *FileModifier) ReadRecord(position int64) (data.IDataRecord, error) {
	return self.writer.ReadRecord(position)
}

func (self *FileModifier) GetRecordAt(index int) (data.IDataRecord, error) {
	return self.writer.GetRecordAt(index)
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

	res := &FileModifier{
		writer: writer,
	}

	return res, nil
}
