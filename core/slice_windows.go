//+build windows

package core

func FillBuffer(b []byte, v byte) {
    for i := range b {
        b[i] = v
    }
}
