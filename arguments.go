package main

import (
	"fmt"
	"os"
	"path/filepath"

	ntfs "essai/ntfstool/core"
	"essai/ntfstool/inspect"
)

type tActionType byte

const (
	_ACT_CONFIG tActionType = iota
	_ACT_DEFAULT
	_ACT_BOOLEAN
	_ACT_STRING
	_ACT_INTEGER
	_ACT_INDEX
)

type tActionArg struct {
	partition string
	disk      *inspect.NtfsDisk
	source    *os.File
	dest      *os.File
	file      *os.File
	from      string
	to        string
	_args     ntfs.Args
}

func (self *tActionArg) Close() error {
	if self.disk != nil {
		self.disk.Close()
	}

	if self.source != nil {
		self.source.Close()
	}

	if self.dest != nil {
		self.dest.Close()
	}

	return nil
}

func (self *tActionArg) Get(key string) string {
	return self._args[key]
}

func (self *tActionArg) GetDef(key, default_value string) string {
	res, ok := self._args[key]
	if !ok {
		res = default_value
	}

	return res
}

func (self *tActionArg) GetExt(key string) (string, bool) {
	res, ok := self._args[key]

	return res, ok
}

func (self *tActionArg) Bool(key string) bool                    { return self._args.Bool(key) }
func (self *tActionArg) BoolDef(key string, val bool) bool       { return self._args.BoolDef(key, val) }
func (self *tActionArg) BoolExt(key string) (bool, bool)         { return self._args.BoolExt(key) }
func (self *tActionArg) BoolFull(key string) (bool, bool, error) { return self._args.BoolFull(key) }
func (self *tActionArg) Int(key string) int64                    { return self._args.Int(key) }
func (self *tActionArg) IntDef(key string, val int64) int64      { return self._args.IntDef(key, val) }
func (self *tActionArg) IntExt(key string) (int64, bool)         { return self._args.IntExt(key) }
func (self *tActionArg) IntFull(key string) (int64, bool, error) { return self._args.IntFull(key) }
func (self *tActionArg) Idx(key string) int64                    { return self._args.Idx(key) }
func (self *tActionArg) IdxDef(key string, val int64) int64      { return self._args.IdxDef(key, val) }
func (self *tActionArg) IdxExt(key string) (int64, bool)         { return self._args.IdxExt(key) }
func (self *tActionArg) IdxFull(key string) (int64, bool, error) { return self._args.IdxFull(key) }

func (self *tActionArg) GetInput() (*os.File, error) {
	if self.source == nil {
		return nil, ntfs.WrapError(fmt.Errorf("No source file specified"))
	}

	return self.source, nil
}

func (self *tActionArg) GetOutput() (*os.File, error) {
	if self.dest == nil {
		return nil, ntfs.WrapError(fmt.Errorf("No destination file specified"))
	}

	return self.dest, nil
}

func (self *tActionArg) GetFile() (*os.File, error) {
	if self.file == nil {
		return nil, ntfs.WrapError(fmt.Errorf("No file specified"))
	}

	return self.file, nil
}

func (self *tActionArg) GetFiles() (src, dest *os.File, err error) {
	src, err = self.GetInput()
	if err == nil {
		dest = self.dest
		if dest == nil {
			dest, err = ntfs.OpenFile(filepath.Join(filepath.Dir(self.source.Name()), "records.dat"), ntfs.OPEN_WRONLY)
			if err != nil {
				return nil, nil, err
			}
		}
	}

	return
}

func (self *tActionArg) GetTransferFiles(dest_default string) (*os.File, string, error) {
	src, err := self.GetInput()
	if err != nil {
		return nil, "", err
	}

	dest := self.GetToParam()
	if len(dest) == 0 {
		if len(dest_default) == 0 {
			return nil, "", ntfs.WrapError(fmt.Errorf("No destination file specified"))
		}

		dest = dest_default
	}

	return src, dest, nil
}

func (self *tActionArg) GetFromParam() string {
	return self.from
}

func (self *tActionArg) GetToParam() string {
	return self.to
}

type iActionDef interface {
	get_name() string
	handle_str(value string, arg *tActionArg) error
	handle_int(value int64, arg *tActionArg) error
	handle_bool(value bool, arg *tActionArg) error
	action_type() tActionType
	do_stop() bool
}

type tConfigActionDef struct {
	handler func(*tActionArg) error
}

