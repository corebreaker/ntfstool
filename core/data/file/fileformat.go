package file

import (
	"fmt"

	"github.com/corebreaker/ntfstool/core"
	"github.com/corebreaker/ntfstool/core/data"
	"github.com/corebreaker/ntfstool/core/data/codec"
)

const (
	ANY_FILEFORMAT   = "<All>"
	SIGNATURE_LENGTH = 16
)

type tFileFormat struct {
	name      string
	signature []byte
	headers   []data.IDataRecord
	registry  *codec.Registry
}

func (self *tFileFormat) String() string {
	return fmt.Sprint("Name:", self.name, "/ Headers:", self.headers)
}

var (
	file_formats    map[string]*tFileFormat = make(map[string]*tFileFormat)
	file_signatures map[string]*tFileFormat = make(map[string]*tFileFormat)
)

func RegisterFileFormat(name, signature string, headers ...data.IDataRecord) {
	var common_headers []data.IDataRecord
	var registry *codec.Registry

	if name != ANY_FILEFORMAT {
		ref, ok := file_formats[ANY_FILEFORMAT]
		if !ok {
			RegisterFileFormat(ANY_FILEFORMAT, "", new(tNullRecord), new(tFileDesc))
			ref = file_formats[ANY_FILEFORMAT]
		}

		common_headers = ref.headers
		registry = ref.registry.SubRegistry()
	} else {
		registry = codec.MakeRegistry()
	}

	sz := len(common_headers)
	res_headers := make([]data.IDataRecord, sz+len(headers))
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
	if name == ANY_FILEFORMAT {
		return nil
	}

	res, ok := file_formats[name]
	if !ok {
		return nil
	}

	return res.registry
}
