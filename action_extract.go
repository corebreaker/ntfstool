package main

import (
	"fmt"
	"os"
	"strings"

	"essai/ntfstool/core"
	"essai/ntfstool/core/data"
	"essai/ntfstool/extract"

	"github.com/siddontang/go/ioutil2"
)

func do_show_id(id string, arg *tActionArg) error {
	src, err := arg.GetInput()
	if err != nil {
		return err
	}

	fmt.Println("Reading")
	files, err := extract.MakeFileReader(src)
	if err != nil {
		return err
	}

	defer core.DeferedCall(files.Close)

	stream, err := files.MakeStream()
	if err != nil {
		return err
	}

	defer core.DeferedCall(stream.Close)

	for entry := range stream {
		record := entry.Record()
		file := record.GetFile()

		if file.Id != id {
			continue
		}

		fmt.Println()
		fmt.Println("Index=   ", entry.Index())
		fmt.Println("Is Dir?= ", record.IsDir())
		fmt.Println("Is File?=", record.IsFile())
		fmt.Println("Is Root?=", file.IsRoot())

		fmt.Println()
		fmt.Println("Record:")
		record.Print()

		return nil
	}

	fmt.Println()
	fmt.Println("Id", id, "has not found.")

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

	defer core.DeferedCall(files.Close)

	stream, err := files.MakeStream()
	if err != nil {
		return err
	}

	defer core.DeferedCall(stream.Close)

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
		return core.WrapError(fmt.Errorf("MFT Id is missing"))
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

	defer core.DeferedCall(files.Close)

	stream, err := files.MakeStream()
	if err != nil {
		return err
	}

	defer core.DeferedCall(stream.Close)

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

func do_list_files(file string, arg *tActionArg) error {
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

	if file != "" {
		node, ok := tree.IdMap[file]
		if !ok {
			return core.WrapError(fmt.Errorf("Bad ID: %d", file))
		}

		fmt.Println("Path:", tree.GetNodePath(node))

		node_list = node.Children
	}

	_, nometa := arg.GetExt("nometa")
	_, noempty := arg.GetExt("noempty")

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

func do_copy_file(file string, arg *tActionArg) error {
	src, dest, err := arg.GetTransferFiles(".")
	if err != nil {
		return err
	}

	fmt.Println("Reading")
	tree, err := extract.ReadTreeFromFile(src)
	if err != nil {
		return err
	}

	node, ok := tree.IdMap[file]
	if !ok {
		return core.WrapError(fmt.Errorf("Bad ID: %d", file))
	}

	destname := strings.TrimRight(dest, string([]rune{os.PathSeparator}))
	if len(destname) == 0 {
		destname = "."
	}

	fmt.Println("Writing to:", destname)
	if ioutil2.FileExists(destname) {
		infos, err := os.Stat(destname)
		if err != nil {
			return core.WrapError(err)
		}

		if !infos.IsDir() {
			return core.WrapError(fmt.Errorf("Path `%s` is a file, not a directory.", destname))
		}
	} else {
		if err := os.MkdirAll(destname, 0770); err != nil {
			return core.WrapError(err)
		}
	}

	disk := arg.disk.GetDisk()
	defer core.DeferedCall(disk.Close)

	_, noempty := arg.GetExt("noempty")
	_, nometa := arg.GetExt("nometa")

	_, err = extract.SaveNode(disk, node, destname, noempty, nometa)

	return err
}

func do_make_dir(name string, arg *tActionArg) error {
	if len(name) == 0 {
		return core.WrapError(fmt.Errorf("No name specified for the new directory"))
	}

	src, err := arg.GetFile()
	if err != nil {
		return err
	}

	fmt.Println("Reading")
	file, err := extract.MakeFileModifier(src)
	if err != nil {
		return err
	}

	defer core.DeferedCall(file.Close)

	tree, err := extract.MakeTree(file)
	if err != nil {
		return err
	}

	parent_id := arg.GetToParam()
	if len(parent_id) == 0 {
		mft, ok := arg.GetExt("to-mft")
		if ok {
			parent_id = func() string {
				for _, root := range tree.Roots {
					id := root.File.Mft
					if id == mft {
						return id
					}
				}

				return ""
			}()
		}

		if len(parent_id) == 0 {
			return core.WrapError(fmt.Errorf("No parent specified"))
		}
	}

	parent_node, ok := tree.IdMap[parent_id]
	if !ok {
		return core.WrapError(fmt.Errorf("Parent `%s` not found", parent_id))
	}

	parent := parent_node.File

	return file.Write(&extract.File{
		Id:        core.NewFileId(),
		Mft:       parent.Mft,
		Parent:    parent.Id,
		ParentIdx: parent.Index,
		Origin:    parent.Origin,
		Index:     int64(file.GetCount()),
		Name:      name,
	})
}

func do_move_to(dir_id string, arg *tActionArg) error {
	if len(dir_id) == 0 {
		return core.WrapError(fmt.Errorf("No destination directory id specified for the new directory"))
	}

	src_file, err := arg.GetFile()
	if err != nil {
		return err
	}

	fmt.Println("Reading")
	file, err := extract.MakeFileModifier(src_file)
	if err != nil {
		return err
	}

	defer core.DeferedCall(file.Close)

	tree, err := extract.MakeTree(file)
	if err != nil {
		return err
	}

	src_id, ok := arg.GetExt("id")
	if !ok {
		return core.WrapError(fmt.Errorf("No file id specified"))
	}

	src_node, ok := tree.IdMap[src_id]
	if !ok {
		return core.WrapError(fmt.Errorf("Source file `%s` not found", src_id))
	}

	src := src_node.File

	dir_node, ok := tree.IdMap[dir_id]
	if !ok {
		return core.WrapError(fmt.Errorf("Destination directory `%s` not found", dir_id))
	}

	dir := dir_node.File

	src.Parent = dir.Id
	src.ParentIdx = dir.Index

	return file.SetRecordAt(int(src.Index), src)
}
