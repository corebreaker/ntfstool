package extract

import (
	"fmt"
	"os"
	"reflect"

	"essai/ntfstool/core"
	"essai/ntfstool/core/dataio"
	"essai/ntfstool/core/dataio/datafile"
)

const FILENODES_FORMAT_NAME = "File nodes"

type IFile interface {
	dataio.IDataRecord

	GetFile() *File
	IsRoot() bool
	IsFile() bool
	IsDir() bool
}

type tNoneFile struct {
	datafile.BaseDataRecord

	zero bool
}

func (self *tNoneFile) IsRoot() bool   { return false }
func (self *tNoneFile) IsFile() bool   { return false }
func (self *tNoneFile) IsDir() bool    { return false }
func (self *tNoneFile) GetFile() *File { return nil }

type File struct {
	tNoneFile

	Ref      int64
	Id       string
	Mft      string
	Parent   string
	Position int64
	Size     uint64
	Name     string
	RunList  core.RunList
}

func (self *File) IsRoot() bool            { return len(self.Parent) == 0 }
func (self *File) IsFile() bool            { return len(self.RunList) > 0 }
func (self *File) IsDir() bool             { return len(self.RunList) == 0 }
func (self *File) GetEncodingCode() string { return "N" }
func (self *File) GetFile() *File          { return self }
func (self *File) Print()                  { core.PrintStruct(self) }

type tFileError struct {
	tNoneFile

	err error
}

func (self *tFileError) GetError() error { return self.err }
func (self *tFileError) GetFile() *File  { return nil }
func (self *tFileError) Print()          { fmt.Println("Error:", self.err) }

func init() {
	datafile.RegisterFileFormat(FILENODES_FORMAT_NAME, "[NTFS - FNODES]", new(File))
}

type FileStream <-chan IFile

func (self FileStream) Close() error {
	defer func() {
		recover()
	}()

	reflect.ValueOf(self).Close()

	return nil
}

type tFileStream struct {
	stream chan IFile
}

func (self *tFileStream) Close() error {
	close(self.stream)

	return nil
}

func (self *tFileStream) SendRecord(_ uint, rec dataio.IDataRecord) {
	self.stream <- rec.(*File)
}

func (self *tFileStream) SendError(err error) {
	self.stream <- &tFileError{err: err}
}

type FileReader struct {
	reader *datafile.DataReader
}

func (self *FileReader) Close() error {
	return self.reader.Close()
}

func (self *FileReader) GetCount() int {
	return self.reader.GetCount()
}

func (self *FileReader) ReadRecord(position int64) (IFile, error) {
	rec, err := self.reader.ReadRecord(position)
	if err != nil {
		return nil, err
	}

	res, ok := rec.(IFile)
	if !ok {
		return nil, core.WrapError(fmt.Errorf("Bad record type"))
	}

	return res, nil
}

func (self *FileReader) GetRecordAt(index int) (IFile, error) {
	rec, err := self.reader.GetRecordAt(index)
	if err != nil {
		return nil, err
	}

	res, ok := rec.(IFile)
	if !ok {
		return nil, core.WrapError(fmt.Errorf("Bad record type"))
	}

	return res, nil
}

func (self *FileReader) MakeStream() (FileStream, error) {
	res := make(chan IFile)

	if err := self.reader.InitStream(&tFileStream{res}); err != nil {
		return nil, err
	}

	return FileStream(res), nil
}

func OpenFileReader(filename string) (*FileReader, error) {
	f, err := core.OpenFile(filename, core.OPEN_RDONLY)
	if err != nil {
		return nil, core.WrapError(err)
	}

	defer core.DeferedCall(f.Close)

	return MakeFileReader(f)
}

func MakeFileReader(file *os.File) (*FileReader, error) {
	reader, err := datafile.MakeDataReader(file, FILENODES_FORMAT_NAME)
	if err != nil {
		return nil, err
	}

	res := &FileReader{
		reader: reader,
	}

	return res, nil
}

type FileWriter struct {
	writer *datafile.DataWriter
}

func (self *FileWriter) Close() (err error) {
	return self.writer.Close()
}

func (self *FileWriter) Write(rec IFile) error {
	return self.writer.Write(rec)
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
