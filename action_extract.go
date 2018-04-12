package main

import (
	"fmt"

	"essai/ntfstool/core"
	"essai/ntfstool/extract"
)

func do_file_count(arg *tActionArg) error {
	src, err := arg.GetInput()
	if err != nil {
		return err
	}

	fmt.Println("Reading")
	files, err := extract.MakeFileReader(src)
	if err != nil {
		return err
	}

	fmt.Println("Count=", files.GetCount())

	return nil
}

func do_list_files(dir string, arg *tActionArg) error {
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

	if dir != "" {
		position, err := core.ToInt(dir)
		if err != nil {
			return err
		}

		node, ok := tree.Positions[position]
		if !ok {
			return core.WrapError(fmt.Errorf("Bad position: %d", position))
		}

		fmt.Println("Path:", tree.GetNodePath(node))

		node_list = node.Children
	}

	for _, node := range node_list {
		var typ string

		if node.IsDir() {
			typ = ", Dir"
		} else if node.IsFile() {
			typ = ", File"
		}

		fmt.Println(fmt.Sprintf("   - %d (%s, children=%d%s)", node.File.Position, node.File.Name, len(node.Children), typ))
	}

	return nil
}

func do_copy_file(file int64, arg *tActionArg) error {
	src, dest, err := arg.GetTransferFiles()
	if err != nil {
		return err
	}

	fmt.Println("Reading")
	tree, err := extract.ReadTreeFromFile(src)
	if err != nil {
		return err
	}

	node, ok := tree.Positions[file]
	if !ok {
		return core.WrapError(fmt.Errorf("Bad position: %d", file))
	}

	disk := arg.disk.GetDisk()
	defer core.DeferedCall(disk.Close)

	_, err = extract.SaveFile(disk, node.File, dest)

	return err
}
