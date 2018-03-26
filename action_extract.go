package main

import (
	"essai/ntfstool/core"
	"essai/ntfstool/extract"
	"essai/ntfstool/inspect"
	"fmt"
	"os"

	"github.com/pborman/uuid"
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

func do_mkfilelist(arg *tActionArg) error {
	src, dest, err := arg.GetFiles()
	if err != nil {
		return err
	}

	fmt.Println("Reading")
	reader, err := inspect.MakeStateReader(src)
	if err != nil {
		return err
	}

	defer core.DeferedCall(reader.Close)

	stream, err := reader.MakeStream()
	if err != nil {
		return err
	}

	mfts := make(map[string]*inspect.StateMft)
	names := make(map[string]map[int64]map[int64]string)
	files := make(map[string]map[int64]*inspect.StateFileRecord)
	file_list := make([]*inspect.StateFileRecord, 0)

	i, cnt := 0, reader.GetCount()

	fmt.Println("Spliting states")
	for item := range stream {
		fmt.Printf("\rDone: %d %%", 100*i/cnt)
		i++

		rec := item.Record()

		if err := rec.GetError(); err != nil {
			return err
		}

		if rec.IsNull() {
			continue
		}

		id := rec.GetMftId()

		switch rec.GetType() {
		case inspect.STATE_RECORD_TYPE_FILE:
			record := rec.(*inspect.StateFileRecord)

			sub_list, ok := files[id]
			if !ok {
				sub_list = make(map[int64]*inspect.StateFileRecord)
				files[id] = sub_list
			}

			sub_list[record.Reference.GetFileIndex()] = record
			file_list = append(file_list, record)

		case inspect.STATE_RECORD_TYPE_INDEX:
			sub_list, ok := names[id]
			if !ok {
				sub_list = make(map[int64]map[int64]string)
				names[id] = sub_list
			}

			record := rec.(*inspect.StateIndexRecord)

			contents, ok := sub_list[record.Reference.GetFileIndex()]
			if !ok {
				contents = make(map[int64]string)
				sub_list[record.Reference.GetFileIndex()] = contents
			}

			for _, file := range record.Entries {
				if file.Header.FilenameType != 1 {
					continue
				}

				contents[file.Header.FileReferenceNumber.GetFileIndex()] = file.Name
			}

		case inspect.STATE_RECORD_TYPE_MFT:
			mfts[id] = rec.(*inspect.StateMft)
		}
	}

	fmt.Println("\r100%                                                      ")

	fmt.Println("Make list")
	cnt = len(file_list)

	file_id_map := make(map[string]map[int64]*extract.File)
	res_list := make([]*extract.File, 0)

	for i, file := range file_list {
		fmt.Printf("\rDone: %d %%", 100*i/cnt)

		attrs := file.Header.GetAttributeFilteredList(core.ATTR_DATA)
		if len(attrs) == 0 {
			continue
		}

		attr_data, err := file.Header.MakeAttributeFromOffset(attrs[0])
		if err != nil {
			return err
		}

		id := uuid.New()
		position := file.Position
		size := attr_data.GetSize()

		mft := file.MftId

		ref := file.Reference.GetFileIndex()
		name := names[mft][file.Parent.GetFileIndex()][ref]

		runlist := core.RunList(nil)
		if (file.Header.Flags & core.FFLAG_DIRECTORY) != core.FFLAG_NONE {
			runlist_src := attr_data.GetRunList()

			runlist_size := len(runlist_src)
			runlist := make(core.RunList, runlist_size)

			origin := core.ClusterNumber(mfts[file.MftId].PartOrigin)

			for j, entry := range runlist_src[1:] {
				runlist[j] = &core.RunEntry{
					Start: origin + entry.Start,
					Count: entry.Count,
					Zero:  entry.Zero,
				}
			}
		}

		f := &extract.File{
			Id:       id,
			Ref:      ref,
			Mft:      mft,
			Position: position,
			Size:     size,
			Name:     name,
			RunList:  runlist,
		}

		res_list = append(res_list, f)

		sub_map, ok := file_id_map[mft]
		if !ok {
			sub_map = make(map[int64]*extract.File)
			file_id_map[mft] = sub_map
		}

		sub_map[ref] = f
	}

	fmt.Println("\r100%                                                      ")

	fmt.Println("Completing directory hierarchy")
	cnt = len(res_list)

	writer, err := extract.MakeFileWriter(dest)
	if err != nil {
		return err
	}

	defer core.DeferedCall(writer.Close)

	for i, entry := range res_list {
		fmt.Printf("\rDone: %d %%", 100*i/cnt)

		mft := entry.Mft

		file := files[mft][entry.Ref]
		parent := file.Parent

		if parent != 0 {
			f, ok := file_id_map[mft][parent.GetFileIndex()]
			if !ok {
				fmt.Fprintln(os.Stderr, fmt.Sprintf("Parent not found for file %s", file))

				continue
			}

			entry.Parent = f.Id
		}

		if err := writer.Write(entry); err != nil {
			return err
		}
	}

	fmt.Println("\r100%                                                      ")

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
