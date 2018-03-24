package datafile

import (
	"fmt"

	"essai/ntfstool/core"
	"essai/ntfstool/core/dataio"
	"essai/ntfstool/core/dataio/codec"
)

const (
	COMMON_DATA_FILEFORMAT_NAME = "<All>"
	SIGNATURE_LENGTH            = 16
)

type tFileFormat struct {
	name      string
	signature []byte
	headers   []dataio.IDataRecord
	registry  *codec.Registry
}

func (self *tFileFormat) String() string {
	return fmt.Sprint("Name:", self.name, "/ Headers:", self.headers)
}

var (
	file_formats    map[string]*tFileFormat = make(map[string]*tFileFormat)
	file_signatures map[string]*tFileFormat = make(map[string]*tFileFormat)
)

func RegisterFileFormat(name, signature string, headers ...dataio.IDataRecord) {
	var common_headers []dataio.IDataRecord
	var registry *codec.Registry

	if name != COMMON_DATA_FILEFORMAT_NAME {
		ref, ok := file_formats[COMMON_DATA_FILEFORMAT_NAME]
		if !ok {
			RegisterFileFormat(COMMON_DATA_FILEFORMAT_NAME, "", new(tNullRecord), new(tFileDesc))
			ref = file_formats[COMMON_DATA_FILEFORMAT_NAME]
		}

		common_headers = ref.headers
		registry = ref.registry.SubRegistry()
	} else {
		registry = codec.MakeRegistry()
	}

	sz := len(common_headers)
	res_headers := make([]dataio.IDataRecord, sz+len(headers))
	copy(res_headers, common_headers)
	copy(res_headers[sz:], headers)

	res := &tFileFormat{
		signature: make([]byte, SIGNATURE_LENGTH),
		name:      name,
		headers:   res_headers,
		registry:  registry,
	}

	core.FillBuffer(res.signature, ' ')
	copy(res.signature, signature)

	for _, header := range headers {
		registry.RegisterName(header.GetEncodingCode(), header)
	}

	file_formats[name] = res
	file_signatures[string(res.signature)] = res
}

func GetRegistry(name string) *codec.Registry {
	if name == COMMON_DATA_FILEFORMAT_NAME {
		return nil
	}

	res, ok := file_formats[name]
	if !ok {
		return nil
	}

	return res.registry
}
