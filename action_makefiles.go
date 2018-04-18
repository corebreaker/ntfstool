package main

import (
	"bytes"
	"fmt"
	"os"

	"essai/ntfstool/core"
	"essai/ntfstool/core/data"
	"essai/ntfstool/extract"
	"essai/ntfstool/inspect"
)

func do_mkfilelist(verbose bool, arg *tActionArg) error {
	src, dest, err := arg.GetFiles()
	if err != nil {
		return err
	}

	fmt.Println("Reading")
	reader, err := inspect.MakeStateReader(src)
	if err != nil {
		return err
	}

	defer core.DeferedCall(reader.Close)

	stream, err := reader.MakeStream()
	if err != nil {
		return err
	}

	type tNode struct {
		file     *extract.File
		children map[string]*tNode

		addChild  func(child *tNode) bool
		setParent func(child *tNode)
		makeNode  func() *extract.Node
	}

	new_node := func(file *extract.File) *tNode {
		node := &tNode{
			file:     file,
			children: make(map[string]*tNode),
		}

		node.addChild = func(child *tNode) bool {
			name := child.file.Name
			prev, exists := node.children[name]
			if exists && (prev.file.FileRef.GetSequenceNumber() > child.file.FileRef.GetSequenceNumber()) {
				for _, subchild := range child.children {
					node.addChild(subchild)
				}

				return false
			}

			node.children[name] = child

			return true
		}

		node.setParent = func(child *tNode) {
			parent_file, child_file := node.file, child.file

			child_file.Parent = parent_file.Id
			child_file.ParentRef = parent_file.FileRef
		}

		node.makeNode = func() *extract.Node {
			var children []*extract.Node

			for _, n := range node.children {
				children = append(children, n.makeNode())
			}

			return &extract.Node{
				File:     node.file,
				Children: children,
			}
		}

		return node
	}

	type tFileId struct {
		id  string
		seq uint16
	}

	type tMft struct {
		state *inspect.StateMft
		list  []*inspect.StateFileRecord
		files map[string]*tNode
		refs  map[data.FileRef]string
		fidxs map[data.FileIndex]*tFileId
		root  *tNode
		lost  *tNode

		getFileFromId func(id string) *tNode
		makeRoot      func() *extract.Node
	}

	new_mft := func(id string) *tMft {
		mft := &tMft{
			refs:  make(map[data.FileRef]string),
			fidxs: make(map[data.FileIndex]*tFileId),
			files: make(map[string]*tNode),
			lost: new_node(&extract.File{
				Id:   core.NewFileId(),
				Mft:  id,
				Name: "lost+found",
			}),
		}

		mft.getFileFromId = func(id string) *tNode {
			if mft.root.file.Id == id {
				return mft.root
			}

			if mft.lost.file.Id == id {
				return mft.lost
			}

			return mft.files[id]
		}

		mft.makeRoot = func() *extract.Node {
			return mft.root.makeNode()
		}

		return mft
	}

	mfts := make(map[string]*tMft)
	file_cnt := 0
	no_mft := 0

	i, cnt := 0, reader.GetCount()

	var log bytes.Buffer

	fmt.Println("Spliting states")
	for item := range stream {
		fmt.Printf("\rDone: %d %%", 100*i/cnt)
		i++

		rec := item.Record()

		if err := rec.GetError(); err != nil {
			return err
		}

		if rec.IsNull() {
			continue
		}

		rectyp := rec.GetType()

		switch rectyp {
		case inspect.STATE_RECORD_TYPE_FILE, inspect.STATE_RECORD_TYPE_MFT:
		default:
			continue
		}

		id := rec.GetMftId()
		if len(id) == 0 {
			no_mft++

			fmt.Fprintln(&log, "No MFT for record:")
			core.FprintStruct(&log, rec)
			fmt.Fprintln(&log)
		}

		mft, ok := mfts[id]
		if !ok {
			mft = new_mft(id)
			mfts[id] = mft
		}

		switch rectyp {
		case inspect.STATE_RECORD_TYPE_FILE:
			record := rec.(*inspect.StateFileRecord)

			mft.list = append(mft.list, record)
			file_cnt++

		case inspect.STATE_RECORD_TYPE_MFT:
			mft.state = rec.(*inspect.StateMft)
		}
	}

	fmt.Println("\r100%                                                      ")

	fmt.Println("Building nodes")
	i, cnt = 0, file_cnt

	for mftid, mft := range mfts {
		if mft.state == nil {
			return core.WrapError(fmt.Errorf("No MFT with ID=%s", mftid))
		}

		origin := mft.state.PartOrigin

		for _, file := range mft.list {
			fmt.Printf("\rDone: %d %%", 100*i/cnt)
			i++

			id := core.NewFileId()
			is_dir := file.IsDir()
			position := file.Position

			ref := file.Reference
			name := file.Name

			var runlist core.RunList
			var size uint64

			if !is_dir {
				attrs := file.GetAttributes(core.ATTR_DATA)
				if len(attrs) == 0 {
					continue
				}

				attr_state := attrs[0]

				attr_data, err := file.GetAttributeDesc(attr_state)
				if err != nil {
					return err
				}

				runlist = attr_state.RunList
				size = attr_data.GetSize()
			}

			f := new_node(&extract.File{
				Id:        id,
				FileRef:   ref,
				ParentRef: file.Parent,
				Mft:       mftid,
				Position:  position,
				Origin:    origin,
				Size:      size,
				Name:      name,
				RunList:   runlist,
			})

			mft.refs[ref] = id

			idx := ref.GetFileIndex()
			fid, exists := mft.fidxs[idx]

			seq := ref.GetSequenceNumber()
			if !exists || (seq > fid.seq) {
				mft.fidxs[idx] = &tFileId{
					id:  id,
					seq: seq,
				}
			}

			if file.Header.MftRecordNumber == 5 {
				mft.root = f
			} else {
				mft.files[id] = f
			}
		}
	}

	fmt.Println("\r100%                                                      ")

	fmt.Println("Making directory hierarchy")
	no_parents := 0
	i = 0

	for mftid, mft := range mfts {
		if mft.root == nil {
			seq := func() uint16 {
				for _, f := range mft.files {
					if f.file.ParentRef.GetFileIndex() == 5 {
						return f.file.ParentRef.GetSequenceNumber()
					}
				}

				return 0
			}()

			if seq == 0 {
				seq = 1
			}

			mft.root = new_node(&extract.File{
				Id:      core.NewFileId(),
				FileRef: data.MakeFileRef(seq, 5),
				Mft:     mftid,
				Name:    ".",
			})
		}

		mft.root.setParent(mft.lost)
		mft.root.addChild(mft.lost)

		for _, file := range mft.files {
			fmt.Printf("\rDone: %d %%", 100*i/cnt)
			i++

			ref := file.file.ParentRef
			parent, ok := mft.refs[ref]
			if !ok {
				idx := ref.GetFileIndex()

				if idx == mft.root.file.FileRef.GetFileIndex() {
					parent = mft.root.file.Id
				} else {
					fid, ok := mft.fidxs[idx]
					if ok {
						parent = fid.id
						file.file.ParentRef = data.MakeFileRef(fid.seq, idx)
					} else {
						fmt.Fprintf(&log, fmt.Sprintf("Parent not found for file %s", file))
						fmt.Fprintln(&log)

						no_parents++

						parent = mft.lost.file.Id
					}
				}
			}

			file.file.Parent = parent
		}
	}

	fmt.Println("\r100%                                                      ")

	fmt.Println("Completing directory hierarchy")
	dup_names := 0
	i, cnt = 0, i

	for _, mft := range mfts {
		get_parent := func(n *tNode) *tNode {
			var f *extract.File

			if n != nil {
				f = n.file
			}

			if (f == nil) || (len(f.Parent) == 0) {
				return nil
			}

			return mft.getFileFromId(f.Parent)
		}

		for _, file := range mft.files {
			fmt.Printf("\rDone: %d %%", 100*i/cnt)
			i++

			parent := get_parent(file)
			if parent != nil {
				parent = mft.lost
			}

			if !parent.addChild(file) {
				dup_names++
			}
		}
	}

	fmt.Println("\r100%                                                      ")

	fmt.Println("Making file tree")

	var roots []*extract.Node

	for _, mft := range mfts {
		roots = append(roots, mft.makeRoot())
	}

	tree := extract.Tree{
		Roots: roots,
	}

	fmt.Println()
	fmt.Println("Writing")

	writer, err := extract.MakeFileWriter(dest)
	if err != nil {
		return err
	}

	defer core.DeferedCall(writer.Close)

	writer.WriteTree(&tree, func(cur, tot int) {
		fmt.Printf("\rDone: %d %%", 100*i/cnt)
	})

	fmt.Println("\r100%                                                      ")

	fmt.Println()
	fmt.Println("File with no parent:", no_parents)
	fmt.Println("File with no MFT:   ", no_mft)
	fmt.Println("Dupplicate names:   ", dup_names)

	if verbose {
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Details:")
		fmt.Fprintln(os.Stderr, &log)
	}

	return nil
}
