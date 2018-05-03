package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func ToInt(v string) (int64, error) {
	if len(v) == 0 {
		return 0, nil
	}

	res, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, WrapError(err)
	}

	return res, nil
}

func ToBool(v string) (bool, error) {
	if len(v) == 0 {
		return false, nil
	}

	res, err := strconv.ParseBool(v)
	if err != nil {
		return false, WrapError(err)
	}

	return res, nil
}

type Args map[string]string

func (self Args) Bool(key string) bool {
	return self.BoolDef(key, false)
}

func (self Args) BoolDef(key string, default_value bool) bool {
	val, ok, err := self.BoolFull(key)
	if err != nil {
		Abort(err)
	}

	if !ok {
		return default_value
	}

	return val
}

func (self Args) BoolExt(key string) (bool, bool) {
	val, ok, err := self.BoolFull(key)
	if (!ok) || (err != nil) {
		return false, false
	}

	return val, true
}

func (self Args) BoolFull(key string) (bool, bool, error) {
	v, ok := self[key]
	if !ok {
		return false, false, nil
	}

	res, err := ToBool(v)
	if err != nil {
		return false, false, err
	}

	return res, true, nil
}

func (self Args) Int(key string) int64 {
	return self.IntDef(key, 0)
}

func (self Args) IntDef(key string, default_value int64) int64 {
	val, ok, err := self.IntFull(key)
	if err != nil {
		Abort(err)
	}

	if !ok {
		return default_value
	}

	return val
}

func (self Args) IntExt(key string) (int64, bool) {
	val, ok, err := self.IntFull(key)
	if (!ok) || (err != nil) {
		return 0, false
	}

	return val, true
}

func (self Args) IntFull(key string) (int64, bool, error) {
	v, ok := self[key]
	if !ok {
		return 0, false, nil
	}

	res, err := ToInt(v)
	if err != nil {
		return 0, false, err
	}

	return res, true, nil
}

func (self Args) Idx(key string) int64 {
	return self.IdxDef(key, 0)
}

func (self Args) IdxDef(key string, default_value int64) int64 {
	val, ok, err := self.IdxFull(key)
	if err != nil {
		Abort(err)
	}

	if !ok {
		return default_value
	}

	return val
}

func (self Args) IdxExt(key string) (int64, bool) {
	val, ok, err := self.IdxFull(key)
	if (!ok) || (err != nil) {
		return 0, false
	}

	return val, true
}

func (self Args) IdxFull(key string) (int64, bool, error) {
	v, ok := self[key]
	if !ok {
		return 0, false, nil
	}

	last := len(v) - 1

	switch {
	case strings.HasSuffix(v, "c"):
		res, err := ToInt(v[:last])
		if err != nil {
			return 0, false, err
		}

		return res * int64(4096), true, nil

	case strings.HasSuffix(v, "s"):
		res, err := ToInt(v[:last])
		if err != nil {
			return 0, false, err
		}

		return res * int64(512), true, nil

	default:
		res, err := ToInt(v)
		if err != nil {
			return 0, false, err
		}

		return res, true, nil
	}
}

func GetArgs() Args {
	res := make(Args)
	for _, a := range os.Args[2:] {
		idx := strings.IndexRune(a, '=')
		key := a
		val := ""

		if idx >= 0 {
			key, val = a[:idx], a[(idx+1):]
		}

		res[key] = val
	}

	return res
}

func GetDirectory() (string, error) {
	res, err := filepath.Abs(os.Args[0])
	if err != nil {
		return "", WrapError(err)
	}

	return filepath.Dir(res), err
}

func WinDiskPath(letter rune) string {
	return fmt.Sprintf("\\\\.\\%c:", letter)
}

func GetPartition() string {
	if len(os.Args) <= 1 {
		return ""
	}

	res := os.Args[1]
	if strings.HasPrefix(res, "@") {
		return res[1:]
	}

	if len(res) == 1 {
		return WinDiskPath([]rune(res)[0])
	}

	return "/dev/" + res
}
