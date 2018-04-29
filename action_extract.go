package main

import (
	"fmt"
	"os"
	"strings"

	ntfs "github.com/corebreaker/ntfstool/core"
	"github.com/corebreaker/ntfstool/core/data"
	"github.com/corebreaker/ntfstool/extract"

	"github.com/siddontang/go/ioutil2"
)

func do_show_id(id string, arg *tActionArg) error {
	var src *os.File
	var err error

	new_name, rename := arg.GetExt("rename")

	if rename {
		src, err = arg.GetFile()
		if err != nil {
			return err
		}
	} else {
		src, err = arg.GetInput()
		if err != nil {
			return err
		}
	}

	fmt.Println("Reading")
	files, err := extract.MakeFileModifier(src)
	if err != nil {
		return err
	}

	defer ntfs.DeferedCall(files.Close)

	rec0, err := files.GetRecordAt(0)
	if err != nil {
		return err
	}

	index, ok := rec0.(*extract.Index)
	if !ok {
		return ntfs.WrapError(fmt.Errorf("Bad file format"))
	}

	i, found := index.IdMap[id]
	if !found {
		fmt.Println()
		fmt.Println("Id", id, "has not found.")
	}

	file, err := files.GetRecordAt(int(i))
	if err != nil {
		return err
	}

	if rename && (len(new_name) > 0) {
		file.SetName(new_name)
		if err := files.SetRecordAt(int(i), file); err != nil {
			return err
		}
	}

	fmt.Println()
	fmt.Println("Index=   ", file.GetIndex())
	fmt.Println("Is Dir?= ", file.IsDir())
	fmt.Println("Is File?=", file.IsFile())
	fmt.Println("Is Root?=", file.IsRoot())

	fmt.Println()
	fmt.Println("Record:")
	file.Print()

	return nil
}

func do_show_parent(id string, arg *tActionArg) error {
	src, err := arg.GetInput()
	if err != nil {
		return err
	}

	fmt.Println("Reading")
	files, err := extract.MakeFileReader(src)
	if err != nil {
		return err
	}

	defer ntfs.DeferedCall(files.Close)

	stream, err := files.MakeStream()
	if err != nil {
		return err
	}

	defer ntfs.DeferedCall(stream.Close)

	fmt.Println()
	fmt.Println("Results:")
	for entry := range stream {
		record := entry.Record()
		file := record.GetFile()

		if file.Parent != id {
			continue
		}

		fmt.Println("  -", file.Id, "(", file.Name, ")")
	}

	fmt.Println()

	return nil
}

func do_show_parent_ref(ref int64, arg *tActionArg) error {
	mft, has_mft := arg.GetExt("with-mft")
	if !has_mft {
		return ntfs.WrapError(fmt.Errorf("MFT Id is missing"))
	}

	src, err := arg.GetInput()
	if err != nil {
		return err
	}

	fmt.Println("Reading")
	files, err := extract.MakeFileReader(src)
	if err != nil {
		return err
	}

	defer ntfs.DeferedCall(files.Close)

	stream, err := files.MakeStream()
	if err != nil {
		return err
	}

	defer ntfs.DeferedCall(stream.Close)

	file_idx := data.FileIndex(ref)

	fmt.Println()
	fmt.Println("Results:")
	for entry := range stream {
		record := entry.Record()
		file := record.GetFile()

		if (file.Mft != mft) && (file.ParentRef.GetFileIndex() == file_idx) {
			continue
		}

		fmt.Println("  -", file.Id, "(", file.Name, ")")
	}

	fmt.Println()

	return nil
}

func do_list_files(node_pattern string, arg *tActionArg) error {
	src, err := arg.GetInput()
	if err != nil {
		return err
	}

	fmt.Println("Reading")
	tree, err := extract.ReadTreeFromFile(src)
	if err != nil {
		return err
	}

	node_list := tree.Roots
	msg := "Results:"

	if node_pattern != "" {
		matcher, err := parseNodePattern(node_pattern, tree)
		if err != nil {
			return err
		}

		var from_node *extract.Node

		from := arg.GetFromParam()
		if len(from) == 0 {
			node, ok := tree.Nodes[from]
			if !ok {
				return ntfs.WrapError(fmt.Errorf("From ID `%s` not found", from))
			}

			if !node.IsDir() {
				return ntfs.WrapError(fmt.Errorf("From ID `%s` is not a directory", from))
			}

			from_node = node
		}

		node_list = matcher.GetNodes(from_node)
		if len(node_list) == 1 {
			for _, n := range node_list {
				if n.IsDir() {
					path := tree.GetNodePath(n)
					root := tree.GetRootID(n.File.Mft)
					msg = fmt.Sprintf("Results for %s (DirID=%s, RootID=%s):", path, n.File.Id, root)
					node_list = n.Children
				}
			}
		}
	}

	_, nometa := arg.GetExt("nometa")
	_, noempty := arg.GetExt("noempty")

	fmt.Println()
	fmt.Println(msg)

	for _, n := range node_list {
		if (nometa && extract.IsMetaFile(n.File)) || (noempty && n.IsEmpty(noempty)) {
			continue
		}

		infos := n.File.Name

		if n.IsDir() {
			infos += fmt.Sprintf(", Dir {children=%d}", n.ChildCount(nometa))
		}

		if n.IsFile() {
			infos += ", File"
		}

		fmt.Println(fmt.Sprintf("   - %s (%s)", n.File.Id, infos))
	}

	return nil
}

func do_save_file(file string, arg *tActionArg) error {
	src, dest, err := arg.GetTransferFiles(".")
	if err != nil {
		return err
	}

	fmt.Println("Reading")
	tree, err := extract.ReadTreeFromFile(src)
	if err != nil {
		return err
	}

	node, ok := tree.Nodes[file]
	if !ok {
		return ntfs.WrapError(fmt.Errorf("Bad ID: %d", file))
	}

	destname := strings.TrimRight(dest, string([]rune{os.PathSeparator}))
	if len(destname) == 0 {
		destname = "."
	}

	fmt.Println("Writing to:", destname)
	if ioutil2.FileExists(destname) {
		infos, err := os.Stat(destname)
		if err != nil {
			return ntfs.WrapError(err)
		}

		if !infos.IsDir() {
			return ntfs.WrapError(fmt.Errorf("Path `%s` is a file, not a directory.", destname))
		}
	} else {
		if err := os.MkdirAll(destname, 0770); err != nil {
			return ntfs.WrapError(err)
		}
	}

	disk := arg.disk.GetDisk()
	defer ntfs.DeferedCall(disk.Close)

	_, noempty := arg.GetExt("noempty")
	_, nometa := arg.GetExt("nometa")

	_, err = extract.SaveNode(disk, node, destname, noempty, nometa)

	return err
}
