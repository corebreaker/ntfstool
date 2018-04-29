package main

import (
	"fmt"

	ntfs "github.com/corebreaker/ntfstool/core"
	"github.com/corebreaker/ntfstool/extract"
)

func do_make_dir(name string, arg *tActionArg) error {
	if len(name) == 0 {
		return ntfs.WrapError(fmt.Errorf("No name specified for the new directory"))
	}

	src, err := arg.GetFile()
	if err != nil {
		return err
	}

	fmt.Println("Reading", src.Name())
	file, err := extract.MakeFileModifier(src)
	if err != nil {
		return err
	}

	defer ntfs.DeferedCall(file.Close)

	tree, err := extract.MakeTree(file)
	if err != nil {
		return err
	}

	parent_id := arg.GetIntoParam()
	if len(parent_id) == 0 {
		mft, ok := arg.GetExt("into-mft")
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
			return ntfs.WrapError(fmt.Errorf("No parent specified"))
		}
	}

	parent_node, ok := tree.Nodes[parent_id]
	if !ok {
		return ntfs.WrapError(fmt.Errorf("Parent `%s` not found", parent_id))
	}

	if _, exists := parent_node.Children[name]; exists {
		return ntfs.WrapError(fmt.Errorf("Name `%s` already exists in `%s`", name, parent_node.File))
	}

	parent_path := tree.GetNodePath(parent_node)
	parent := parent_node.File

	res := &extract.File{
		Id:        ntfs.NewFileId(),
		Mft:       parent.Mft,
		Parent:    parent.Id,
		ParentIdx: parent.Index,
		Origin:    parent.Origin,
		Index:     int64(file.GetCount()),
		Name:      name,
	}

	const msg = "Making directory `%s` with new ID `%s` to directory `%s` (DirID=%s, RootID=%s)"

	fmt.Printf(msg, name, res.Id, parent_path, parent.Id, tree.GetRootID(parent.Mft))
	fmt.Println()

	return file.Write(res)
}

