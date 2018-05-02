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

		dbg := state.GetPosition() == 3222191104
		pass := func(dbgi int) {
			if !dbg {
				return
			}

			fmt.Printf("DBG-PASS %03d", dbgi)
			fmt.Println()
		}

		pass(1) //

		if state.IsNull() {
			pass(2) //
			continue
		}

		switch state.GetType() {
		case inspect.STATE_RECORD_TYPE_FILE:
			pass(3) //
			file_state := state.(*inspect.StateFileRecord)
			file_pos := file_state.Position
			is_dir := file_state.IsDir()

			attr_list := file_state.GetAttributes(ntfs.ATTR_FILE_NAME, ntfs.ATTR_DATA, ntfs.ATTR_INDEX_ROOT)
			switch len(attr_list) {
			case 0, 1:
				pass(4) //
				continue

			case 2:
				if is_dir {
					if attr_list[1].Header.AttributeType != ntfs.ATTR_INDEX_ROOT {
						pass(400) //
						continue
					}
				} else {
					if attr_list[1].Header.AttributeType != ntfs.ATTR_DATA {
						pass(401) //
						continue
					}
				}
			}

			pass(5) //
			fname_attr, data_attr := attr_list[0], attr_list[1]
			mft_pos := file_pos

			get_attr := func(rec *ntfs.FileRecord, atype ntfs.AttributeType) (*ntfs.AttributeDesc, error) {
				pass(6) //
				attrs, err := rec.GetAttributes(false)
				if err != nil {
					return nil, err
				}

				attr_lst := rec.GetAttributeFilteredList(atype)
				if len(attr_lst) == 0 {
					pass(7) //
					return nil, nil
				}

				attr_hdr, ok := attrs[attr_lst[0]]
				if !ok {
					pass(8) //
					return nil, nil
				}

				pass(9) //
				return rec.MakeAttributeFromHeader(attr_hdr)
			}

			pass(10) //
			is_mft, is_mirror, err := func() (bool, bool, error) {
				pass(11) //
				if fname_attr.Header.NonResident.Value() {
					pass(12) //
					return false, false, nil
				}

				pass(13) //
				name, err := file_state.Header.GetFilename(nil)
				if err != nil {
					pass(14) //
					return false, false, err
				}

				mft_rec_num := file_state.Header.MftRecordNumber

				pass(15) //
				switch name {
				case "$Volume":
					pass(16) //
					if mft_rec_num != 3 {
						pass(17) //
						return false, false, nil
					}

					pass(18) //
					mft_rec_num--
					mft_pos -= 1024
					fallthrough

				case "$LogFile":
					pass(19) //
					if mft_rec_num != 2 {
						pass(20) //
						return false, false, nil
					}

					mft_rec_num--
					mft_pos -= 1024
					fallthrough

				case "$MFTMirr":
					pass(21) //
					if mft_rec_num != 1 {
						pass(22) //
						return false, false, nil
					}

					mft_rec_num--
					mft_pos -= 1024
					fallthrough

				case "$MFT":
					pass(23) //
					if mft_rec_num != 0 {
						pass(24) //
						return false, false, nil
					}

					var record ntfs.FileRecord

					pass(25) //
					err := func() error {
						pass(26) //
						disk := arg.disk.GetDisk()
						defer disk.Close()

						disk.SetOffset(mft_pos + 4096)

						return disk.ReadStruct(0, &record)
					}()

					pass(27) //
					if err != nil {
						return false, false, err
					}

					if !record.Type.IsGood() {
						pass(28) //
						return false, true, nil
					}

					pass(29) //
					if record.MftRecordNumber != 4 {
						pass(30) //
						return false, true, nil
					}

					pass(31) //
					attr, err := get_attr(&record, ntfs.ATTR_FILE_NAME)
					if err != nil {
						pass(32) //
						return false, false, err
					}

					pass(33) //
					if attr == nil {
						pass(34) //
						return false, true, nil
					}

					pass(35) //
					if attr.Header.NonResident.Value() {
						pass(36) //
						return false, true, nil
					}

					pass(37) //
					n, err := record.GetFilename(nil)
					if err != nil {
						pass(38) //
						return false, false, err
					}

					res := n == "$AttrDef"
					pass(39) //

					return res, !res, nil
				}
				pass(40) //

				return false, false, nil
			}()

			pass(41) //
			if err != nil {
				return err
			}

			pass(42) //
			mft_state := file_state
			if is_mirror {
				var record0, record1 ntfs.FileRecord

				pass(43) //
				err := func() error {
					pass(44) //
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

				pass(45) //
				if err != nil {
					return err
				}

				pass(46) //
				ok, err := func() (bool, error) {
					pass(47) //
					attr0, err := get_attr(&record0, ntfs.ATTR_DATA)
					if err != nil {
						return false, err
					}

					pass(48) //
					attr1, err := get_attr(&record1, ntfs.ATTR_DATA)
					if err != nil {
						return false, err
					}

					pass(49) //
					rl0, rl1 := attr0.GetRunList(), attr1.GetRunList()

					pos0, pos1 := int64(rl0[0].Start*0x1000), int64(rl1[0].Start*0x1000)
					origin := mft_pos - pos1
					mft_pos += pos0 - pos1

					existing_mft, ok := tables[mft_pos]
					if ok {
						pass(50) //
						if existing_mft.PartOrigin != origin {
							pass(51) //
							return false, nil
						}

						pass(52) //
						currentMft = existing_mft

						return true, nil
					}

					var mft_record ntfs.FileRecord

					pass(53) //
					state, err := func() (*inspect.StateFileRecord, error) {
						pass(54) //
						disk := arg.disk.GetDisk()
						defer disk.Close()

						disk.SetOffset(mft_pos)
						if err := disk.ReadStruct(0, &mft_record); err != nil {
							return nil, err
						}

						pass(55) //
						res := &inspect.StateFileRecord{
							StateBase: inspect.StateBase{Position: mft_pos},
							Header:    mft_record,
						}

						ok, err := res.Init(disk)
						if err != nil {
							return nil, err
						}

						pass(56) //
						if !ok {
							pass(57) //
							res = nil
						}

						return res, nil
					}()

					pass(58) //
					if err != nil {
						return false, err
					}

					pass(59) //
					if state == nil {
						pass(60) //
						return false, nil
					}

					pass(61) //
					attrs := state.GetAttributes(ntfs.ATTR_DATA)
					if len(attrs) == 0 {
						pass(62) //
						return false, nil
					}

					is_mft = true
					mft_state, data_attr = state, attrs[0]
					pass(63) //

					return true, nil
				}()

				pass(64) //
				if err != nil {
					return err
				}

				pass(65) //
				if !ok {
					pass(66) //
					continue
				}
				pass(67) //
			}

			pass(68) //
			if is_mft {
				pass(69) //
				ok, err := func() (bool, error) {
					pass(70) //
					mft, found := tables[mft_pos]
					if found {
						pass(71) //
						currentMft = mft

						return false, nil
					}

					pass(72) //
					mft = &inspect.StateMft{
						StateBase: inspect.StateBase{
							Position: mft_pos,
						},
						Header:     mft_state.Header,
						RunList:    data_attr.RunList,
						PartOrigin: mft_pos - int64(data_attr.RunList[0].Start*0x1000),
					}

					pass(73) //
					ok, err := arg.disk.InitState(mft)
					if err != nil {
						return false, err
					}

					pass(74) //
					if !(ok && mft.IsMft()) {
						pass(75) //
						return false, nil
					}

					pass(76) //
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

					pass(77) //
					currentMft = mft

					records = append(records, nil)
					copy(records[1:], records)
					records[0] = mft

					return true, nil
				}()

				pass(78) //
				if err != nil {
					return err
				}

				pass(79) //
				if !ok {
					pass(80) //
					continue
				}
				pass(81) //
			}

			pass(82) //
			mft, found := tables[file_pos]
			if !found {
				pass(83) //
				pendings[file_pos] = tPending{
					state: file_state,
					mft:   currentMft,
				}

				continue
			}

			pass(84) //
			if err := fix(file_state, mft); err != nil {
				return err
			}

			pass(85) //
			if file_state.Reference.IsNull() {
				pass(86) //
				continue
			}
		}

		pass(87) //
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
