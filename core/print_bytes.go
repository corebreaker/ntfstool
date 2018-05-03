package core

import (
	"fmt"
	"io"
	"os"
	"unicode"
)

func PrintBytes(b []byte) {
	FprintBytes(os.Stdout, b)
}

func FprintBytes(w io.Writer, b []byte) {
	line := ""
	for i, v := range b {
		if (i % 16) == 0 {
			fmt.Fprintf(w, "%08x:", i)
		}

		if (i % 2) == 0 {
			fmt.Fprint(w, " ")
		}

		fmt.Fprintf(w, "%02x", v)
		r := rune(v)
		if unicode.IsPrint(r) {
			line += fmt.Sprintf("%c", r)
		} else {
			line += "."
		}

		if ((i + 1) % 16) == 0 {
			fmt.Fprintln(w, " ", line)
			line = ""
		}
	}

	sz := len(b) % 16
	if sz != 0 {
		for i := sz; i < 16; i++ {
			if (i % 2) == 0 {
				fmt.Fprint(w, " ")
			}

			fmt.Fprint(w, "  ")
		}

		fmt.Fprintln(w, " ", line)
	}
}
