package memlayout

import (
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// ChangedStruct contains a struct that was changed in a diff with its previous
// version from the base revision.
type ChangedStruct struct {
	// Base is the struct version in the base revision. May not be present.
	Base *Struct
	// Head is the struct version in the head revision.
	Head Struct
}

// ChangedStructs returns all the changed structs in the diff.
func ChangedStructs(
	base, head []byte,
	baseStructs, headStructs []Struct,
) []ChangedStruct {
	lines := changedLines(base, head)

	var baseStructsByName = make(map[string]*Struct)
	for _, s := range baseStructs {
		baseStructsByName[s.Name] = &s
	}

	var result []ChangedStruct
	for _, s := range headStructs {
		for _, l := range lines {
			if l >= s.Start && l <= s.End {
				base := baseStructsByName[s.Name]
				result = append(result, ChangedStruct{
					Base: base,
					Head: s,
				})
				break
			}
		}
	}

	return result
}

func changedLines(base, head []byte) []int {
	dmp := diffmatchpatch.New()
	diff := dmp.DiffMain(string(base), string(head), false)

	var acc int
	var result []int

	for _, d := range diff {
		lines := len(strings.Split(d.Text, "\n"))
		if d.Type == diffmatchpatch.DiffDelete {
			result = append(result, acc)
			continue
		}

		if d.Type == diffmatchpatch.DiffInsert {
			for i := 1; i <= lines; i++ {
				result = append(result, acc+i)
			}
		}

		acc += lines
	}

	return result
}
