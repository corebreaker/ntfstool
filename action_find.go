package main

import (
	"bytes"
	"fmt"

	"essai/ntfstool/core"
	"essai/ntfstool/inspect"
)

func do_find_state(arg *tActionArg) error {
	src, err := arg.GetInput()
	if err != nil {
		return err
	}

	fmt.Println("Reading")
	states, err := inspect.MakeStateReader(src)
	if err != nil {
		return err
	}

	defer core.DeferedCall(states.Close)

	stream, err := states.MakeStream()
	if err != nil {
		return err
	}

	defer core.DeferedCall(stream.Close)

	position, pos_ok := arg.IdxExt("position")

	i, sz := 0, states.GetCount()

	var out bytes.Buffer

	min_pos, max_pos := position, position
	min_idx, max_idx := 0, 0
	pos_found := false

	fmt.Println("Searching")
	for item := range stream {
		fmt.Printf("\r%d %%", i*100/sz)
		i++

		state := item.Record()

		if err := state.GetError(); err != nil {
			return err
		}

		if state.IsNull() {
			continue
		}

		if pos_ok {
			idx := item.Index()
			p := state.GetPosition()

			if p == position {
				fmt.Fprintln(&out, "  - Position found at:", item.Index())
				pos_found = true
			}

			if (min_pos < p) && (p <= position) {
				min_pos = p
				min_idx = idx
			}

			if (max_pos > p) && (p >= position) {
				max_pos = p
				max_idx = idx
			}
		}
	}

	fmt.Println("\rDone.")
	fmt.Println()

	fmt.Println("Result:")
	fmt.Println(&out)
	if pos_ok {
		if !pos_found {
			const msg = "Position between indexes %d (position= %d) and %d (position= %d)"

			fmt.Printf(msg, min_idx, min_pos, max_idx, max_pos)
			fmt.Println()
		}
	}

	return nil
}
