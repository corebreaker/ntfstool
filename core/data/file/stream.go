package file

import (
	"reflect"

	"github.com/corebreaker/ntfstool/core"
	"github.com/corebreaker/ntfstool/core/data"
)

type IStreamItem interface {
	Index() int
	Offset() int64
	Record() data.IDataRecord
}

type tStreamError struct {
	record data.IDataRecord
}

func (*tStreamError) Index() int                  { return -1 }
func (*tStreamError) Offset() int64               { return -1 }
func (se *tStreamError) Record() data.IDataRecord { return se.record }

type tStreamRecord struct {
	tStreamError

	index int
	pos   int64
}

func (sr *tStreamRecord) Index() int    { return sr.index }
func (sr *tStreamRecord) Offset() int64 { return sr.pos }

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

func (self *tStream) SendRecord(i uint, pos int64, rec data.IDataRecord) {
	defer core.DiscardPanic()

	self.stream <- &tStreamRecord{
		tStreamError: tStreamError{rec},
		pos:          pos,
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

func (self *DataReader) MakeStreamFrom(from int64) (RecordStream, error) {
	res := make(chan IStreamItem)

	if err := self.InitStreamFrom(&tStream{res}, from); err != nil {
		return nil, err
	}

	return RecordStream(res), nil
}
