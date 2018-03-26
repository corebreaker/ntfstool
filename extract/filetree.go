package extract

import (
	"essai/ntfstool/core"
	"fmt"
	"os"
)

type Node struct {
	File     *File
	Children []*Node
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

type Tree struct {
	Roots     []*Node
	Positions map[int64]*Node
	Parents   map[string]*Node
}

func (self *Tree) get_node_path(node *Node, suffix string) string {
	if node == nil {
		return suffix
	}

	file := node.File
	parent := file.Parent

	res := file.Name
	if suffix != "" {
		res = fmt.Sprintf("%s/%s", res, suffix)
	}

	if parent == "" {
		return res
	}

	return self.get_node_path(self.Parents[parent], res)
}

func (self *Tree) GetNodePath(node *Node) string {
	return self.get_node_path(node, "")
}

func (self *Tree) GetFilePath(file *File) string {
	return self.get_node_path(self.Positions[file.Position], "")
}

func MakeTree(files *FileReader) (*Tree, error) {
	stream, err := files.MakeStream()
	if err != nil {
		return nil, err
	}

	defer core.DeferedCall(stream.Close)

	res := &Tree{
		Positions: make(map[int64]*Node),
		Parents:   make(map[string]*Node),
	}

	var roots []*Node

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
		res.Parents[file.Id] = t
		res.Positions[file.Position] = t

		if file.Parent == "" {
			roots = append(roots, t)
		} else {
			dir, ok := res.Parents[file.Parent]
			if ok {
				dir.Children = append(dir.Children, t)
			}
		}
	}

	for _, f := range roots {
		if f.IsDir() {
			res.Roots = append(res.Roots, f)
		}
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

	return MakeTree(files)
}
