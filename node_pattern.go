package main

import (
	"encoding/hex"
	"strings"

	"github.com/gobwas/glob"

	ntfs "github.com/corebreaker/ntfstool/core"
	"github.com/corebreaker/ntfstool/extract"
)

type tNodePattern struct {
	tree  *extract.Tree
	root  *extract.Node
	ids   map[string]bool
	globs []glob.Glob
}

func (np *tNodePattern) Match(file *extract.File) bool {
	if file.Id == "" {
		return false
	}

	if np.ids[file.Id] {
		return true
	}

	name := file.Name
	path := np.tree.GetFilePath(file)

	for _, g := range np.globs {
		if g.Match(name) || g.Match(path) {
			return true
		}
	}

	return false
}

func (np *tNodePattern) getNodes(node *extract.Node, to map[string]*extract.Node) {
	for _, n := range node.Children {
		np.getNodes(n, to)
	}

	if np.Match(node.File) {
		to[node.File.Id] = node
	}
}

func (np *tNodePattern) GetNodes(from *extract.Node) map[string]*extract.Node {
	res := make(map[string]*extract.Node)

	if from == nil {
		from = np.root
	}

	np.getNodes(from, res)

	return res
}

func parseNodePattern(src string, tree *extract.Tree) (*tNodePattern, error) {
	parts := strings.Split(src, ",")
	ids := make(map[string]bool)

	var globs []glob.Glob

	for _, part := range parts {
		if part[0] == '@' {
			id := part[1:]
			if _, err := hex.DecodeString(id); err != nil {
				return nil, ntfs.WrapError(err)
			}

			ids[id] = true

			continue
		}

		g, err := glob.Compile(part)
		if err != nil {
			return nil, ntfs.WrapError(err)
		}

		globs = append(globs, g)
	}

	res := &tNodePattern{
		tree:  tree,
		ids:   ids,
		globs: globs,
		root: &extract.Node{
			File:     new(extract.File),
			Children: tree.Roots,
		},
	}

	return res, nil
}
