package main

import (
	"fmt"
	"os"

	ntfs "github.com/corebreaker/ntfstool/core"
	"github.com/corebreaker/ntfstool/core/data"
	datafile "github.com/corebreaker/ntfstool/core/data/file"
	"github.com/corebreaker/ntfstool/inspect"
)

func do_fillinfo(arg *tActionArg) error {
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

	disk := arg.disk.GetDisk()
	defer ntfs.DeferedCall(disk.Close)

	stream, err := states.MakeStream()
	if err != nil {
		return err
	}

	defer ntfs.DeferedCall(stream.Close)

	writer, err := inspect.MakeStateWriter(dest)
	if err != nil {
		return err
	}

	defer ntfs.DeferedCall(writer.Close)

	i, cnt := 0, states.GetCount()
	idx_count, idx_good := 0, 0
	resident_datas, external_names := 0, 0
	no_name, no_data := 0, 0

	fmt.Println(fmt.Sprintf("Filling (count= %d)", cnt))
	for item := range stream {
		progress := 100 * i / cnt
		fmt.Printf("\rDone: %d %%", progress)
		i++

		state := item.Record()
		if err := state.GetError(); err != nil {
			return err
		}

		if state.IsNull() {
			continue
		}

		rectyp := state.GetType()

		idx_ok := rectyp == inspect.STATE_RECORD_TYPE_INDEX
		if idx_ok {
			idx_count++
		}

		ok, err := state.Init(disk)
		if err != nil {
			return err
		}

		if ok {
			if rectyp == inspect.STATE_RECORD_TYPE_FILE {
				rec := state.(*inspect.StateFileRecord)

				if rec.IsDir() {
					attrs := rec.GetAttributes(ntfs.ATTR_INDEX_ROOT)
					if len(attrs) == 0 {
						no_data++

						continue
					}
				} else {
					attrs := rec.GetAttributes(ntfs.ATTR_DATA)
					if len(attrs) == 0 {
						no_data++

						continue
					}

					if !attrs[0].Header.NonResident.Value() {
						resident_datas++
					}
				}

				attrs := rec.GetAttributes(ntfs.ATTR_FILE_NAME)
				if len(attrs) == 0 {
					no_name++
				}

				var fname string

				for _, attr := range attrs {
					if attr.Header.NonResident.Value() {
						external_names++

						continue
					}

					desc, err := state.GetAttributeDesc(attr)
					if err != nil {
						return err
					}

					attr_val, err := desc.GetValue(nil)
					if err != nil {
						return err
					}

					name := attr_val.GetFilename()

					if attr_val.IsLongName() {
						fname = name
					} else {
						if fname == "" {
							fname = name
						}
					}
				}

				state.SetName(fname)
			}

			if idx_ok {
				idx_good++
			}

			if err := writer.Write(state); err != nil {
				return err
			}
		}
	}

	fmt.Println("\r100 %                                                      ")
	fmt.Println("Indexes:                       ", idx_good, "/", idx_count)
	fmt.Println("Records with Resident datas:   ", resident_datas)
	fmt.Println("Records with Non-Resident name:", external_names)
	fmt.Println("Records with no data found:    ", no_data)
	fmt.Println("Records with no name found:    ", no_name)

	return nil
}

func do_check(verbose bool, arg *tActionArg) error {
	src, err := arg.GetInput()
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

	reg := make(inspect.FileFrequencies)

	non_resident_names := make([]string, 0)
	duplicates := make([]string, 0)

	i, sz := 0, states.GetCount()

	fmt.Println("Searching")
	for item := range stream {
		fmt.Printf("\r%d %%", i*100/sz)
		i++

		record := item.Record()
		if err := record.GetError(); err != nil {
			return err
		}

		if record.IsNull() {
			continue
		}

		if record.GetType() != inspect.STATE_RECORD_TYPE_FILE {
			continue
		}

		r := record.(*inspect.StateFileRecord)
		for n, attr := range r.Attributes {
			if attr.Header.AttributeType != ntfs.ATTR_FILE_NAME {
				continue
			}

			if attr.Header.NonResident != ntfs.BOOL_FALSE {
				msg := fmt.Sprintf("  - Nonresident found at %d [record %d, attribute %d]", r.Position, item.Index(), n)
				non_resident_names = append(non_resident_names, msg)
			}
		}

		if (r.Names != nil) && (len(r.Names) > 0) && (r.Parent != 0) {
			if reg.Add(r.Parent, r.Name) {
				msg := fmt.Sprintf("  - Dupplicate found at %d [name pair %s/%v]", r.Position, r.Name, r.Parent)
				duplicates = append(duplicates, msg)
			}
		}
	}

	fmt.Println("\rDone.")

	cnt := 0

	print_result := func(list []string, msg string) {
		fmt.Println()
		sz := len(list)
		if sz > 0 {
			if verbose {
				fmt.Fprintln(os.Stderr, msg+":")
				for _, l := range list {
					fmt.Fprintln(os.Stderr, l)
				}
			} else {
				fmt.Println(msg+":", sz)
			}

			cnt++
		}
	}

	print_result(non_resident_names, "Non-resident names")
	print_result(duplicates, "Duplicate paths")

	if cnt == 0 {
		fmt.Println("No problem encountered")
	}

	return nil
}

