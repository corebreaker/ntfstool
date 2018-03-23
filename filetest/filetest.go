package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strconv"

	"essai/ntfstool/core"
	"essai/ntfstool/core/dataio"
	"essai/ntfstool/core/dataio/codec"
	"essai/ntfstool/core/dataio/datafile"
)

type tReader struct {
	src io.Reader
	i   int
}

func (self *tReader) Read(p []byte) (int, error) {
	x, err := self.src.Read(p)
	if err != nil {
		return 0, err
	}

	self.i += x

	return x, nil
}

func filter(r io.Reader) *tReader {
	return &tReader{src: r}
}

type tBufData struct {
	buffer []byte
	offset int
}

func (self *tBufData) Read(p []byte) (n int, err error) {
	if self.offset >= len(self.buffer) {
		if len(p) == 0 {
			return
		}

		return 0, io.EOF
	}

	n = copy(p, self.buffer[self.offset:])
	self.offset += n

	return
}

type Buffer struct {
	data   *tBufData
	reader *codec.Decoder
	b      *tReader
}

func (self *Buffer) Get(size int) []byte {
	data := self.data
	if (0 >= size) || (size > len(data.buffer)) {
		return data.buffer
	}

	return data.buffer[:size]
}

func (self *Buffer) Reset() {
	self.data.offset = 0
}

func (self *Buffer) Decode() (interface{}, error) {
	var res interface{}

	_, err := self.reader.Decode(&res)

	return res, err
}

func MakeBuffer(size int, registry *codec.Registry) *Buffer {
	data := &tBufData{
		buffer: make([]byte, size),
	}

	b := filter(data)

	return &Buffer{
		b:      b,
		data:   data,
		reader: codec.MakeDecoder(b, registry),
	}
}

type tRecord struct {
	datafile.BaseDataRecord

	I int
}

func (self *tRecord) String() string          { return fmt.Sprintf("<V:%d>", self.I) }
func (self *tRecord) GetPosition() int64      { return int64(self.I) }
func (self *tRecord) GetEncodingCode() string { return "VAL" }
func (self *tRecord) Print()                  { fmt.Println(self) }

type tStream struct {
	cancel context.CancelFunc
}

func (s *tStream) Close() error                    { fmt.Println("End"); s.cancel(); return nil }
func (*tStream) SendRecord(rec dataio.IDataRecord) { rec.Print() }
func (*tStream) SendError(err error)               { panic(err) }

func work() error {
	datafile.RegisterFileFormat("Example", "[-- EXAMPLE --]", new(tRecord))

	filename := flag.String("file", "", "File path")
	is_record := flag.Bool("record", false, "Read record")
	is_record2 := flag.Bool("record2", false, "Read record")
	is_read := flag.Bool("read", false, "Read file")
	is_count := flag.Bool("count", false, "Count records")
	index := flag.Int("show", -1, "Show at index")
	flag.Parse()

	if *filename == "" {
		return core.WrapError(fmt.Errorf("No file specified"))
	}

	if *is_record {
		b := MakeBuffer(102400, datafile.GetRegistry("Example"))
		rd := func(name string) error {
			f1, err := core.OpenFile(*filename+name, core.OPEN_RDONLY)
			if err != nil {
				return core.WrapError(err)
			}

			defer f1.Close()

			b.Reset()
			_, err = f1.Read(b.Get(102400))
			if err != nil {
				return core.WrapError(err)
			}

			return nil
		}

		if err := rd("1.dat"); err != nil {
			return err
		}

		rec, err := core.ReadRecord(b)
		if err != nil {
			return err
		}

		fmt.Println("Result=", rec, b.b.i)

		if err := rd("2.dat"); err != nil {
			return err
		}

		rec, err = core.ReadRecord(b)
		if err != nil {
			return err
		}

		fmt.Println("Result=", rec, b.b.i)

		return nil
	}

	if *is_record2 {
		f, err := core.OpenFile(*filename, core.OPEN_RDONLY)
		if err != nil {
			return err
		}

		defer f.Close()

		r := filter(f)

		d := codec.MakeDecoder(r, datafile.GetRegistry("Example"))
		dec := d.ToCoreDecoder()

		rec, err := core.ReadRecord(dec)
		if err != nil {
			return err
		}

		fmt.Println("Result=", rec, r.i)
		rec, err = core.ReadRecord(dec)
		if err != nil {
			return err
		}

		fmt.Println("Result=", rec, r.i)

		return nil
	}

	if *is_count {
		f, err := datafile.OpenDataReader(*filename, "Example")
		if err != nil {
			return err
		}

		defer f.Close()

		fmt.Println("Count=", f.GetCount())

		return nil
	}

	if *is_read {
		f, err := datafile.OpenDataReader(*filename, "Example")
		if err != nil {
			return err
		}

		defer f.Close()

		fmt.Println("Count=", f.GetCount(), "- At:", *index)
		if *index < 0 {
			ctx, cancel := context.WithCancel(context.Background())

			fmt.Println("List:")
			if err := f.InitStream(&tStream{cancel}); err != nil {
				return err
			}

			<-ctx.Done()

			return nil
		}

		rec, err := f.GetRecordAt(*index)
		if err != nil {
			return err
		}

		fmt.Println("Result=", rec)

		return nil
	}

	f, err := datafile.OpenDataWriter(*filename, "Example")
	if err != nil {
		return err
	}

	defer core.DeferedCall(f.Close)

	for _, v := range flag.Args() {
		val, err := strconv.Atoi(v)
		if err != nil {
			return err
		}

		if err := f.Write(&tRecord{I: val}); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	core.CheckedMain(work)
}
