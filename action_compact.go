package main

import (
	"fmt"

	ntfs "github.com/corebreaker/ntfstool/core"
	"github.com/corebreaker/ntfstool/core/data"
	datafile "github.com/corebreaker/ntfstool/core/data/file"
)

func do_compact(arg *tActionArg) error {
	file, err := arg.GetFile()
	if err != nil {
		return err
	}

	fname := file.Name()

	var records []data.IDataRecord
	var format string

	err = func() error {
		in, err := datafile.MakeDataReader(file, "")
		if err != nil {
			return err
		}

		defer ntfs.DeferedCall(in.Close)

		cnt := in.GetCount()
		format = in.GetFormatName()

		stream, err := in.MakeStream()
		if err != nil {
			return err
		}

		defer ntfs.DeferedCall(stream.Close)

		i := 0

		fmt.Println("Reading")
		for item := range stream {
			fmt.Printf("\rDone: %d %%", i*100/cnt)
			i++

			rec := item.Record()
			if err := rec.GetError(); err != nil {
				return err
			}

			records = append(records, rec)
		}

		fmt.Println("\rDone.      ")

		return nil
	}()

	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("Writing")

	cnt := len(records)

	out, err := datafile.OpenDataWriter(fname, format)
	if err != nil {
		return err
	}

	defer ntfs.DeferedCall(out.Close)

	for i, r := range records {
		fmt.Printf("\rDone: %d %%", i*100/cnt)

		if err := out.Write(r); err != nil {
			return err
		}
	}

	fmt.Println("\rDone.      ")

	return nil
}
