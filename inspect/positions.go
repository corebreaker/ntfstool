package inspect

import (
	"bytes"
	"fmt"
	"os"

	"github.com/corebreaker/ntfstool/core"
)

const BUFFER_SIZE int64 = 100 * 1024 * 1024

func has_done(indexes []int64) bool {
	for _, i := range indexes {
		if (0 <= i) && (i < BUFFER_SIZE) {
			return false
		}
	}

	return true
}

func FindPositionsWithType(name string, rectype core.RecordType, rectypes ...core.RecordType) ([][]int64, error) {
	patterns := make([][]byte, len(rectypes))
	for i, t := range rectypes {
		p, err := t.Bytes()
		if err != nil {
			return nil, err
		}

		patterns[i] = p
	}

	pattern, err := rectype.Bytes()
	if err != nil {
		return nil, err
	}

	return FindPositionsWithPattern(name, pattern, patterns...)
}

func FindPositionsWithPattern(name string, pattern []byte, patterns ...[]byte) ([][]int64, error) {
	fmt.Println("Preparation")

	file, err := os.Open(name)
	if err != nil {
		return nil, core.WrapError(err)
	}

	defer core.DeferedCall(file.Close)

	info, err := file.Stat()
	if err != nil {
		return nil, core.WrapError(err)
	}

	if info.IsDir() {
		return nil, core.WrapError(fmt.Errorf("Path '%s' specifies a directory"))
	}

	size, err := file.Seek(0, os.SEEK_END)
	if err != nil {
		return nil, core.WrapError(err)
	}

	p_count := len(patterns) + 1
	pattern_list := make([][]byte, p_count)
	copy(pattern_list[1:], patterns)
	pattern_list[0] = pattern

	res := make([][]int64, p_count)
	starts := make([]int64, p_count)
	indexes := make([]int64, p_count)
	counts := make([]int, p_count)
	fpos := int64(0)

	fmt.Println("Buffer allocation")
	buffer := make([]byte, BUFFER_SIZE+int64(len(pattern))-1)

	buffer_size, err := file.ReadAt(buffer, 0)
	if (err != nil) && (!core.IsEof(err)) {
		return nil, err
	}

	fmt.Println("Start of scanning")
	for err == nil {
		for i, p := range pattern_list {
			idx := indexes[i]
			if (0 > idx) || (idx >= BUFFER_SIZE) {
				continue
			}

			offset := starts[i]

			idx = int64(bytes.Index(buffer[offset:buffer_size], p))
			if (0 <= idx) && (idx < BUFFER_SIZE) {
				offset += idx
				starts[i] = offset + 1

				res[i] = append(res[i], fpos+offset)
				counts[i]++
			}

			indexes[i] = idx
		}

		if has_done(indexes) {
			core.ClearBuffer(buffer)

			fpos += BUFFER_SIZE
			for i := range starts {
				starts[i] = 0
				indexes[i] = 0
			}

			pos := fpos * 10000 / size
			fmt.Printf("\r%d.%02d%% (found: %d)", pos/100, pos%100, counts)

			buffer_size, err = file.ReadAt(buffer, fpos)
		}
	}

	fmt.Println(fmt.Sprintf("\r100.00%% (found: %d)", counts))
	fmt.Println("End of scanning")

	if !core.IsEof(err) {
		return nil, err
	}

	return res, nil
}
