package main

import (
	"fmt"
	"os"
)

func do_help(arg *tActionArg) error {
	prog := os.Args[0] + " " + _HELP_PARTITION_EXAMPLE

	fmt.Println("Usage:", os.Args[0], "partition parameters")
	fmt.Println()
	fmt.Println("  partition  =", _HELP_PARTITION_DESC)
	fmt.Println("  parameters = space separated list of parameters formated as following:")
	fmt.Println("     - `name=value` for a parameter with a value, `name` is the parameter name")
	fmt.Println("     - `name` for a parameter without any value")
	fmt.Println()
	fmt.Println("   example:", prog, "param1=val2 param2=value2 param3=value3")
	fmt.Println(`
Some parameters configures commands. This is the list of command configurators:
  - in=pathname:     specifies an input file for others commands
  - out=pathname:    specifies an output file for others commands
  - file=pathname:   specifies a input/output file for others commands
  - mft=offset:      specifies the MFT shift from the partition starting with an offset in the partition
  - start=offset:    specifies the offset in the partition where the readind starts (partition start)
  - from=file-id:    specifies a file ID or a directorry ID for others commands
  - to=dest:         specifies a ` + "`" + `dest` + "`" + ` file or directory pathname for others commands

Some parameters are commands.
For input and output files, there are 2 file formats (used for file recovery:
  - The state format: to prepare recovery (intermediate states of MFT entries),
  - The file node format: definive format by registering files, directories and their tree.

Global commands:
  - help:            shows this help

Commands to inspect the partition:
  - count=pattern:   shows the counts of the records that follows the record type pattern
  - mft-names:       shows names of MFT records in MFT from the partition
  - record=offset:   shows MFT record read from partition with an offset in MFT
  - sector=offset:   shows the sector with its offset in the partition
  - cluster=offset:  shows the cluster with its offset in the partition
  - file-num=number: inspects file records in MFT from the partition

Commands to explore the input file:
  - record-count:    shows count of file records in the input file with a file node format
  - positions=n:     shows the ` + "`" + `n` + "`" + ` fisrt positions of the records in the input file
  - head=n:          shows the ` + "`" + `n` + "`" + ` fisrt records in the input file
  - tail=n:          shows the ` + "`" + `n` + "`" + ` last records in the input file
  - list-names:      list filenames in input file with the state format
  - show-names:      show filenames and their parent in input file
  - show-mft=id:     shows the MFT from its ID in the input file with the state format
  - show=n:          shows n-th record in the input file, first record has ` + "`" + `n` + "`" + ` equal to zero
  - at=offset:       shows the record in the input file at the specified file position (offset)
  - find-state:      find a record in the input file in state format
  - find-file=query: find a record in the input file in file node format
  - show-attr=attr   shows the attribute from its position for a state file record in the input file
  - id=file-id:      shows the record with file ID in the input file in file node format
  - parent=file-id:  shows children files of file ID in the input file in file node format
  - parent-ref=idx:  shows children files of file index in the input file in file node format
  - check:           checks the integrity of data structures in the input file in the state format
  - ls[=dir-id]:     list files in directory from the input file in file node format
  - move-to=dir-id:  moves a file or a directory to a directory from input filn in file node format
  - mkdir=name:      create a directory to a directory from input filn in file node format

Commands for file recovery:
  - scan:            scans the partition to find MFTs and MFT records and report them to the output file
  - fill:            fill data info from the input file into the output file (in the state format)
  - fix-mft:         fixes MFT entries from the input file into the output file (in the state format)
  - complete:        completes datas from the input file into the output file (in the state format)
  - make-filelist:   builds the file list from the input file (states) into the output file (file nodes)
  - cp=file-id:      copy file from partition into the output file with the help of the input file

Offset has unit suffixes:
  - c = clusters, example: 2c = 2 clusters
  - s = sectors, example: 4s = 4 sectors (2Ko)
`)
	fmt.Println("Shows the content of the MBR:", prog, "(with no parameter)")
	fmt.Println()
	fmt.Println("For inspecting file records in MFT from partition:")
	fmt.Println("  -", prog, "mft=2c file-num=0 raw")
	fmt.Println()
	fmt.Println("Chain of commands for file recovery:")
	fmt.Println(" 1.", prog, "out=00_scan.dat scan")
	fmt.Println(" 2.", prog, "in=00_scan.dat out=01_base.dat fill")
	fmt.Println(" 3.", prog, "in=01_base.dat out=02_records.dat fix-mft")
	fmt.Println(" 4.", prog, "in=02_records.dat out=03_files.dat complete")
	fmt.Println(" 5.", prog, "in=03_files.dat out=04_fslist.dat make-filelist")
	fmt.Println(" 6.", prog, "in=04_fslist.dat ls")
	fmt.Println(" 7.", prog, "in=04_fslist.dat cp=c3eb25a23f0b4448a8fc94ce521847e2 to=recovery.dir")
	fmt.Println()

	return nil
}
