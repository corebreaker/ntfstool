package main

import (
	"bytes"
	"fmt"

	"github.com/pborman/uuid"

	"essai/ntfstool/core"
	"essai/ntfstool/core/dataio"
	"essai/ntfstool/extract"
	"essai/ntfstool/inspect"
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

func do_mkfilelist(verbose bool, arg *tActionArg) error {
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
	files := make(map[string]map[dataio.FileIndex]*inspect.StateFileRecord)
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
				sub_list = make(map[dataio.FileIndex]*inspect.StateFileRecord)
				files[id] = sub_list
			}

			sub_list[record.Reference.GetFileIndex()] = record
			file_list = append(file_list, record)

			/**

			            name_sub_list, ok := names[id]
						if !ok {
							name_sub_list = make(map[int64]map[int64]string)
							names[id] = name_sub_list
						}

						contents, ok := name_sub_list[record.Reference.GetFileIndex()]
						if !ok {
							contents = make(map[int64]string)
							name_sub_list[record.Parent.GetFileIndex()] = contents
						}

			            contents[record.Reference.GetFileIndex()] = record.Name

			*/

		case inspect.STATE_RECORD_TYPE_INDEX:
			sub_list, ok := names[id]
			if !ok {
				sub_list = make(map[int64]map[int64]string)
				names[id] = sub_list
			}

			/*
				            record := rec.(*inspect.StateIndexRecord)

									contents, ok := sub_list[int64(record.Reference.GetFileIndex())]
										if !ok {
											contents = make(map[int64]string)
											sub_list[int64(record.Reference.GetFileIndex())] = contents
										}

										for _, file := range record.Entries {
											if file.Header.FilenameType != 1 {
												continue
											}

											contents[int64(file.Header.FileReferenceNumber.GetFileIndex())] = file.Name
										}
			*/

		case inspect.STATE_RECORD_TYPE_MFT:
			mfts[id] = rec.(*inspect.StateMft)
		}
	}

	fmt.Println("\r100%                                                      ")

	fmt.Println("Make list")
	cnt = len(file_list)

	file_id_map := make(map[string]map[dataio.FileIndex]*extract.File)
	res_list := make([]*extract.File, 0)

	var log bytes.Buffer

	no_parents := 0

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

		ref := file.Reference
		//name := names[mft][file.Parent.GetFileIndex()][ref]
		name := file.Name

		runlist := core.RunList(nil)
		if file.IsDir() {
			runlist_src := attr_data.GetRunList()

			runlist_size := len(runlist_src)
			runlist = make(core.RunList, runlist_size)

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
			FileRef:  ref,
			Mft:      mft,
			Position: position,
			Size:     size,
			Name:     name,
			RunList:  runlist,
		}

		res_list = append(res_list, f)

		sub_map, ok := file_id_map[mft]
		if !ok {
			sub_map = make(map[dataio.FileIndex]*extract.File)
			file_id_map[mft] = sub_map
		}

		sub_map[ref.GetFileIndex()] = f
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

		file := files[mft][entry.FileRef.GetFileIndex()]
		parent := file.Parent

		if parent != 0 {
			f, ok := file_id_map[mft][parent.GetFileIndex()]
			if !ok {
				fmt.Fprintf(&log, fmt.Sprintf("Parent not found for file %s", file))
				fmt.Fprintln(&log)

				no_parents++

				continue
			}

			entry.Parent = f.Id
			entry.ParentRef = f.FileRef
		}

		if err := writer.Write(entry); err != nil {
			return err
		}
	}

	fmt.Println("\r100%                                                      ")

	fmt.Println()
	fmt.Println("File with no parent:", no_parents)

	if verbose {
		fmt.Println()
		fmt.Println("Details:")
		fmt.Println(&log)
	}

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