func (self tConfigActionDef) get_name() string                         { return "" }
func (self tConfigActionDef) action_type() tActionType                 { return _ACT_CONFIG }
func (self tConfigActionDef) do_stop() bool                            { return false }
func (self tConfigActionDef) handle_int(v int64, a *tActionArg) error  { return self.handler(a) }
func (self tConfigActionDef) handle_str(v string, a *tActionArg) error { return self.handler(a) }
func (self tConfigActionDef) handle_bool(v bool, a *tActionArg) error  { return self.handler(a) }

type tDefaultActionDef struct {
	handler func(*tActionArg) error
	name    string
	next    bool
}

func (self tDefaultActionDef) get_name() string                         { return self.name }
func (self tDefaultActionDef) action_type() tActionType                 { return _ACT_DEFAULT }
func (self tDefaultActionDef) do_stop() bool                            { return !self.next }
func (self tDefaultActionDef) handle_int(v int64, a *tActionArg) error  { return self.handler(a) }
func (self tDefaultActionDef) handle_str(v string, a *tActionArg) error { return self.handler(a) }
func (self tDefaultActionDef) handle_bool(v bool, a *tActionArg) error  { return self.handler(a) }

type tBoolActionDef struct {
	handler func(bool, *tActionArg) error
	name    string
	next    bool
}

func (self tBoolActionDef) get_name() string                         { return self.name }
func (self tBoolActionDef) action_type() tActionType                 { return _ACT_BOOLEAN }
func (self tBoolActionDef) do_stop() bool                            { return !self.next }
func (self tBoolActionDef) handle_int(v int64, a *tActionArg) error  { return nil }
func (self tBoolActionDef) handle_str(v string, a *tActionArg) error { return nil }
func (self tBoolActionDef) handle_bool(v bool, a *tActionArg) error  { return self.handler(v, a) }

type tStringActionDef struct {
	handler func(string, *tActionArg) error
	name    string
	next    bool
}

func (self tStringActionDef) get_name() string                         { return self.name }
func (self tStringActionDef) action_type() tActionType                 { return _ACT_STRING }
func (self tStringActionDef) do_stop() bool                            { return !self.next }
func (self tStringActionDef) handle_int(v int64, a *tActionArg) error  { return nil }
func (self tStringActionDef) handle_str(v string, a *tActionArg) error { return self.handler(v, a) }
func (self tStringActionDef) handle_bool(v bool, a *tActionArg) error  { return nil }

type tIntegerActionDef struct {
	handler func(int64, *tActionArg) error
	name    string
	next    bool
	offset  bool
}

func (self tIntegerActionDef) get_name() string                         { return self.name }
func (self tIntegerActionDef) do_stop() bool                            { return !self.next }
func (self tIntegerActionDef) handle_int(v int64, a *tActionArg) error  { return self.handler(v, a) }
func (self tIntegerActionDef) handle_str(v string, a *tActionArg) error { return nil }
func (self tIntegerActionDef) handle_bool(v bool, a *tActionArg) error  { return nil }
func (self tIntegerActionDef) action_type() tActionType {
	if self.offset {
		return _ACT_INDEX
	}

	return _ACT_INTEGER
}

func run_action(action iActionDef, arg *tActionArg) (bool, error) {
	var err error

	name := action.get_name()

	switch action.action_type() {
	case _ACT_CONFIG:
		err = action.handle_int(0, arg)

	case _ACT_BOOLEAN:
		value, ok, i_err := arg.BoolFull(name)
		if i_err != nil {
			return false, i_err
		}

		if !ok {
			return false, nil
		}

		err = action.handle_bool(value, arg)

	case _ACT_STRING:
		value, ok := arg.GetExt(name)
		if !ok {
			return false, nil
		}

		err = action.handle_str(value, arg)

	case _ACT_INTEGER:
		value, ok, i_err := arg.IntFull(name)
		if i_err != nil {
			return false, i_err
		}

		if !ok {
			return false, nil
		}

		err = action.handle_int(value, arg)

	case _ACT_INDEX:
		value, ok, i_err := arg.IdxFull(name)
		if err != nil {
			return false, i_err
		}

		if !ok {
			return false, nil
		}

		err = action.handle_int(value, arg)

	case _ACT_DEFAULT:
		if _, ok := arg.GetExt(name); !ok {
			return false, nil
		}

		err = action.handle_int(0, arg)
	}

	if err != nil {
		return false, err
	}

	return action.do_stop(), nil
}
