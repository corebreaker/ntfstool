package extract

import (
    "essai/ntfstool/core"
    "strings"
)

func SaveFile(from_disk *core.DiskIO, file *File, to_path string) (int64, error) {
    if strings.HasSuffix(to_path, "/") {
        to_path += file.Name
    }

    dest, err := core.OpenFile(to_path, core.OPEN_WRONLY)
    if err != nil {
        return 0, nil
    }

    defer core.DeferedCall(dest.Close)

    var bytes [4096]byte

    buffer := bytes[:]
    size, file_size, buf_size := 0, file.Size, uint64(len(bytes))

    for _, run := range file.RunList {
        if run.Zero {
            core.ClearBuffer(buffer)
            for i := int64(0); i < run.Count; i++ {
                cnt, err := dest.Write(buffer)
                if err != nil {
                    return 0, core.WrapError(err)
                }

                size += cnt
            }
        } else {
            start, end := run.Start, run.GetNext()
            for pos := start; (pos < end) && (file_size > 0); pos++ {
                if err := from_disk.ReadCluster(int64(pos), buffer); err != nil {
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
}
