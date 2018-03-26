package main

import (
	"essai/ntfstool/core"
	"essai/ntfstool/inspect"
	"fmt"
)

func do_fixmft(arg *tActionArg) error {
	src, dest, err := arg.GetFiles()
	if err != nil {
		return err
	}

	fmt.Println("Reading")
	states, err := inspect.MakeStateReader(src)
	if err != nil {
		return err
	}

	defer core.DeferedCall(states.Close)

	stream, err := states.MakeStream()
	if err != nil {
		return err
	}

	tables := make(map[int64]*inspect.StateMft)
	mfts := make(map[string]*inspect.StateMft)
	dirs := make(map[int64]*inspect.StateFileRecord)
	files := make(map[string]map[int64]*inspect.StateFileRecord)

	i, cnt := 0, states.GetCount()
	res := make([]inspect.IStateRecord, 0)

	fmt.Println("Finding MFTs")
	for item := range stream {
		fmt.Printf("\rDone: %d %%", 100*i/cnt)
		i++

		state := item.Record()

		if err := state.GetError(); err != nil {
			return err
		}

		if state.IsNull() {
			continue
		}

		switch state.GetType() {
		case inspect.STATE_RECORD_TYPE_FILE:
			file_state := state.(*inspect.StateFileRecord)
			file_pos := file_state.Position

			attr_list := file_state.GetAttributes(core.ATTR_DATA)
			if len(attr_list) == 0 {
				continue
			}

			data_attr := attr_list[0]

			if file_state.Name == "$MFT" {
				current := &inspect.StateMft{
					StateBase: inspect.StateBase{
						Position: file_pos,
					},
					Header:     file_state.Header,
					RunList:    data_attr.RunList,
					PartOrigin: file_pos - int64(data_attr.RunList[0].Start*0x1000),
				}

				ok, err := arg.disk.InitState(current)
				if err != nil {
					return err
				}

				if !ok {
					continue
				}

				ok, err = current.IsMft(arg.disk)
				if err != nil {
					return err
				}

				if ok {
					res = append(res, current)

					mftid := current.GetMftId()
					files[mftid] = make(map[int64]*inspect.StateFileRecord)

					cnt := 0

					for _, run := range data_attr.RunList {
						if run.Zero {
							continue
						}

						pos_beg := current.PartOrigin + int64(run.Start*0x1000)
						pos_end := pos_beg + (run.Count * 0x1000)

						for position := pos_beg; position < pos_end; position += 1024 {
							prev, exists := tables[position]
							if exists {
								msg := "\rWarning: MFT %s use position %d that has already used by MFT %s."
								fmt.Println(msg, current, position, prev)
							}

							tables[position] = current
							cnt++
						}
					}

					if cnt > 0 {
						mfts[mftid] = current
					}
				}

				res = append(res, state)
				continue
			}

			mft, found := tables[file_pos]
			if !found {
				continue
			}

			file_state.SetMft(mft)

			ref := mft.GetReference(file_pos)
			if ref == 0 {
				fmt.Println("\rWarning: File at position %d is not in MFT %s", file_pos, mft)

				continue
			}

			file_state.Reference = ref
			files[file_state.GetMftId()][ref.GetFileIndex()] = file_state

			if (file_state.Header.Flags & core.FFLAG_DIRECTORY) != core.FFLAG_NONE {
				attr_list := file_state.GetAttributes(core.ATTR_INDEX_ALLOCATION)
				if len(attr_list) == 0 {
					break
				}

				for _, dir_attr := range attr_list {
					for _, run := range dir_attr.RunList {
						if run.Zero {
							continue
						}

						pos_beg := mft.PartOrigin + int64(run.Start*0x1000)
						pos_end := pos_beg + (run.Count * 0x1000)

						for position := pos_beg; position < pos_end; position += 0x1000 {
							prev, exists := dirs[position]
							if exists {
								const msg = "\rWarning: File %s use position %d that has already used by file %s."

								fmt.Println(msg, file_state, position, prev)
							}

							dirs[position] = file_state
						}
					}
				}
			}
		}

		res = append(res, state)
	}

	fmt.Println("\r100%                                                      ")

	writer, err := inspect.MakeStateWriter(dest)
	if err != nil {
		return err
	}

	defer core.DeferedCall(writer.Close)

	cnt = len(res)

	fmt.Println("Finding Non-resident Directory Indexes")
	for i, state := range res {
		fmt.Printf("\rDone: %d %%", 100*i/cnt)

		func() {
			switch state.GetType() {
			case inspect.STATE_RECORD_TYPE_INDEX:
				index_state := state.(*inspect.StateIndexRecord)

				file_state, ok := dirs[index_state.Position]
				if !ok {
					return
				}

				good_name := func(entry *inspect.StateDirEntry) bool {
					name := entry.Name
					for _, n := range file_state.Names {
						if name == n {
							return true
						}
					}

					return len(file_state.Names) == 0
				}

				for _, entry := range index_state.Entries {
					ref := file_state.Reference

					if ((ref != 0) && (entry.Parent != ref)) || (!good_name(entry)) {
						return
					}
				}

				index_state.Reference = file_state.Reference
				index_state.Parent = file_state.Parent

				mft := mfts[file_state.MftId]
				index_state.SetMft(mft)
			}
		}()

		if err := writer.Write(state); err != nil {
			return err
		}
	}

	fmt.Println("\r100%                                                      ")

	return nil
}
