package codec

import (
    "io"

    "github.com/DeDiS/protobuf"
    fb "github.com/google/flatbuffers/go"
    "github.com/tinylib/msgp/msgp"
)

type Decoder struct {
    reader io.Reader
}

func (self *Decoder) Decode(ref interface{}) error {
    fb.NewBuilder()
}
