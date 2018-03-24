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

	fmt.Println("Count=", states.GetCount())

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

	fmt.Println(fmt.Sprintf("Filling (count= %d)", cnt))
	for state := range stream {
		progress := 100 * i / cnt
		fmt.Printf("\rDone: %d %%", progress)
		i++

		ok, err := state.Init(disk)
		if err != nil {
			return err
		}

		if ok {
			if err := writer.Write(state); err != nil {
				return nil
			}
		}
	}

	fmt.Println("\r100 %                                                      ")

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
	for record := range stream {
		fmt.Printf("\r%d %%", i*100/sz)
		i++

		if record.GetType() != inspect.STATE_RECORD_TYPE_FILE {
			continue
		}

		r := record.(*inspect.StateFileRecord)
		for n, attr := range r.Attributes {
			if attr.Header.AttributeType != core.ATTR_FILE_NAME {
				continue
			}

			if attr.Header.NonResident != core.BOOL_FALSE {
				msg := fmt.Sprintf("  - Nonresident found at %d [attribute %d]", r.Position, n)
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

	cnt := 0

	fmt.Println("\rDone.")

	print_result := func(list []string, msg string) {
		fmt.Println()
		sz := len(list)
		if sz > 0 {
			if verbose {
				fmt.Println(msg + ":")
				for _, l := range list {
					fmt.Println(l)
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
		fmt.Println("No Record found")
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
	for record := range stream {
		fmt.Printf("\r%d %%", i*100/sz)
		i++

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
	for record := range stream {
		j := i
		i++

		if record.GetType() != inspect.STATE_RECORD_TYPE_FILE {
			continue
		}

		r := record.(*inspect.StateFileRecord)

		f_type := "File"
		if (r.Header.Flags & core.FFLAG_DIRECTORY) != core.FFLAG_NONE {
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
		for state := range stream {
			if (state.GetType() == inspect.STATE_RECORD_TYPE_MFT) && (state.GetMftId() == mft) {
				core.PrintStruct(state)

				return nil
			}
		}

		return core.WrapError(fmt.Errorf("MFT %s does not exist", mft))
	}

	var list []inspect.IStateRecord

	for state := range stream {
		if state.GetType() == inspect.STATE_RECORD_TYPE_MFT {
			list = append(list, state)
		}
	}

	core.PrintStruct(list)

	return nil
}
