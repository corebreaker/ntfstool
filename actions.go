package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/siddontang/go/ioutil2"

	ntfs "essai/ntfstool/core"
	"essai/ntfstool/inspect"
)

var (
	actions = []iActionDef{
		tDefaultActionDef{handler: do_help, name: "help"},
		tStringActionDef{handler: do_count, name: "count"},
		tStringActionDef{handler: do_set_source, name: "in", next: true},
		tStringActionDef{handler: do_set_destination, name: "out", next: true},
		tDefaultActionDef{handler: do_scan, name: "scan"},
		tDefaultActionDef{handler: do_file_count, name: "file-count"},
		tDefaultActionDef{handler: do_state_count, name: "state-count"},
		tIntegerActionDef{handler: do_show, name: "show"},
		tIntegerActionDef{handler: do_position, name: "at"},
		tStringActionDef{handler: do_show_mft, name: "show-mft"},
		tBoolActionDef{handler: do_check, name: "check"},
		tDefaultActionDef{handler: do_listnames, name: "list-names"},
		tDefaultActionDef{handler: do_complete, name: "complete"},
		tConfigActionDef{handler: do_open_disk},
		tStringActionDef{handler: do_list_files, name: "ls"},
		tIntegerActionDef{handler: do_copy_file, name: "cp"},
		tDefaultActionDef{handler: do_fillinfo, name: "fill"},
		tBoolActionDef{handler: do_fixmft, name: "fix-mft"},
		tBoolActionDef{handler: do_mkfilelist, name: "make-filelist"},
		tIntegerActionDef{handler: do_start, name: "start", next: true, index: true},
		tIntegerActionDef{handler: do_mft, name: "mft", next: true, index: true},
		tDefaultActionDef{handler: do_names, name: "names"},
		tDefaultActionDef{handler: do_find, name: "find"},
		tIntegerActionDef{handler: do_record, name: "record", index: true},
		tIntegerActionDef{handler: do_sector, name: "sector", index: true},
		tIntegerActionDef{handler: do_cluster, name: "cluster", index: true},
		tIntegerActionDef{handler: do_file, name: "file"},
	}
)

func do_open_disk(arg *tActionArg) error {
	disk, err := inspect.OpenNtfsDisk(arg.partition, 0)
	if err != nil {
		return err
	}

	arg.disk = disk

	return nil
}

func do_start(offset int64, arg *tActionArg) error {
	arg.disk.SetStart(offset)

	return nil
}

func do_mft(offset int64, arg *tActionArg) error {
	arg.disk.SetMftShift(offset)

	return nil
}

func do_set_source(source string, arg *tActionArg) error {
	if source == "" {
		return ntfs.WrapError(fmt.Errorf("No source file specified"))
	}

	if !ioutil2.FileExists(source) {
		return ntfs.WrapError(fmt.Errorf("Can't access to source file"))
	}

	f, err := ntfs.OpenFile(source, ntfs.OPEN_RDONLY)
	if err != nil {
		return err
	}

	arg.source = f

	return nil
}

func do_set_destination(dest string, arg *tActionArg) error {
	if dest == "" {
		return ntfs.WrapError(fmt.Errorf("No destination file specified"))
	}

	f, err := ntfs.OpenFile(dest, ntfs.OPEN_WRONLY)
	if err != nil {
		return err
	}

	arg.dest = f

	return nil
}

func do_count(pattern string, arg *tActionArg) error {
	if pattern == "" {
		return ntfs.WrapError(errors.New("No pattern specified"))
	}

	patterns := strings.Split(pattern, ",")
	list := make([][]byte, len(patterns)-1)
	for _, p := range patterns[1:] {
		if p != "" {
			list = append(list, []byte(p))
		}
	}

	indexes, err := inspect.FindPositionsWithPattern(arg.partition, []byte(patterns[0]), list...)
	if err != nil {
		return err
	}

	lengths := make([]int, len(patterns))
	for i, idx_list := range indexes {
		lengths[i] = len(idx_list)
	}

	fmt.Println("Results:")
	for i, idx_list := range indexes {
		sz := lengths[i]
		if sz > 10 {
			sz = 10
		}

		fmt.Println(fmt.Sprintf("  - `%s`: Count=%d Firsts=%v", patterns[i], lengths[i], idx_list[:sz]))
	}

	return nil
}

func do_names(arg *tActionArg) error {
	fmt.Println()
	fmt.Println("Names:")
	for i := ntfs.FileReferenceNumber(0); true; i++ {
		var val ntfs.RecordHeader

		if err := arg.disk.ReadRecordHeaderFromRef(i, &val); err != nil {
			return err
		}

		if !val.Type.IsGood() {
			return nil
		}

		if val.Type != ntfs.RECTYP_FILE {
			continue
		}

		var file ntfs.FileRecord

		if err := arg.disk.ReadFileRecordFromRef(i, &file); err != nil {
			return err
		}

		name, err := arg.disk.GetFileRecordFilename(&file)
		if err != nil {
			return err
		}

		if name == "" {
			continue
		}

		fmt.Println(fmt.Sprintf("  - %d : %s", i.GetFileIndex(), name))
	}

	return nil
}

func do_record(record_num int64, arg *tActionArg) error {
	var record ntfs.RecordHeader

	if err := arg.disk.ReadRecordHeader(record_num, &record); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("Record:")
	ntfs.PrintStruct(record)

	return nil
}

func do_sector(offset int64, arg *tActionArg) error {
	var sector [512]byte

	num := (offset + 511) / 512
	if err := arg.disk.GetDisk().ReadSector(num, sector[:]); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("Content:")
	fmt.Printf("Content at %d:", num)
	fmt.Println()
	ntfs.PrintBytes(sector[:])

	return nil
}

func do_cluster(offset int64, arg *tActionArg) error {
	var cluster [4096]byte

	num := (offset + 4095) / 4096
	if err := arg.disk.GetDisk().ReadCluster(num, cluster[:]); err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("Content at %d:", num*8)
	fmt.Println()
	ntfs.PrintBytes(cluster[:])

	return nil
}

func do_scan(arg *tActionArg) error {
	destination, err := arg.GetOutput()
	if err != nil {
		return err
	}

	return inspect.Scan(arg.partition, destination)
}
