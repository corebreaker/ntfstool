package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"

	ntfs "github.com/corebreaker/ntfstool/core"
	"github.com/corebreaker/ntfstool/core/data"
	"github.com/corebreaker/ntfstool/core/data/codec"
	"github.com/corebreaker/ntfstool/core/data/file"
	"github.com/corebreaker/ntfstool/extract"
	"github.com/corebreaker/ntfstool/inspect"
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
	file.BaseDataRecord

	I, J int
}

func (r *tRecord) String() string          { return fmt.Sprintf("<V:%d/%d>", r.I, r.J) }
func (r *tRecord) GetPosition() int64      { return int64(r.I) }
func (r *tRecord) GetEncodingCode() string { return "VAL" }
func (r *tRecord) Print()                  { fmt.Println(r) }

func (r *tRecord) MarshalBinary() ([]byte, error) {
	return []byte(fmt.Sprintf("%d|%d", r.I, r.J)), nil
}

func (r *tRecord) UnmarshalBinary(data []byte) error {
	parts := strings.Split(string(data), "|")

	i, err := strconv.Atoi(parts[0])
	if err != nil {
		return ntfs.WrapError(err)
	}

	j, err := strconv.Atoi(parts[1])
	if err != nil {
		return ntfs.WrapError(err)
	}

	r.I, r.J = i, j

	return nil
}

type tStream struct {
	cancel context.CancelFunc
}

type IDR = data.IDataRecord

func (s *tStream) Close() error                    { fmt.Println("End"); s.cancel(); return nil }
func (*tStream) SendError(err error)               { panic(err) }
func (*tStream) SendRecord(i uint, p int64, r IDR) { fmt.Printf("  - %d (%d): ", i+1, p); r.Print() }

func work() error {
	file.RegisterFileFormat(
		"Example",
		"[-- EXAMPLE --]",
		new(tRecord),
		new(extract.File),
		new(inspect.StateMft),
		new(inspect.StateIndexRecord),
		new(inspect.StateFileRecord),
	)

	filename := flag.String("file", "", "File path")
	is_record := flag.Bool("record", false, "Read record")
	is_record2 := flag.Bool("record2", false, "Read record")
	is_read := flag.Bool("read", false, "Read file")
	is_count := flag.Bool("count", false, "Count records")
	index := flag.Int("show", -1, "Show at index")
	flag.Parse()

	if *filename == "" {
		return ntfs.WrapError(fmt.Errorf("No file specified"))
	}

	if *is_record {
		b := MakeBuffer(102400, file.GetRegistry("Example"))
		rd := func(name string) error {
			f1, err := ntfs.OpenFile(*filename+name, ntfs.OPEN_RDONLY)
			if err != nil {
				return ntfs.WrapError(err)
			}

			defer f1.Close()

			b.Reset()
			_, err = f1.Read(b.Get(102400))
			if err != nil {
				return ntfs.WrapError(err)
			}

			return nil
		}

		if err := rd("1.dat"); err != nil {
			return err
		}

		rec, err := ntfs.ReadRecord(b)
		if err != nil {
			return err
		}

		fmt.Println("Result=", rec, b.b.i)

		if err := rd("2.dat"); err != nil {
			return err
		}

		rec, err = ntfs.ReadRecord(b)
		if err != nil {
			return err
		}

		fmt.Println("Result=", rec, b.b.i)

		return nil
	}

	if *is_record2 {
		f, err := ntfs.OpenFile(*filename, ntfs.OPEN_RDONLY)
		if err != nil {
			return err
		}

		defer f.Close()

		r := filter(f)

		d := codec.MakeDecoder(r, file.GetRegistry("Example"))
		dec := d.ToCoreDecoder()

		rec, err := ntfs.ReadRecord(dec)
		if err != nil {
			return err
		}

		fmt.Println("Result=", rec, r.i)
		rec, err = ntfs.ReadRecord(dec)
		if err != nil {
			return err
		}

		fmt.Println("Result=", rec, r.i)

		return nil
	}

	if *is_count {
		f, err := file.OpenDataReader(*filename, "Example")
		if err != nil {
			return err
		}

		defer f.Close()

		fmt.Println("Count=", f.GetCount())

		return nil
	}

	if *is_read {
		f, err := file.OpenDataReader(*filename, "Example")
		if err != nil {
			return err
		}

		defer f.Close()

		fmt.Println("Count=", f.GetCount(), "- At:", *index)
		if *index <= 0 {
			ctx, cancel := context.WithCancel(context.Background())

			fmt.Println("List:")
			if err := f.InitStream(&tStream{cancel}); err != nil {
				return err
			}

			<-ctx.Done()

			return nil
		}

		rec, err := f.GetRecordAt(*index - 1)
		if err != nil {
			return err
		}

		fmt.Println("Result=", rec)

		return nil
	}

	f, err := file.OpenDataWriter(*filename, "Example")
	if err != nil {
		return err
	}

	defer ntfs.DeferedCall(f.Close)

	for i, v := range flag.Args() {
		parts := strings.Split(v, ":")

		if len(parts) < 2 {
			parts = []string{fmt.Sprint(i + 1), parts[0]}
		}

		valI, err := strconv.Atoi(parts[0])
		if err != nil {
			return err
		}

		valJ, err := strconv.Atoi(parts[1])
		if err != nil {
			return err
		}

		if err := f.Write(&tRecord{I: valI, J: valJ}); err != nil {
			return err
		}
	}

	pos := int64(len(flag.Args())) + 1
	setPos := func(val interface{}) data.IDataRecord {
		reflect.ValueOf(val).Elem().FieldByName("Position").Set(reflect.ValueOf(pos))
		pos++

		return val.(data.IDataRecord)
	}

	if err := f.Write(setPos(new(inspect.StateMft))); err != nil {
		return err
	}

	if err := f.Write(setPos(new(inspect.StateIndexRecord))); err != nil {
		return err
	}

	if err := f.Write(setPos(new(inspect.StateFileRecord))); err != nil {
		return err
	}

	return nil
}

func main() {
	ntfs.CheckedMain(work)
}
