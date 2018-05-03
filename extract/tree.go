package extract

import (
	"fmt"
	"os"

	"github.com/corebreaker/ntfstool/core"
)

type IFileReader interface {
	MakeFileStream() (FileStream, error)
}

type Node struct {
	File     *File
	Children map[string]*Node
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

func (self *Node) AddNode(n *Node) {
	self.Children[n.File.Name] = n
}

func (self *Node) AddFile(f *File) *Node {
	n := NewNode(f)
	self.AddNode(n)

	return n
}

func (self *Node) remove(n *Node) {
	delete(self.Children, n.File.Name)
}

func (self *Node) walk(stream *tFileStream) {
	file := self.File
	stream.SendRecord(uint(file.Index), file.Position, file)

	for _, n := range self.Children {
		n.walk(stream)
	}
}

func NewNode(f *File) *Node {
	return &Node{
		File:     f,
		Children: make(map[string]*Node),
	}
}

type Tree struct {
	Mfts  map[string]string
	Nodes map[string]*Node
	Roots map[string]*Node
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

	return self.get_node_path(self.Nodes[parent], res)
}

func (self *Tree) GetNodePath(node *Node) string {
	return self.get_node_path(node, "")
}

func (self *Tree) GetFilePath(file *File) string {
	return self.get_node_path(self.Nodes[file.Id], "")
}

func (self *Tree) GetFilePathFromFile(file IFile) string {
	return self.get_node_path(self.Nodes[file.GetId()], "")
}

func (self *Tree) GetRoot(mft string) *Node {
	id, ok := self.Mfts[mft]
	if !ok {
		return nil
	}

	return self.Roots[id]
}

func (self *Tree) Remove(node *Node) {
	file := node.File
	parent, ok := self.Nodes[file.Parent]
	if ok {
		parent.remove(node)
	}

	idx := file.Index

	for _, n := range self.Nodes {
		if n.File.Index > idx {
			n.File.Index--
		}
	}

	id := file.Id
	_, is_root := self.Roots[id]
	if is_root {
		delete(self.Roots, id)
		delete(self.Mfts, file.Mft)
	}
}

func (self *Tree) GetRootID(mft string) string {
	root := self.GetRoot(mft)
	if root == nil {
		return ""
	}

	return root.File.Id
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
	start, ok := self.Nodes[id]
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

	var nodes []*Node

	mfts := make(map[string]string)
	roots := make(map[string]*Node)
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

		t := NewNode(file)
		id := file.Id

		id_map[id] = t
		nodes = append(nodes, t)

		if file.IsRoot() && file.IsDir() {
			roots[id] = t
			mfts[file.Mft] = id
		}
	}

	for _, n := range nodes {
		parent, found := id_map[n.File.Parent]
		if !found {
			continue
		}

		parent.AddNode(n)
	}

	res := &Tree{
		Mfts:  mfts,
		Nodes: id_map,
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
