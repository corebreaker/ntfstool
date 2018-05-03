//+build !windows

package core

/*
#include <string.h>
*/
import "C"
import "unsafe"
import "reflect"

func FillBuffer(b []byte, v byte) {
    slice := (*reflect.SliceHeader)(unsafe.Pointer(&b))

    C.memset(unsafe.Pointer(slice.Data), C.int(v), C.size_t(slice.Len))
}
