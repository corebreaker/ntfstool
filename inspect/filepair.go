package inspect

import "essai/ntfstool/core"

type FilePair struct {
    Parent core.FileReferenceNumber
    Name   string
}

type FileFrequencies map[FilePair]uint

func (self FileFrequencies) Add(parent core.FileReferenceNumber, name string) bool {
    key := FilePair{
        Parent: parent,
        Name:   name,
    }

    self[key]++

    return self[key] == 2
}
