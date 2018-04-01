package main

import (
	"bytes"
	"fmt"

	"essai/ntfstool/core"
	"essai/ntfstool/inspect"
)

func do_fixmft(verbose bool, arg *tActionArg) error {
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

	type tPending struct {
		state *inspect.StateFileRecord
		mft   *inspect.StateMft
	}

	tables := make(map[int64]*inspect.StateMft)
	pendings := make(map[int64]tPending)

	i, cnt := 0, states.GetCount()
	res := make([]inspect.IStateRecord, 0)

	var log bytes.Buffer
	var currentMft *inspect.StateMft

	mft_collision := 0
	pos_collision := 0
	orpheans := 0

	fix := func(state *inspect.StateFileRecord, mft *inspect.StateMft) error {
		state.SetMft(mft)

		ref := mft.GetReference(state)
		if ref.IsNull() {
			ref = state.Header.FileReferenceNumber()

			fmt.Fprintf(&log, "  - Warning: File at position %d (ref=%s) is not in MFT %s", state.Position, ref, mft)
			fmt.Fprintln(&log)

			orpheans++
		}

		state.Reference = ref

		return nil
	}

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

			attr_list := file_state.GetAttributes(core.ATTR_DATA, core.ATTR_FILE_NAME)
			if len(attr_list) < 2 {
				continue
			}

			data_attr, fname_attr := attr_list[0], attr_list[1]
			mft_pos := file_pos

			get_attr := func(rec *core.FileRecord, atype core.AttributeType) (*core.AttributeDesc, error) {
				attrs, err := rec.GetAttributes(false)
				if err != nil {
					return nil, err
				}

				attr_lst := rec.GetAttributeFilteredList(core.ATTR_DATA)
				if len(attr_lst) == 0 {
					return nil, nil
				}

				attr_hdr, ok := attrs[attr_lst[0]]
				if !ok {
					return nil, nil
				}

				return rec.MakeAttributeFromHeader(attr_hdr)
			}

			is_mft, is_mirror, err := func() (bool, bool, error) {
				if fname_attr.Header.NonResident.Value() {
					return false, false, nil
				}

				name, err := file_state.Header.GetFilename(nil)
				if err != nil {
					return false, false, err
				}

				mft_rec_num := file_state.Header.MftRecordNumber

				switch name {
				case "$Volume":
					if mft_rec_num != 3 {
						return false, false, nil
					}

					mft_rec_num--
					mft_pos -= 1024
					fallthrough

				case "$LogFile":
					if mft_rec_num != 2 {
						return false, false, nil
					}

					mft_rec_num--
					mft_pos -= 1024
					fallthrough

				case "$MFTMirr":
					if mft_rec_num != 1 {
						return false, false, nil
					}

					mft_rec_num--
					mft_pos -= 1024
					fallthrough

				case "$MFT":
					if mft_rec_num != 0 {
						return false, false, nil
					}

					var record core.FileRecord

					err := func() error {
						disk := arg.disk.GetDisk()
						defer disk.Close()

						return disk.ReadStruct(mft_pos+4096, &record)
					}()

					if err != nil {
						return false, false, err
					}

					if !record.Type.IsGood() {
						return false, true, nil
					}

					if record.MftRecordNumber != 4 {
						return false, true, nil
					}

					attr, err := get_attr(&record, core.ATTR_FILE_NAME)
					if err != nil {
						return false, false, err
					}

					if attr == nil {
						return false, true, nil
					}

					if attr.Header.NonResident.Value() {
						return false, true, nil
					}

					n, err := record.GetFilename(nil)
					if err != nil {
						return false, false, err
					}

					res := n == "$AttrDef"

					return res, !res, nil
				}

				return false, false, nil
			}()

			if err != nil {
				return err
			}

			mft_state := file_state
			if is_mirror {
				var record0, record1 core.FileRecord

				err := func() error {
					disk := arg.disk.GetDisk()
					defer disk.Close()

					if err := disk.ReadStruct(mft_pos, &record0); err != nil {
						return err
					}

					if err := disk.ReadStruct(mft_pos+1024, &record1); err != nil {
						return err
					}

					return nil
				}()

				if err != nil {
					return err
				}

				ok, err := func() (bool, error) {
					attr0, err := get_attr(&record0, core.ATTR_DATA)
					if err != nil {
						return false, err
					}

					attr1, err := get_attr(&record1, core.ATTR_DATA)
					if err != nil {
						return false, err
					}

					rl0, rl1 := attr0.GetRunList(), attr1.GetRunList()

					pos0, pos1 := int64(rl0[0].Start*0x1000), int64(rl1[0].Start*0x1000)
					origin := mft_pos - pos1
					mft_pos += pos0 - pos1
					existing_mft, ok := tables[mft_pos]
					if ok {
						if existing_mft.PartOrigin != origin {
							return false, nil
						}

						currentMft = existing_mft

						return true, nil
					}

					var mft_record core.FileRecord

					state, err := func() (*inspect.StateFileRecord, error) {
						disk := arg.disk.GetDisk()
						defer disk.Close()

						if err := disk.ReadStruct(mft_pos, &mft_record); err != nil {
							return nil, err
						}

						res := &inspect.StateFileRecord{
							StateBase: inspect.StateBase{Position: mft_pos},
							Header:    mft_record,
						}

						ok, err := res.Init(disk)
						if err != nil {
							return nil, err
						}

						if !ok {
							res = nil
						}

						return res, nil
					}()

					if err != nil {
						return false, err
					}

					if state == nil {
						return false, nil
					}

					attrs := state.GetAttributes(core.ATTR_DATA)
					if len(attrs) == 0 {
						return false, nil
					}

					is_mft = true
					mft_state, data_attr = state, attrs[0]

					return true, nil
				}()

				if err != nil {
					return err
				}

				if !ok {
					continue
				}
			}

			if is_mft {
				ok, err := func() (bool, error) {
					mft, found := tables[mft_pos]
					if found {
						currentMft = mft

						return false, nil
					}

					mft = &inspect.StateMft{
						StateBase: inspect.StateBase{
							Position: mft_pos,
						},
						Header:     mft_state.Header,
						RunList:    data_attr.RunList,
						PartOrigin: mft_pos - int64(data_attr.RunList[0].Start*0x1000),
					}

					ok, err := arg.disk.InitState(mft)
					if err != nil {
						return false, err
					}

					if !(ok && mft.IsMft()) {
						return false, nil
					}

					for _, run := range data_attr.RunList {
						if run.Zero {
							continue
						}

						pos_beg := mft.PartOrigin + int64(run.Start*0x1000)
						pos_end := pos_beg + (run.Count * 0x1000)

						for position := pos_beg; position < pos_end; position += 1024 {
							prev, exists := tables[position]
							if exists {
								const msg = "  - Warning: MFT %s use position %d that has already used by MFT %s."

								fmt.Fprintf(&log, msg, mft, position, prev)
								fmt.Fprintln(&log)

								mft_collision++
							}

							tables[position] = mft

							pending, found := pendings[position]
							if found {
								err := fix(pending.state, mft)
								if err != nil {
									return false, err
								}

								delete(pendings, position)
							}
						}
					}

					currentMft = mft

					res = append(res, nil)
					copy(res[1:], res)
					res[0] = mft

					return true, nil
				}()

				if err != nil {
					return err
				}

				if !ok {
					continue
				}
			}

			mft, found := tables[file_pos]
			if !found {
				pendings[file_pos] = tPending{
					state: file_state,
					mft:   currentMft,
				}

				continue
			}

			if err := fix(file_state, mft); err != nil {
				return err
			}
		}

		res = append(res, state)
	}

	fmt.Println("\r100%                                                      ")

	fmt.Println("Fixing remaining (", len(pendings), ")")
	i, cnt = 0, len(pendings)

	for _, p := range pendings {
		fmt.Printf("\rDone: %d %%", 100*i/cnt)

		if err := fix(p.state, p.mft); err != nil {
			return err
		}
	}

	fmt.Println("\r100%                                                      ")

	writer, err := inspect.MakeStateWriter(dest)
	if err != nil {
		return err
	}

	defer core.DeferedCall(writer.Close)

	cnt = len(res)

	fmt.Println("Writing")
	for i, state := range res {
		fmt.Printf("\rDone: %d %%", 100*i/cnt)

		if err := writer.Write(state); err != nil {
			return err
		}
	}

	fmt.Println("\r100%                                                      ")

	fmt.Println()
	fmt.Println("Summary:")
	fmt.Println("  - Orpheans:           ", orpheans)
	fmt.Println("  - MFT collisions:     ", mft_collision)
	fmt.Println("  - Position collisions:", pos_collision)

	if verbose {
		fmt.Println()
		fmt.Println("Details:")
		fmt.Println(&log)
	}

	return nil
}
