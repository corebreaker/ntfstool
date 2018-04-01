package datafile

import (
	"reflect"

	"essai/ntfstool/core"
	"essai/ntfstool/core/dataio"
)

type IStreamItem interface {
	Index() int
	Record() dataio.IDataRecord
}

type tStreamError struct {
	record dataio.IDataRecord
}

func (*tStreamError) Index() int                    { return -1 }
func (se *tStreamError) Record() dataio.IDataRecord { return se.record }

type tStreamRecord struct {
	tStreamError

	index int
}

func (sr *tStreamRecord) Index() int { return sr.index }

type RecordStream <-chan IStreamItem

func (self RecordStream) Close() error {
	defer core.DiscardPanic()

	reflect.ValueOf(self).Close()

	return nil
}

type tStream struct {
	stream chan IStreamItem
}

func (self *tStream) Close() error {
	defer core.DiscardPanic()

	close(self.stream)

	return nil
}

func (self *tStream) SendRecord(i uint, rec dataio.IDataRecord) {
	defer core.DiscardPanic()

	self.stream <- &tStreamRecord{
		tStreamError: tStreamError{rec},
		index:        int(i),
	}
}

func (self *tStream) SendError(err error) {
	defer core.DiscardPanic()

	self.stream <- &tStreamError{&tErrorRecord{err: err}}
}

func (self *DataReader) MakeStream() (RecordStream, error) {
	res := make(chan IStreamItem)

	if err := self.InitStream(&tStream{res}); err != nil {
		return nil, err
	}

	return RecordStream(res), nil
}