func do_move_to(node_pattern string, arg *tActionArg) error {
	if len(node_pattern) == 0 {
		return ntfs.WrapError(fmt.Errorf("No file id specified"))
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

	defer ntfs.DeferedCall(file.Close)

	tree, err := extract.MakeTree(file)
	if err != nil {
		return err
	}

	pattern, err := parseNodePattern(node_pattern, tree)
	if err != nil {
		return err
	}

	dir_id := arg.GetIntoParam()
	if len(dir_id) == 0 {
		mft, ok := arg.GetExt("into-mft")
		if ok {
			dir_id = func() string {
				for _, root := range tree.Roots {
					id := root.File.Mft
					if id == mft {
						return id
					}
				}

				return ""
			}()
		}

		if len(dir_id) == 0 {
			return ntfs.WrapError(fmt.Errorf("No destination directory id specified for the new directory"))
		}
	}

	dir_node, ok := tree.Nodes[dir_id]
	if !ok {
		return ntfs.WrapError(fmt.Errorf("Destination directory `%s` not found", dir_id))
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

	src_nodes := pattern.GetNodes(from_node)
	if len(src_nodes) == 0 {
		fmt.Println("No moved node")

		return nil
	}

	for _, src_node := range src_nodes {
		src := src_node.File

		if _, exists := dir_node.Children[src.Name]; exists {
			const msg = "Name `%s` (from file `%s`, FileId=%s, RootId=%s) already exists in `%s` (DirID=%s, RootID=%s)"

			src_path := tree.GetNodePath(src_node)
			dir_path := tree.GetNodePath(dir_node)

			f := src_node.File
			fr := tree.GetRootID(f.Mft)
			d := dir_node.File
			dr := tree.GetRootID(d.Mft)

			return ntfs.WrapError(fmt.Errorf(msg, f.Name, src_path, f.Id, fr, dir_path, d.Id, dr))
		}
	}

	for _, src_node := range src_nodes {
		src_path := tree.GetNodePath(src_node)
		src := src_node.File

		dir_path := tree.GetNodePath(dir_node)
		dir := dir_node.File

		src.Parent = dir.Id
		src.ParentIdx = dir.Index

		const msg = "Moving file `%s` (FileID=%s, RootID=%s) to directory `%s` (DirID=%s, RootID=%s)"

		fmt.Printf(msg, src_path, src.Id, tree.GetRootID(src.Mft), dir_path, dir.Id, tree.GetRootID(dir.Mft))
		fmt.Println()

		if err := file.SetRecordAt(int(src.Index), src); err != nil {
			return err
		}
	}

	return nil
}

func do_copy_to(node_pattern string, arg *tActionArg) error {
	if len(node_pattern) == 0 {
		return ntfs.WrapError(fmt.Errorf("No file id specified"))
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

	defer ntfs.DeferedCall(file.Close)

	tree, err := extract.MakeTree(file)
	if err != nil {
		return err
	}

	pattern, err := parseNodePattern(node_pattern, tree)
	if err != nil {
		return err
	}

	type tCopier struct {
		do func(src_node, dir_node *extract.Node) error
	}

	var copier tCopier

	copier.do = func(src_node, dir_node *extract.Node) error {
		src_path, dir_path := tree.GetNodePath(src_node), tree.GetNodePath(dir_node)
		src, dir := src_node.File, dir_node.File

		res := &extract.File{
			Id:        ntfs.NewFileId(),
			Parent:    dir.Id,
			ParentIdx: dir.Index,
			Index:     int64(file.GetCount()),
			Mft:       src.Mft,
			Origin:    src.Origin,
			Name:      src.Name,
			FileRef:   src.FileRef,
			Position:  src.Position,
			Size:      src.Size,
			RunList:   src.RunList,
		}

		const msg = "Copy File `%s` (RootID=%s) with new ID `%s` to directory `%s` (DirID=%s, RootID=%s)"

		fmt.Printf(msg, src_path, tree.GetRootID(res.Mft), res.Id, dir_path, dir.Id, tree.GetRootID(dir.Mft))
		fmt.Println()

		if err := file.Write(res); err != nil {
			return err
		}

		if src.IsDir() {
			node := dir_node.AddFile(res)
			tree.Nodes[res.Id] = node

			for _, child := range src_node.Children {
				if err := copier.do(child, node); err != nil {
					return err
				}
			}
		}

		return nil
	}

	dir_id := arg.GetIntoParam()
	if len(dir_id) == 0 {
		mft, ok := arg.GetExt("into-mft")
		if ok {
			dir_id = func() string {
				for _, root := range tree.Roots {
					id := root.File.Mft
					if id == mft {
						return id
					}
				}

				return ""
			}()
		}

		if len(dir_id) == 0 {
			return ntfs.WrapError(fmt.Errorf("No destination directory id specified for the new directory"))
		}
	}

	dir_node, ok := tree.Nodes[dir_id]
	if !ok {
		return ntfs.WrapError(fmt.Errorf("Destination directory `%s` not found", dir_id))
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

	src_nodes := pattern.GetNodes(from_node)
	if len(src_nodes) == 0 {
		fmt.Println("No removed node")

		return nil
	}

	for _, src_node := range src_nodes {
		src := src_node.File

		if _, exists := dir_node.Children[src.Name]; exists {
			const msg = "Name `%s` (from file `%s`, FileId=%s, RootId=%s) already exists in `%s` (DirID=%s, RootID=%s)"

			src_path := tree.GetNodePath(src_node)
			dir_path := tree.GetNodePath(dir_node)

			f := src_node.File
			fr := tree.GetRootID(f.Mft)
			d := dir_node.File
			dr := tree.GetRootID(d.Mft)

			return ntfs.WrapError(fmt.Errorf(msg, f.Name, src_path, f.Id, fr, dir_path, d.Id, dr))
		}
	}

	for _, src_node := range src_nodes {
		if err := copier.do(src_node, dir_node); err != nil {
			return err
		}
	}

	return nil
}

func do_remove_from(node_pattern string, arg *tActionArg) error {
	if len(node_pattern) == 0 {
		return ntfs.WrapError(fmt.Errorf("No file id specified"))
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

	defer ntfs.DeferedCall(file.Close)

	tree, err := extract.MakeTree(file)
	if err != nil {
		return err
	}

	pattern, err := parseNodePattern(node_pattern, tree)
	if err != nil {
		return err
	}

	type tRemover struct {
		do func(*extract.Node) error
	}

	var remover tRemover

	remover.do = func(node *extract.Node) error {
		for _, child := range node.Children {
			if err := remover.do(child); err != nil {
				return err
			}
		}

		src := node.File

		path := tree.GetNodePath(node)

		fmt.Printf("Remove File `%s` (FileID=%s, RootID=%s)", path, src.Id, tree.GetRootID(src.Mft))
		fmt.Println()

		if err := file.DelRecordWithId(src.Id); err != nil {
			return err
		}

		tree.Remove(node)

		return nil
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

	src_nodes := pattern.GetNodes(from_node)
	if len(src_nodes) == 0 {
		fmt.Println("No removed node")

		return nil
	}

	for _, src_node := range src_nodes {
		if err := remover.do(src_node); err != nil {
			return err
		}
	}

	return nil
}