func do_shownames(arg *tActionArg) error {
	src, err := arg.GetInput()
	if err != nil {
		return err
	}

	fmt.Println("Reading")
	reader, err := datafile.MakeDataReader(src, datafile.ANY_FILEFORMAT)
	if err != nil {
		return err
	}

	defer ntfs.DeferedCall(reader.Close)

	stream, err := reader.MakeStream()
	if err != nil {
		return err
	}

	defer ntfs.DeferedCall(stream.Close)

	fmt.Println("Result")
	for item := range stream {
		rec := item.Record()
		if !rec.HasName() {
			continue
		}

		fmt.Fprintln(
			os.Stderr,
			fmt.Sprintf("   %06d - %s", item.Index(), rec),
		)
	}

	return nil
}

func do_listnames(arg *tActionArg) error {
	src, err := arg.GetInput()
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

	i := 0

	fmt.Println("Result")
	for item := range stream {
		j := i
		i++

		record := item.Record()
		record.GetName()
		if err := record.GetError(); err != nil {
			return err
		}

		if record.IsNull() {
			continue
		}

		if record.GetType() != inspect.STATE_RECORD_TYPE_FILE {
			continue
		}

		r := record.(*inspect.StateFileRecord)

		f_type := "File"
		if r.IsDir() {
			f_type = "Dir"
		}

		fmt.Fprintln(
			os.Stderr,
			fmt.Sprintf("   %06d - %d : %s { %s, Parent= %v }", j, r.Position, r.Names, f_type, r.Parent),
		)
	}

	return nil
}

func do_position(position int64, arg *tActionArg) error {
	src, err := arg.GetInput()
	if err != nil {
		return err
	}

	fmt.Println("Reading")
	records, err := datafile.MakeDataReader(src, datafile.ANY_FILEFORMAT)
	if err != nil {
		return err
	}

	defer ntfs.DeferedCall(records.Close)

	record, err := records.ReadRecord(position)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("Format:", records.GetFormatName())

	fmt.Println()
	fmt.Println("Result:")
	record.Print()

	file, ok := record.(*inspect.StateFileRecord)
	if ok {
		fmt.Println()
		fmt.Println("Data:")
		ntfs.PrintBytes(file.Header.Data[:])
	}

	return nil
}

func do_show_attribute(attribute_position int64, arg *tActionArg) error {
	src, err := arg.GetInput()
	if err != nil {
		return err
	}

	fmt.Println("Reading")
	records, err := datafile.MakeDataReader(src, datafile.ANY_FILEFORMAT)
	if err != nil {
		return err
	}

	defer ntfs.DeferedCall(records.Close)

	var state data.IDataRecord

	file_pos, file_pos_ok := arg.IntExt("for-file-at")
	if file_pos_ok {
		state, err = records.ReadRecord(file_pos)
		if err != nil {
			return err
		}
	}

	file_idx, file_idx_ok := arg.IntExt("for-file")
	if file_idx_ok {
		state, err = records.GetRecordAt(int(file_idx))
		if err != nil {
			return err
		}
	}

	file, ok := state.(*inspect.StateFileRecord)
	if !ok {
		return ntfs.WrapError(fmt.Errorf("Bad record type"))
	}

	attr := file.GetAttribute(attribute_position)
	if attr == nil {
		return ntfs.WrapError(fmt.Errorf("Attribute not found"))
	}

	desc, err := file.GetAttributeDesc(attr)
	if err != nil {
		return err
	}

	val, err := desc.GetValue(nil)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("File:")
	ntfs.PrintStruct(desc)

	fmt.Println()
	fmt.Println("Value:")
	ntfs.PrintStruct(val)

	return nil
}

