package main

import (
	"fmt"
	"os"

	"essai/ntfstool/core"
	"essai/ntfstool/core/dataio/datafile"
	"essai/ntfstool/inspect"
)

func do_state_count(arg *tActionArg) error {
	src, err := arg.GetInput()
	if err != nil {
		return err
	}

	fmt.Println("Reading")
	states, err := inspect.MakeStateReader(src)
	if err != nil {
		return err
	}

	defer core.DeferedCall(states.Close)

	for rec_type, count := range states.GetCounts() {
		fmt.Println(rec_type.GetLabel(), "=", count)
	}

	fmt.Println()
	fmt.Println("Total Record Count =", states.GetCount())

	return nil
}

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

	defer core.DeferedCall(states.Close)

	disk := arg.disk.GetDisk()
	defer core.DeferedCall(disk.Close)

	stream, err := states.MakeStream()
	if err != nil {
		return err
	}

	defer core.DeferedCall(stream.Close)

	writer, err := inspect.MakeStateWriter(dest)
	if err != nil {
		return err
	}

	defer core.DeferedCall(writer.Close)

	i, cnt := 0, states.GetCount()
	idx_count := 0
	idx_good := 0

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

		_, idx_ok := state.(*inspect.StateIndexRecord)
		if idx_ok {
			idx_count++
		}

		ok, err := state.Init(disk)
		if err != nil {
			return err
		}

		if ok {
			if idx_ok {
				idx_good++
			}
			if err := writer.Write(state); err != nil {
				return nil
			}
		}
	}

	fmt.Println("\r100 %                                                      ")
	fmt.Println("Indexes:", idx_good, "/", idx_count)

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

	defer core.DeferedCall(states.Close)

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
			if attr.Header.AttributeType != core.ATTR_FILE_NAME {
				continue
			}

			if attr.Header.NonResident != core.BOOL_FALSE {
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

	defer core.DeferedCall(states.Close)

	stream, err := states.MakeStream()
	if err != nil {
		return err
	}

	defer core.DeferedCall(stream.Close)

	writer, err := inspect.MakeStateWriter(dest)
	if err != nil {
		return err
	}

	defer core.DeferedCall(writer.Close)

	i, sz := 0, states.GetCount()

	fmt.Println("Completing")
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

		err := func() error {
			if record.GetType() != inspect.STATE_RECORD_TYPE_FILE {
				return nil
			}

				parent := core.FileReferenceNumber(0)
				fname := ""

				r := record.(*inspect.StateFileRecord)
				for _, attr := range r.Attributes {
					if (attr.Header.NonResident == core.BOOL_FALSE) && (attr.Header.AttributeType == core.ATTR_FILE_NAME) {
						desc, err := r.Header.MakeAttributeFromHeader(&attr.Header)
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
							parent = attr_val.GetParent()
						} else {
							if fname == "" {
								fname = name
							}

							if parent == 0 {
								parent = attr_val.GetParent()
							}
						}

						r.Names = append(r.Names, name)
					}
			}

			r.Parent = parent
			r.Name = fname

			return nil
		}()

		if err != nil {
			return err
		}

		if err := writer.Write(record); err != nil {
			return err
		}
	}

	fmt.Println("\r100 %")

	return nil
}

func do_shownames(arg *tActionArg) error {
	src, err := arg.GetInput()
	if err != nil {
		return err
	}

	fmt.Println("Reading")
	reader, err := datafile.MakeDataReader(src, "")
	if err != nil {
		return err
	}

	defer core.DeferedCall(reader.Close)

	stream, err := reader.MakeStream()
	if err != nil {
		return err
	}

	defer core.DeferedCall(stream.Close)

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

	defer core.DeferedCall(states.Close)

	stream, err := states.MakeStream()
	if err != nil {
		return err
	}

	defer core.DeferedCall(stream.Close)

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
	records, err := datafile.MakeDataReader(src, "")
	if err != nil {
		return err
	}

	defer core.DeferedCall(records.Close)

	record, err := records.ReadRecord(position)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("Format:", records.GetFormatName())

	fmt.Println()
	fmt.Println("Result:")
	record.Print()

	return nil
}

func do_show(index int64, arg *tActionArg) error {
	src, err := arg.GetInput()
	if err != nil {
		return err
	}

	fmt.Println("Reading")
	records, err := datafile.MakeDataReader(src, "")
	if err != nil {
		return err
	}

	defer core.DeferedCall(records.Close)

	record, err := records.GetRecordAt(int(index))
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("Format:", records.GetFormatName())

	fmt.Println()
	fmt.Println("Result:")
	record.Print()

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

	defer core.DeferedCall(states.Close)

	stream, err := states.MakeStream()
	if err != nil {
		return err
	}

	defer core.DeferedCall(stream.Close)

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
				core.PrintStruct(state)

				return nil
			}
		}

		return core.WrapError(fmt.Errorf("MFT %s does not exist", mft))
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

	core.PrintStruct(list)

	return nil
}
