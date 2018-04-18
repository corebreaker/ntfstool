package extract

import (
	"fmt"
	"os"

	"essai/ntfstool/core"
)

type IFileReader interface {
	MakeFileStream() (FileStream, error)
}

type Node struct {
	File     *File
	Children []*Node
}

func (self *Node) IsEmpty(nometa bool) bool {
	if self.IsFile() {
		return self.File.Size == 0
	}

	if self.ChildCount(nometa) == 0 {
		return true
	}

	for _, subnode := range self.Children {
		if nometa && IsMetaFile(subnode.File) {
			continue
		}

		if !subnode.IsEmpty(nometa) {
			return false
		}
	}

	return true
}

func (self *Node) ChildCount(nometa bool) int {
	if !nometa {
		return len(self.Children)
	}

	res := 0
	for _, n := range self.Children {
		if !IsMetaFile(n.File) {
			res++
		}
	}

	return res
}

func (self *Node) IsRoot() bool {
	return self.File.IsRoot()
}

func (self *Node) IsFile() bool {
	return (len(self.Children) == 0) && self.File.IsFile()
}

func (self *Node) IsDir() bool {
	return self.File.IsDir()
}

func (self *Node) walk(stream *tFileStream) {
	file := self.File
	stream.SendRecord(uint(file.Index), file.Position, file)

	for _, n := range self.Children {
		n.walk(stream)
	}
}

type Tree struct {
	Index Index
	Roots []*Node
	IdMap map[string]*Node
}

func (self *Tree) get_node_path(node *Node, suffix string) string {
	if node == nil {
		return suffix
	}

	file := node.File
	parent := file.Parent

	switch parent {
	case "", file.Id:
		return "/" + suffix
	}

	res := file.Name
	if suffix != "" {
		res = fmt.Sprintf("%s/%s", res, suffix)
	}

	return self.get_node_path(self.IdMap[parent], res)
}

func (self *Tree) GetNodePath(node *Node) string {
	return self.get_node_path(node, "")
}

func (self *Tree) GetFilePath(file *File) string {
	return self.get_node_path(self.IdMap[file.Id], "")
}

func (self *Tree) GetFilePathFromFile(file IFile) string {
	return self.get_node_path(self.IdMap[file.GetId()], "")
}

func (self *Tree) MakeStream() FileStream {
	c := make(chan IFileStreamItem)
	res := FileStream(c)
	stream := &tFileStream{
		stream: c,
	}

	go func() {
		defer core.DeferedCall(res.Close)

		for _, root := range self.Roots {
			root.walk(stream)
		}
	}()

	return res
}

func (self *Tree) MakeStreamFrom(id string) FileStream {
	start, ok := self.IdMap[id]
	if !ok {
		return nil
	}

	c := make(chan IFileStreamItem)
	res := FileStream(c)
	stream := &tFileStream{
		stream: c,
	}

	go func() {
		defer core.DeferedCall(res.Close)

		start.walk(stream)
	}()

	return res
}

func MakeTree(reader IFileReader) (*Tree, error) {
	stream, err := reader.MakeFileStream()
	if err != nil {
		return nil, err
	}

	defer core.DeferedCall(stream.Close)

	var roots []*Node
	var nodes []*Node

	id_map := make(map[string]*Node)

	for item := range stream {
		rec := item.Record()
		if err := rec.GetError(); err != nil {
			return nil, err
		}

		if rec.IsNull() {
			continue
		}

		file := rec.GetFile()

		t := &Node{File: file}

		id_map[file.Id] = t
		nodes = append(nodes, t)

		if file.IsRoot() && file.IsDir() {
			roots = append(roots, t)
		}
	}

	for _, n := range nodes {
		parent, found := id_map[n.File.Parent]
		if !found {
			continue
		}

		parent.Children = append(parent.Children, n)
	}

	res := &Tree{
		IdMap: id_map,
		Roots: roots,
	}

	return res, nil
}

func ReadTree(filename string) (*Tree, error) {
	files, err := OpenFileReader(filename)
	if err != nil {
		return nil, err
	}

	return MakeTree(files)
}

func ReadTreeFromFile(file *os.File) (*Tree, error) {
	files, err := MakeFileReader(file)
	if err != nil {
		return nil, err
	}

	defer core.DeferedCall(files.Close)

	return MakeTree(files)
}
