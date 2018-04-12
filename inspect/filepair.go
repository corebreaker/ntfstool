package inspect

import (
	"essai/ntfstool/core/data"
)

type FilePair struct {
	Parent data.FileRef
	Name   string
}

type FileFrequencies map[FilePair]uint

func (self FileFrequencies) Add(parent data.FileRef, name string) bool {
	key := FilePair{
		Parent: parent,
		Name:   name,
	}

	self[key]++

	return self[key] == 2
}
