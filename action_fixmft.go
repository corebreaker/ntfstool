package main

import (
	"bytes"
	"fmt"
	"os"

	ntfs "github.com/corebreaker/ntfstool/core"
	"github.com/corebreaker/ntfstool/inspect"
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

	defer ntfs.DeferedCall(states.Close)

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

	var records []inspect.IStateRecord
	var log bytes.Buffer
	var currentMft *inspect.StateMft

	mft_collision := 0
	pos_collision := 0
	orpheans := 0

	fix := func(state *inspect.StateFileRecord, mft *inspect.StateMft) error {
		state.SetMft(mft)

		ref := mft.GetReference(state)
		if ref.IsNull() {
			ref = state.Header.FileRef()

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
			is_dir := file_state.IsDir()

			attr_list := file_state.GetAttributes(ntfs.ATTR_FILE_NAME, ntfs.ATTR_DATA, ntfs.ATTR_INDEX_ROOT)
			switch len(attr_list) {
			case 0, 1:
				continue

			case 2:
				if is_dir {
					if attr_list[1].Header.AttributeType != ntfs.ATTR_INDEX_ROOT {
						continue
					}
				} else {
					if attr_list[1].Header.AttributeType != ntfs.ATTR_DATA {
						continue
					}
				}
			}

			fname_attr, data_attr := attr_list[0], attr_list[1]
			mft_pos := file_pos

			get_attr := func(rec *ntfs.FileRecord, atype ntfs.AttributeType) (*ntfs.AttributeDesc, error) {
				attrs, err := rec.GetAttributes(false)
				if err != nil {
					return nil, err
				}

				attr_lst := rec.GetAttributeFilteredList(atype)
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

					var record ntfs.FileRecord

					err := func() error {
						disk := arg.disk.GetDisk()
						defer disk.Close()

						disk.SetOffset(mft_pos + 4096)

						return disk.ReadStruct(0, &record)
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

					attr, err := get_attr(&record, ntfs.ATTR_FILE_NAME)
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
				var record0, record1 ntfs.FileRecord

				err := func() error {
					disk := arg.disk.GetDisk()
					defer disk.Close()

					disk.SetOffset(mft_pos)
					if err := disk.ReadStruct(0, &record0); err != nil {
						return err
					}

					disk.SetOffset(mft_pos + 1024)
					if err := disk.ReadStruct(0, &record1); err != nil {
						return err
					}

					return nil
				}()

				if err != nil {
					return err
				}

				ok, err := func() (bool, error) {
					attr0, err := get_attr(&record0, ntfs.ATTR_DATA)
					if err != nil {
						return false, err
					}

					attr1, err := get_attr(&record1, ntfs.ATTR_DATA)
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

					var mft_record ntfs.FileRecord

					state, err := func() (*inspect.StateFileRecord, error) {
						disk := arg.disk.GetDisk()
						defer disk.Close()

						disk.SetOffset(mft_pos)
						if err := disk.ReadStruct(0, &mft_record); err != nil {
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

					attrs := state.GetAttributes(ntfs.ATTR_DATA)
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

								if !pending.state.Reference.IsNull() {
									records = append(records, pending.state)
								}

								delete(pendings, position)
							}
						}
					}

					currentMft = mft

					records = append(records, nil)
					copy(records[1:], records)
					records[0] = mft

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

			if file_state.Reference.IsNull() {
				continue
			}
		}

		records = append(records, state)
	}

	fmt.Println("\r100%                                                      ")

	fmt.Println("Fixing remaining (", len(pendings), ")")
	i, cnt = 0, len(pendings)

	for _, p := range pendings {
		fmt.Printf("\rDone: %d %%", 100*i/cnt)

		if err := fix(p.state, p.mft); err != nil {
			return err
		}

		if p.state.Reference.IsNull() {
			continue
		}

		records = append(records, p.state)
	}

	fmt.Println("\r100%                                                      ")

	writer, err := inspect.MakeStateWriter(dest)
	if err != nil {
		return err
	}

	defer ntfs.DeferedCall(writer.Close)

	cnt = len(records)
	no_mft := 0

	fmt.Println("Writing")
	for i, state := range records {
		fmt.Printf("\rDone: %d %%", 100*i/cnt)

		if (state.GetHeader().Type == ntfs.RECTYP_FILE) && (len(state.GetMftId()) == 0) {
			no_mft++

			fmt.Fprintf(&log, "No MFT for position %d", state.GetPosition())
			fmt.Fprintln(&log)
		}

		if err := writer.Write(state); err != nil {
			return err
		}
	}

	fmt.Println("\r100%                                                      ")

	fmt.Println()
	fmt.Println("Summary:")
	fmt.Println("  - Records:            ", len(records))
	fmt.Println("  - Orpheans:           ", orpheans)
	fmt.Println("  - MFT collisions:     ", mft_collision)
	fmt.Println("  - MFT missings:       ", no_mft)
	fmt.Println("  - Position collisions:", pos_collision)

	if verbose {
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Details:")
		fmt.Fprintln(os.Stderr, &log)
	}

	return nil
}
