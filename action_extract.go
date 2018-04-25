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

		if file == nil {
			record.Print()

			return nil
		}

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

		node_list = matcher.GetNodes()
		if len(node_list) == 1 {
			for _, n := range node_list {
				if n.IsDir() {
					msg = fmt.Sprintf("Results for %s:", n.File)
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
