package extract

import (
	"fmt"
	"os"
	"reflect"

	"github.com/corebreaker/ntfstool/core"
	"github.com/corebreaker/ntfstool/core/data"
	datafile "github.com/corebreaker/ntfstool/core/data/file"
)

const FILENODES_FORMAT_NAME = "File nodes"

type IFile interface {
	data.IDataRecord

	GetFile() *File
	GetId() string
	IsRoot() bool
	IsFile() bool
	IsDir() bool
	SetName(string)

	setParentIndex(*Index)
}

type tNoneFile struct {
	datafile.BaseDataRecord

	Zero bool
}

func (*tNoneFile) IsRoot() bool          { return false }
func (*tNoneFile) IsFile() bool          { return false }
func (*tNoneFile) IsDir() bool           { return false }
func (*tNoneFile) GetFile() *File        { return nil }
func (*tNoneFile) GetId() string         { return "" }
func (*tNoneFile) GetIndex() int         { return 0 }
func (*tNoneFile) SetName(string)        {}
func (*tNoneFile) setParentIndex(*Index) {}

type tFileError struct {
	tNoneFile

	err error
}

func (self *tFileError) GetError() error { return self.err }
func (self *tFileError) GetFile() *File  { return nil }
func (self *tFileError) Print()          { fmt.Println("Error:", self.err) }

func init() {
	datafile.RegisterFileFormat(FILENODES_FORMAT_NAME, "[NTFS - FNODES]", new(File), new(Index))
}

type IFileStreamItem interface {
	Index() int
	Offset() int64
	Record() IFile
}

type tFileStreamError struct {
	record IFile
}

func (*tFileStreamError) Index() int       { return -1 }
func (*tFileStreamError) Offset() int64    { return -1 }
func (se *tFileStreamError) Record() IFile { return se.record }

type tFileStreamRecord struct {
	tFileStreamError

	index int
	pos   int64
}

func (sr *tFileStreamRecord) Index() int    { return sr.index }
func (sr *tFileStreamRecord) Offset() int64 { return sr.pos }

type FileStream <-chan IFileStreamItem

func (self FileStream) Close() error {
	defer core.DiscardPanic()

	reflect.ValueOf(self).Close()

	return nil
}

type tFileStream struct {
	stream chan IFileStreamItem
}

func (self *tFileStream) Close() error {
	defer core.DiscardPanic()

	close(self.stream)

	return nil
}

func (self *tFileStream) SendRecord(i uint, pos int64, rec data.IDataRecord) {
	defer core.DiscardPanic()

	self.stream <- &tFileStreamRecord{
		tFileStreamError: tFileStreamError{rec.(*File)},
		index:            int(i),
		pos:              pos,
	}
}

func (self *tFileStream) SendError(err error) {
	defer core.DiscardPanic()

	self.stream <- &tFileStreamError{&tFileError{err: err}}
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

func (self *FileReader) MakeFileStream() (FileStream, error) {
	return self.MakeStreamFrom(1)
}

func (self *FileReader) MakeStreamFrom(index int64) (FileStream, error) {
	res := make(chan IFileStreamItem)

	if err := self.reader.InitStreamFrom(&tFileStream{res}, index); err != nil {
		return nil, err
	}

	return FileStream(res), nil
}

func (self *FileReader) MakeStream() (FileStream, error) {
	res := make(chan IFileStreamItem)

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
