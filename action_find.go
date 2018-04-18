package main

import (
	"bytes"
	"fmt"

	"essai/ntfstool/core"
	"essai/ntfstool/extract"
	"essai/ntfstool/inspect"

	"github.com/gobwas/glob"
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

func do_find_file(pattern string, arg *tActionArg) error {
	p, err := glob.Compile(pattern)
	if err != nil {
		return core.WrapError(err)
	}

	src, err := arg.GetInput()
	if err != nil {
		return err
	}

	fmt.Println("Reading")
	tree, err := extract.ReadTreeFromFile(src)
	if err != nil {
		return err
	}

	var stream extract.FileStream

	from := arg.GetFromParam()
	if from == "" {
		stream = tree.MakeStream()
	} else {
		stream = tree.MakeStreamFrom(from)
		if stream == nil {
			return core.WrapError(fmt.Errorf("Id `%s` not found", from))
		}
	}

	defer core.DeferedCall(stream.Close)

	fmt.Println()
	fmt.Println("Result:")
	for item := range stream {
		rec := item.Record()
		if rec.IsNull() || (!rec.HasName()) || (rec.GetFile() == nil) || (!p.Match(rec.GetName())) {
			continue
		}

		path := tree.GetFilePathFromFile(rec)
		if !p.Match(path) {
			continue
		}

		fmt.Printf("  - %d: %s", rec.GetId(), path)
	}

	return nil
}
