package main

import (
	"encoding/hex"
	"strings"

	"essai/ntfstool/core"
	"essai/ntfstool/extract"

	"github.com/gobwas/glob"
)

type tNodePattern struct {
	tree  *extract.Tree
	ids   map[string]bool
	globs []glob.Glob
}

func (np *tNodePattern) Match(file *extract.File) bool {
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

func (np *tNodePattern) GetNodes() map[string]*extract.Node {
	res := make(map[string]*extract.Node)

	for _, node := range np.tree.Nodes {
		if np.Match(node.File) {
			res[node.File.Id] = node
		}
	}

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
				return nil, core.WrapError(err)
			}

			ids[id] = true

			continue
		}

		g, err := glob.Compile(part)
		if err != nil {
			return nil, core.WrapError(err)
		}

		globs = append(globs, g)
	}

	res := &tNodePattern{
		tree:  tree,
		ids:   ids,
		globs: globs,
	}

	return res, nil
}
