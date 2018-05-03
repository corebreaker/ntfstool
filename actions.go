package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/siddontang/go/ioutil2"

	ntfs "github.com/corebreaker/ntfstool/core"
	"github.com/corebreaker/ntfstool/core/data"
	datafile "github.com/corebreaker/ntfstool/core/data/file"
	"github.com/corebreaker/ntfstool/inspect"
)

var (
	actions = []iActionDef{
		tDefaultActionDef{handler: do_help, name: "help"},

		// Commands using input/output files
		tStringActionDef{handler: do_set_source, name: "in", next: true},
		tStringActionDef{handler: do_set_destination, name: "out", next: true},
		tStringActionDef{handler: do_set_file, name: "file", next: true},
		tStringActionDef{handler: do_set_from_param, name: "from", next: true},
		tStringActionDef{handler: do_set_into_param, name: "to", next: true},
		tDefaultActionDef{handler: do_record_count, name: "record-count"},
		tIntegerActionDef{handler: do_show, name: "show"},
		tIntegerActionDef{handler: do_head, name: "head"},
		tIntegerActionDef{handler: do_tail, name: "tail"},
		tIntegerActionDef{handler: do_offsets, name: "positions"},
		tDefaultActionDef{handler: do_find_state, name: "find"},
		tStringActionDef{handler: do_show_id, name: "id"},
		tStringActionDef{handler: do_show_parent, name: "parent"},
		tIntegerActionDef{handler: do_show_parent_ref, name: "parent-ref"},
		tIntegerActionDef{handler: do_position, name: "at", offset: true},
		tBoolActionDef{handler: do_check, name: "check"},
		tIntegerActionDef{handler: do_show_attribute, name: "show-attr"},
		tStringActionDef{handler: do_show_mft, name: "show-mft"},
		tDefaultActionDef{handler: do_listnames, name: "list-names"},
		tDefaultActionDef{handler: do_shownames, name: "show-names"},
		tDefaultActionDef{handler: do_scan, name: "scan"},
		tStringActionDef{handler: do_list_files, name: "ls"},
		tStringActionDef{handler: do_move_to, name: "mv"},
		tStringActionDef{handler: do_copy_to, name: "cp"},
		tStringActionDef{handler: do_remove_from, name: "rm"},
		tStringActionDef{handler: do_make_dir, name: "mkdir"},
		tDefaultActionDef{handler: do_compact, name: "compact"},

		// Commands to use partition with the help of input/output files
		tConfigActionDef{handler: do_open_disk},
		tStringActionDef{handler: do_save_file, name: "save"},
		tDefaultActionDef{handler: do_fillinfo, name: "fill"},
		tBoolActionDef{handler: do_fixmft, name: "fix-mft"},
		tBoolActionDef{handler: do_mkfilelist, name: "make-filelist"},
		tDefaultActionDef{handler: do_complete, name: "complete"},

		// Command to explore partition
		tIntegerActionDef{handler: do_start, name: "start", next: true, offset: true},
		tIntegerActionDef{handler: do_mft, name: "mft", next: true, offset: true},
		tStringActionDef{handler: do_count, name: "count"},
		tDefaultActionDef{handler: do_mftnames, name: "mft-names"},
		tIntegerActionDef{handler: do_record, name: "record", offset: true},
		tIntegerActionDef{handler: do_sector, name: "sector", offset: true},
		tIntegerActionDef{handler: do_cluster, name: "cluster", offset: true},
		tIntegerActionDef{handler: do_file_num, name: "file-num"},
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

func do_set_file(pathname string, arg *tActionArg) error {
	if pathname == "" {
		return ntfs.WrapError(fmt.Errorf("No destination file specified"))
	}

	f, err := ntfs.OpenFile(pathname, ntfs.OPEN_RDWR)
	if err != nil {
		return err
	}

	arg.file = f

	return nil
}

func do_set_from_param(val string, arg *tActionArg) error {
	arg.from = val

	return nil
}

func do_set_into_param(val string, arg *tActionArg) error {
	arg.into = val

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

func do_record_count(arg *tActionArg) error {
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

	fmt.Println()
	fmt.Println("Format:", records.GetFormatName())

	fmt.Println()
	for rec_type, count := range records.GetCounts() {
		fmt.Println(rec_type.GetLabel(), "=", count)
	}

	fmt.Println()
	fmt.Println("Total Record Count =", records.GetCount())

	return nil
}

func do_mftnames(arg *tActionArg) error {
	fmt.Println()
	fmt.Println("Names:")
	for i := data.FileRef(0); true; i++ {
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

	disk := arg.disk.GetDisk()
	defer ntfs.DeferedCall(disk.Close)

	if err := disk.ReadSector(num, sector[:]); err != nil {
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

	disk := arg.disk.GetDisk()
	defer ntfs.DeferedCall(disk.Close)

	if err := disk.ReadCluster(num, cluster[:]); err != nil {
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
