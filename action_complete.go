package main

import (
	"bytes"
	"fmt"

	ntfs "github.com/corebreaker/ntfstool/core"
	"github.com/corebreaker/ntfstool/core/data"
	"github.com/corebreaker/ntfstool/inspect"
)

func do_complete(arg *tActionArg) error {
	src, dest, err := arg.GetFiles()
	if err != nil {
		return err
	}

	fmt.Println("Reading")
	states, err := inspect.MakeStateReader(src)
	if err != nil {
		return err
	}

	defer ntfs.DeferedCall(states.Close)

	stream, err := states.MakeStream()
	if err != nil {
		return err
	}

	defer ntfs.DeferedCall(stream.Close)

	var files []*inspect.StateFileRecord
	var records []inspect.IStateRecord

	type tDirEntry struct {
		dir  data.FileRef
		name string
	}

	type tMftEntry struct {
		state *inspect.StateMft
		root  data.FileRef
		dirs  map[data.FileIndex]*inspect.StateFileRecord
		files map[data.FileRef]*tDirEntry
	}

	mfts := make(map[string]*tMftEntry)
	dircount := 0

	get_mft := func(id string) *tMftEntry {
		res, ok := mfts[id]
		if !ok {
			res = &tMftEntry{
				dirs:  make(map[data.FileIndex]*inspect.StateFileRecord),
				files: make(map[data.FileRef]*tDirEntry),
			}

			mfts[id] = res
		}

		return res
	}

	add_dir_entries := func(mft *tMftEntry, dir *inspect.StateFileRecord, attr *inspect.StateAttribute) (bool, error) {
		desc, err := dir.GetAttributeDesc(attr)
		if err != nil {
			return false, err
		}

		disk := arg.disk.GetDisk()
		defer disk.Close()

		disk.SetOffset(mft.state.PartOrigin)

		val, err := desc.GetValue(disk)
		if err != nil {
			return false, err
		}

		if val == nil {
			return false, nil
		}

		entry, err := val.GetFirstEntry()
		if err != nil {
			return false, err
		}

		for ii := 0; entry != nil; ii++ {
			var out bytes.Buffer

			ntfs.FprintStruct(&out, entry)

			mft.files[entry.FileReferenceNumber] = &tDirEntry{
				name: entry.Name,
				dir:  dir.Reference,
			}

			entry, err = val.GetNextEntry(entry)
			if err != nil {
				return false, err
			}
		}

		return true, nil
	}

	i, sz := 0, states.GetCount()

	fmt.Println("Record sorting")
	for item := range stream {
		fmt.Printf("\rDone: %d %%", i*100/sz)
		i++

		record := item.Record()
		if err := record.GetError(); err != nil {
			return err
		}

		if record.IsNull() {
			continue
		}

		switch record.GetType() {
		case inspect.STATE_RECORD_TYPE_MFT:
			mft := get_mft(record.GetMftId())
			mft.state = record.(*inspect.StateMft)

			records = append(records, record)

		case inspect.STATE_RECORD_TYPE_FILE:
			r := record.(*inspect.StateFileRecord)

			if r.IsDir() {
				mft := get_mft(r.MftId)
				mft.dirs[r.Reference.GetFileIndex()] = r
				if r.Header.MftRecordNumber == 5 {
					mft.root = r.Reference
				}

				dircount++
			}

			files = append(files, r)

		default:
			records = append(records, record)
		}
	}

	fmt.Println("\rDone: 100 %")

	// Gets root directories if not exist
	for mftid, mft := range mfts {
		if !mft.root.IsNull() {
			continue
		}

		var record ntfs.FileRecord

		arg.disk.SetStart(mft.state.PartOrigin)
		arg.disk.SetMftShift(int64(mft.state.RunList[0].Start) * 4096)

		if err := arg.disk.ReadFileRecord(5, &record); err != nil {
			return err
		}

		state := &inspect.StateFileRecord{
			StateBase: inspect.StateBase{
				Position: mft.state.PartOrigin + (int64(mft.state.RunList[0].Start) * 4096) + 5120,
				MftId:    mftid,
			},
			Header: record,
		}

		err := func() error {
			disk := arg.disk.GetDisk()
			defer disk.Close()

			_, err := state.Init(disk)

			return err
		}()

		if err != nil {
			return err
		}

		state.Reference = mft.state.GetReference(state)

		mft.dirs[state.Reference.GetFileIndex()] = state
		files = append(files, state)

		mft.root = state.Reference

		dircount++
	}

	fmt.Println("Getting names")
	sz = len(files)
	for i, file := range files {
		fmt.Printf("\rDone: %d %%", i*100/sz)

		mft := get_mft(file.MftId)

		err := func() error {
			disk := arg.disk.GetDisk()
			defer disk.Close()

			disk.SetOffset(mft.state.PartOrigin)

			parent := data.FileRef(0)
			fname := ""

			for _, attr := range file.Attributes {
				if attr.Header.AttributeType != ntfs.ATTR_FILE_NAME {
					continue
				}

				desc, err := file.GetAttributeDesc(attr)
				if err != nil {
					return err
				}

				attr_val, err := desc.GetValue(disk)
				if err != nil {
					return err
				}

				name := attr_val.GetFilename()

				if attr_val.IsLongName() {
					fname = name
					parent = attr_val.GetParent()
				} else {
					if fname == "" {
						fname = name
					}

					if parent.IsNull() {
						parent = attr_val.GetParent()
					}
				}

				file.Names = append(file.Names, name)
			}

			file.Parent = parent
			file.Name = fname

			return nil
		}()

		if err != nil {
			return err
		}
	}

	fmt.Println("\rDone: 100 %")

	to_remove := make(map[*inspect.StateFileRecord]bool)

	fmt.Println("Scanning directories")

	processed_dircount := 0
	progress := func() {
		fmt.Printf("\rDone: %d %%", processed_dircount*100/dircount)
		processed_dircount++
	}

	for _ /*mftid*/, mft := range mfts {
		dirs := make(map[data.FileIndex]*inspect.StateFileRecord)
		for idx, dir := range mft.dirs {
			progress()

			attrs := dir.GetAttributes(ntfs.ATTR_INDEX_ROOT, ntfs.ATTR_INDEX_ALLOCATION)
			if len(attrs) == 0 {
				to_remove[dir] = true
				continue
			}

			ok, err := add_dir_entries(mft, dir, attrs[0])
			if err != nil {
				return err
			}

			if !ok {
				to_remove[dir] = true
				continue
			}

			if len(attrs) > 1 {
				if _, err := add_dir_entries(mft, dir, attrs[1]); err != nil {
					return err
				}
			}

			dirs[idx] = dir
		}

		mft.dirs = dirs
	}

	fmt.Println("\rDone: 100 %")

	fmt.Println("Filtering and completing files")
	for i, file := range files {
		fmt.Printf("\rDone: %d %%", i*100/sz)

		if to_remove[file] {
			continue
		}

		mft := get_mft(file.MftId)

		entry, found := mft.files[file.Reference]
		if found {
			if len(file.Name) == 0 {
				file.Name = entry.name
			}

			dir := entry.dir

			if file.Parent.IsNull() {
				file.Parent = dir
			}

			if !file.Parent.IsNull() {
				parent, ok := mft.dirs[file.Parent.GetFileIndex()]
				if ok && (parent.Header.SequenceNumber != file.Parent.GetSequenceNumber()) {
					if file.Parent == dir {
						continue
					}

					parent, ok = mft.dirs[dir.GetFileIndex()]
					if ok && (parent.Header.SequenceNumber != dir.GetSequenceNumber()) {
						continue
					}

					file.Parent = dir
				}
			}
		} else {
			if len(file.Name) == 0 {
				continue
			}

			parent, ok := mft.dirs[file.Parent.GetFileIndex()]
			if ok && (parent.Header.SequenceNumber != file.Parent.GetSequenceNumber()) {
				continue
			}
		}

		if len(file.Name) > 0 {
			records = append(records, file)
		}
	}

	fmt.Println("\rDone: 100 %")

	writer, err := inspect.MakeStateWriter(dest)
	if err != nil {
		return err
	}

	defer ntfs.DeferedCall(writer.Close)

	fmt.Println()
	fmt.Println("Writing")
	sz = len(records)
	for i, rec := range records {
		fmt.Printf("\rDone: %d %%", i*100/sz)

		if err := writer.Write(rec); err != nil {
			return err
		}
	}

	fmt.Println("\rDone: 100 %")

	return nil
}
