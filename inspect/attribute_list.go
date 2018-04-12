package inspect

import (
	"essai/ntfstool/core"
	"sort"
)

type tAttributeList struct {
	list   []*StateAttribute
	filter map[core.AttributeType]int
}

func (al *tAttributeList) Len() int                { return len(al.list) }
func (al *tAttributeList) Less(i, j int) bool      { return al.posFor(i) < al.posFor(j) }
func (al *tAttributeList) Swap(i, j int)           { al.list[i], al.list[j] = al.list[j], al.list[i] }
func (al *tAttributeList) sort() []*StateAttribute { sort.Sort(al); return al.list }
func (al *tAttributeList) posFor(i int) int        { return al.filter[al.list[i].Header.AttributeType] }

func (al *tAttributeList) add(att *StateAttribute) {
	_, ok := al.filter[att.Header.AttributeType]
	if !ok {
		return
	}

	al.list = append(al.list, att)
}

func makeAttributeList(attr core.AttributeType, others []core.AttributeType) *tAttributeList {
	filter := make(map[core.AttributeType]int)

	filter[attr] = 0
	for i, t := range others {
		filter[t] = i + 1
	}

	return &tAttributeList{
		filter: filter,
	}
}
