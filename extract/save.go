package extract

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/corebreaker/ntfstool/core"
)

func SaveNode(from_disk *core.DiskIO, node *Node, to_path string, noempty, nometa bool) (int64, error) {
	file := node.File
	if (nometa && IsMetaFile(file)) || (noempty && node.IsEmpty(nometa)) {
		return 0, nil
	}

	if node.IsFile() {
		destname := filepath.Join(to_path, file.Name)
		fmt.Println("  -", destname)

		dest, err := core.OpenFile(destname, core.OPEN_WRONLY)
		if err != nil {
			return 0, nil
		}

		defer core.DeferedCall(dest.Close)

		var bytes [4096]byte

		buffer := bytes[:]
		size, file_size, buf_size := 0, file.Size, uint64(len(bytes))

		from_disk.SetOffset(file.Origin)
		for _, run := range file.RunList {
			if file_size == 0 {
				break
			}

			if run.Zero {
				core.ClearBuffer(buffer)
				for i := int64(0); (i < run.Count) && (file_size > 0); i++ {
					cur_size := file_size
					if cur_size > buf_size {
						cur_size = buf_size
					}

					file_size -= cur_size

					cnt, err := dest.Write(buffer[:cur_size])
					if err != nil {
						return 0, core.WrapError(err)
					}

					size += cnt
				}
			} else {
				start, end := run.Start, run.GetNext()
				for pos := start; (pos < end) && (file_size > 0); pos++ {
					if err := from_disk.ReadCluster(int64(pos), buffer); (err != nil) && (core.GetSource(err) != io.EOF) {
						return 0, err
					}

					cur_size := file_size
					if cur_size > buf_size {
						cur_size = buf_size
					}

					file_size -= cur_size

					cnt, err := dest.Write(buffer[:cur_size])
					if err != nil {
						return 0, core.WrapError(err)
					}

					size += cnt
				}
			}
		}

		return int64(size), nil
	} else {
		if !file.IsDir() {
			return 0, core.WrapError(fmt.Errorf("Node %s do not represent a directory.", file))
		}

		dirname := filepath.Join(to_path, file.Name)
		size := int64(0)

		if err := os.MkdirAll(dirname, 0770); err != nil {
			return 0, core.WrapError(err)
		}

		for _, child := range node.Children {
			sz, err := SaveNode(from_disk, child, dirname, noempty, nometa)
			if err != nil {
				return 0, err
			}

			size += sz
		}

		return int64(size), nil
	}
}
