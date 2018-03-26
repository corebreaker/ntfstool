package core

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"runtime"
)

type tError struct {
	source  error
	message string
	trace   []string
}

func (self *tError) Error() string {
	var out bytes.Buffer

	fmt.Fprintln(&out, self.source.Error())
	fmt.Fprint(&out, self.message)
	for _, line := range self.trace {
		fmt.Fprintln(&out, "   ", line)
	}

	if len(self.trace) > 0 {
		fmt.Fprintln(&out, "-------------------------------------------------------------------------------")
	}

	return out.String()
}

func (self *tError) add_info(format string, args ...interface{}) error {
	var out bytes.Buffer

	fmt.Fprintln(&out, fmt.Sprintf(format, args...))

	self.message += out.String()

	return self
}

func wrap_err(err error) *tError {
	if err == nil {
		return nil
	}

	var pc uintptr = 1

	stack := make([]string, 0)
	for i := 2; pc != 0; i++ {
		ptr, file, line, ok := runtime.Caller(i)
		pc = ptr
		if (pc == 0) || (!ok) {
			continue
		}

		f := runtime.FuncForPC(pc)

		stack = append(stack, fmt.Sprintf("%s (%s:%d)", f.Name(), file, line))
	}

	res, ok := err.(*tError)
	if ok {
		res.trace = stack
	} else {
		res = &tError{
			source: err,
			trace:  stack,
		}
	}

	return res
}

func Recover() error {
	z_err := recover()
	if z_err == nil {
		return nil
	}

	err, ok := z_err.(error)
	if !ok {
		return WrapError(fmt.Errorf("%v", err))
	}

	return EnsureWrapped(err)
}

func IsWrapped(err error) bool {
	_, ok := err.(*tError)

	return ok
}

func GetSource(err error) error {
	if err == nil {
		return nil
	}

	t_err, ok := err.(*tError)
	if !ok {
		return err
	}

	return t_err.source
}

func EnsureWrapped(err error) error {
	if err == nil {
		return nil
	}

	if IsWrapped(err) {
		return err
	}

	return wrap_err(err)
}

func WrapError(err error) error {
	if err == nil {
		return nil
	}

	return wrap_err(err)
}

func AddErrorInfo(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}

	res_err, ok := err.(*tError)
	if !ok {
		res_err = wrap_err(err)
	}

	return res_err.add_info(format, args...)
}

func IsEof(err error) bool {
	res := err == io.EOF
	if !res {
		t_err, ok := err.(*tError)
		if ok {
			res = IsEof(t_err.source)
		}
	}

	return res
}

func PrintError(err error) {
	if err != nil {
		fmt.Println(EnsureWrapped(err).Error())
	}
}

func Abort(err error) {
	PrintError(err)
	os.Exit(1)
}

type Handler func() error

func CheckedMain(main_task_func Handler) {
	if err := main_task_func(); err != nil {
		Abort(err)
	}
}

func DeferedCall(defered_func Handler) {
	PrintError(defered_func())
}

func DiscardPanic() {
	recover()
}