func do_show(index int64, arg *tActionArg) error {
	src, err := arg.GetInput()
	if err != nil {
		return err
	}

	fmt.Println("Reading")
	records, err := datafile.MakeDataReader(src, datafile.ANY_FILEFORMAT)
	if err != nil {
		return err
	}

	defer ntfs.DeferedCall(records.Close)

	offsets := records.Offsets()

	fmt.Println()
	fmt.Println("Format:", records.GetFormatName())
	fmt.Println("Offset:", offsets[index])

	if (0 > index) || (index >= int64(records.GetCount())) {
		return nil
	}

	record, err := records.GetRecordAt(int(index))
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("Result:")
	record.Print()

	return nil
}

func do_head(index int64, arg *tActionArg) error {
	src, err := arg.GetInput()
	if err != nil {
		return err
	}

	fmt.Println("Reading")
	records, err := datafile.MakeDataReader(src, datafile.ANY_FILEFORMAT)
	if err != nil {
		return err
	}

	defer ntfs.DeferedCall(records.Close)

	str, err := records.MakeStream()
	if err != nil {
		return err
	}

	defer ntfs.DeferedCall(str.Close)

	if index <= 0 {
		index = 10
	}

	fmt.Println()
	fmt.Println("Format:", records.GetFormatName())

	fmt.Println()
	fmt.Println("List:")
	for i := int64(0); i < index; i++ {
		item := <-str

		fmt.Println(">>", item.Index(), "(", item.Offset(), ")")
		item.Record().Print()
		fmt.Println("------------------------------------------------------------------------------")
		fmt.Println()
	}

	return nil
}

func do_tail(index int64, arg *tActionArg) error {
	src, err := arg.GetInput()
	if err != nil {
		return err
	}

	fmt.Println("Reading")
	records, err := datafile.MakeDataReader(src, datafile.ANY_FILEFORMAT)
	if err != nil {
		return err
	}

	defer ntfs.DeferedCall(records.Close)

	if index <= 0 {
		index = 10
	}

	str, err := records.MakeStreamFrom(int64(records.GetCount()) - index)
	if err != nil {
		return err
	}

	defer ntfs.DeferedCall(str.Close)

	fmt.Println()
	fmt.Println("Format:", records.GetFormatName())

	fmt.Println()
	fmt.Println("List from", index, ":")
	for item := range str {
		fmt.Println(">>", item.Index(), "(", item.Offset(), ")")
		item.Record().Print()
		fmt.Println("------------------------------------------------------------------------------")
		fmt.Println()
	}

	return nil
}

func do_offsets(index int64, arg *tActionArg) error {
	src, err := arg.GetInput()
	if err != nil {
		return err
	}

	fmt.Println("Reading")
	records, err := datafile.MakeDataReader(src, datafile.ANY_FILEFORMAT)
	if err != nil {
		return err
	}

	defer ntfs.DeferedCall(records.Close)

	if index <= 0 {
		index = 10
	}

	offsets := records.Offsets()

	lim := int64(len(offsets) - 1)
	if index > lim {
		index = lim
	}

	fmt.Println()
	fmt.Println("Format:", records.GetFormatName())

	fmt.Println()
	fmt.Println("List:")
	for i := int64(0); i < index; i++ {
		fmt.Println("  -", i+1, ":", offsets[i])
	}

	return nil
}

func do_show_mft(mft string, arg *tActionArg) error {
	src, err := arg.GetInput()
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

	fmt.Println()
	fmt.Println("Result:")
	if mft != "" {
		for item := range stream {
			state := item.Record()

			if err := state.GetError(); err != nil {
				return err
			}

			if state.IsNull() {
				continue
			}

			if (state.GetType() == inspect.STATE_RECORD_TYPE_MFT) && (state.GetMftId() == mft) {
				ntfs.PrintStruct(state)

				return nil
			}
		}

		return ntfs.WrapError(fmt.Errorf("MFT %s does not exist", mft))
	}

	var list []inspect.IStateRecord

	for item := range stream {
		state := item.Record()

		if err := state.GetError(); err != nil {
			return err
		}

		if state.IsNull() {
			continue
		}

		if state.GetType() == inspect.STATE_RECORD_TYPE_MFT {
			list = append(list, state)
		}
	}

	ntfs.PrintStruct(list)

	return nil
}
