package main

import (
	"bytes"
	"fmt"
	"sort"

	"essai/ntfstool/core"
	"essai/ntfstool/core/data"
	"essai/ntfstool/extract"
	"essai/ntfstool/inspect"
)

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

	type tMft struct {
		state *inspect.StateMft
		list  []*inspect.StateFileRecord
		files map[string]*extract.File
		refs  map[data.FileRef]string
		root  *extract.File
		lost  *extract.File
	}

	mfts := make(map[string]*tMft)
	file_cnt := 0

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

		mft, ok := mfts[id]
		if !ok {
			mft = &tMft{
				refs:  make(map[data.FileRef]string),
				files: make(map[string]*extract.File),
				lost: &extract.File{
					Id:   core.NewFileId(),
					Mft:  id,
					Name: "Lost+Found",
				},
			}

			mfts[id] = mft
		}

		switch rec.GetType() {
		case inspect.STATE_RECORD_TYPE_FILE:
			record := rec.(*inspect.StateFileRecord)

			mft.list = append(mft.list, record)
			file_cnt++

		case inspect.STATE_RECORD_TYPE_MFT:
			mft.state = rec.(*inspect.StateMft)
		}
	}

	fmt.Println("\r100%                                                      ")

	fmt.Println("Make list")
	i, cnt = 0, file_cnt

	var res_list []*extract.File

	for mftid, mft := range mfts {
		res_list = append(res_list, mft.lost)

		for _, file := range mft.list {
			fmt.Printf("\rDone: %d %%", 100*i/cnt)
			i++

			id := core.NewFileId()
			is_dir := file.IsDir()
			position := file.Position

			ref := file.Reference
			name := file.Name

			var runlist core.RunList
			var size uint64

			if !is_dir {
				attrs := file.GetAttributes(core.ATTR_DATA)
				if len(attrs) == 0 {
					continue
				}

				attr_state := attrs[0]

				attr_data, err := file.Header.MakeAttributeFromHeader(&attr_state.Header)
				if err != nil {
					return err
				}

				size = attr_data.GetSize()

				origin := mft.state.PartOrigin

				for _, entry := range attr_state.RunList {
					is_zero := entry.Zero

					start := entry.Start
					if is_zero {
						start = core.ClusterNumber(int64(start) + origin)
					}

					runlist = append(runlist, &core.RunEntry{
						Start: start,
						Count: entry.Count,
						Zero:  is_zero,
					})
				}
			}

			f := &extract.File{
				Id:        id,
				FileRef:   ref,
				ParentRef: file.Parent,
				Mft:       mftid,
				Position:  position,
				Size:      size,
				Name:      name,
				RunList:   runlist,
			}

			mft.refs[ref] = id
			res_list = append(res_list, f)

			if file.Header.MftRecordNumber == 5 {
				mft.root = f
			} else {
				mft.files[id] = f
			}
		}
	}

	fmt.Println("\r100%                                                      ")

	fmt.Println("Completing directory hierarchy")

	var log bytes.Buffer

	no_parents := 0
	i = 0

	for mftid, mft := range mfts {
		if mft.root == nil {
			seq := func() uint16 {
				for _, f := range mft.files {
					if f.ParentRef.GetFileIndex() == 5 {
						return f.ParentRef.GetSequenceNumber()
					}
				}

				return 0
			}()

			if seq == 0 {
				seq++
			}

			f := &extract.File{
				Id:      core.NewFileId(),
				FileRef: data.MakeFileRef(seq, 5),
				Mft:     mftid,
				Name:    ".",
			}

			res_list = append(res_list, f)
			mft.root = f
		}

		mft.lost.Parent = mft.root.Id
		mft.lost.ParentRef = mft.root.FileRef

		for _, file := range mft.files {
			fmt.Printf("\rDone: %d %%", 100*i/cnt)
			i++

			parent, ok := mft.refs[file.ParentRef]
			if !ok {
				fmt.Fprintf(&log, fmt.Sprintf("Parent not found for file %s", file))
				fmt.Fprintln(&log)

				no_parents++

				parent = mft.lost.Id
			}

			file.Parent = parent
		}
	}

	fmt.Println("\r100%                                                      ")

	fmt.Println("Sorting")

	sort.Slice(res_list, func(i, j int) bool {
		f1, f2 := res_list[i], res_list[j]
		if f1.Mft != f1.Mft {
			return false
		}

		if f1.IsRoot() {
			return true
		}

		id, mft := f1.Id, mfts[f1.Mft]

		get_parent := func(f *extract.File) *extract.File {
			if (f == nil) || (len(f.Parent) == 0) {
				return nil
			}

			id := f.Parent

			if mft.root.Id == id {
				return mft.root
			}

			if mft.lost.Id == id {
				return mft.lost
			}

			return mft.files[id]
		}

		for f := f2; f != nil; f = get_parent(f) {
			if f.Parent == id {
				return true
			}
		}

		return false
	})

	fmt.Println("Make index map")
	indexes := make(map[string]int64)

	cnt = len(res_list)
	for i, file := range res_list {
		fmt.Printf("\rDone: %d %%", 100*i/cnt)

		indexes[file.Id] = int64(i)
	}

	fmt.Println("\r100%                                                      ")

	fmt.Println("Make index map")

	for i, file := range res_list {
		fmt.Printf("\rDone: %d %%", 100*i/cnt)

		file.ParentIdx = indexes[file.Parent]
	}

	fmt.Println("\r100%                                                      ")

	fmt.Println("Writing")

	writer, err := extract.MakeFileWriter(dest)
	if err != nil {
		return err
	}

	defer core.DeferedCall(writer.Close)

	for i, entry := range res_list {
		fmt.Printf("\rDone: %d %%", 100*i/cnt)

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
