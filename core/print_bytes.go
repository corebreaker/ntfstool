package core

import (
	"fmt"
    "unicode"
)

func PrintBytes(b []byte) {
    line := ""
	for i, v := range b {
		if (i % 16) == 0 {
			fmt.Printf("%08x:", i)
		}

		if (i % 2) == 0 {
			fmt.Print(" ")
		}

		fmt.Printf("%02x", v)
        r := rune(v)
        if unicode.IsPrint(r) {
            line += fmt.Sprintf("%c", r)
        } else {
            line += "."
        }

		if ((i + 1) % 16) == 0 {
			fmt.Println(" ", line)
            line = ""
		}
	}
}
