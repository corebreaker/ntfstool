package core

import (
    "os"

    "github.com/siddontang/go/ioutil2"
)

type OpenMode byte

const (
    OPEN_RDONLY OpenMode = iota
    OPEN_WRONLY
    OPEN_RDWR
    OPEN_APPEND
)

func OpenFile(path string, mode OpenMode) (*os.File, error) {
    var open_mode int

    switch mode {
    case OPEN_RDONLY:
        f, err := os.Open(path)
        if err != nil {
            return nil, WrapError(err)
        }

        return f, nil

    case OPEN_WRONLY:
        open_mode = os.O_WRONLY
        if ioutil2.FileExists(path) {
            open_mode |= os.O_TRUNC
        } else {
            open_mode |= os.O_CREATE
        }

    case OPEN_RDWR:
        open_mode = os.O_RDWR
        if !ioutil2.FileExists(path) {
            open_mode |= os.O_CREATE
        }

    case OPEN_APPEND:
        open_mode = os.O_APPEND
    }

    f, err := os.OpenFile(path, open_mode, 0664)
    if err != nil {
        return nil, WrapError(err)
    }

    return f, nil
}
