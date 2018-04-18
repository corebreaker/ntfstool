package core

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
)

type BootFormat [8]byte

func (bf BootFormat) String() string {
	return string(bf[:])
}

type BootCode [0x1AE]byte

func (bc BootCode) String() string {
	var out bytes.Buffer

	fmt.Fprintln(&out)
	FprintBytes(&out, bc[:])

	return out.String()
}

type BootBlock struct {
	Jump                  [3]Byte
	Format                BootFormat
	BytesPerSector        uint16
	SectorsPerCluster     uint8
	BootSectors           uint16
	Mbz1                  uint8
	Mbz2                  uint16
	Reserved1             uint16
	MediaType             uint8
	Mbz3                  uint16
	SectorsPerTrack       uint16
	NumberOfHeads         uint16
	PartitionOffset       uint32
	Reserved2             [2]uint32
	TotalSectors          uint64
	MftStartLcn           ClusterNumber
	Mft2StartLcn          ClusterNumber
	ClustersPerFileRecord uint32
	ClustersPerIndexBlock uint32
	VolumeSerialNumber    uint64
	Code                  BootCode
	BootSignature         uint16
}

func PrintBoot(disk_name string) {
	f, err := os.Open(disk_name)
	if err != nil {
		Abort(err)
	}

	defer DeferedCall(f.Close)

	res := new(BootBlock)

	err = binary.Read(f, binary.LittleEndian, res)
	if err != nil {
		Abort(err)
	}

	PrintStruct(res)
}
