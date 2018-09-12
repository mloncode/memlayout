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
	structs []Struct,
) []Struct {
	lines := changedLines(base, head)

	var result []Struct
	for _, s := range structs {
		for _, l := range lines {
			if l >= s.Start && l <= s.End {
				result = append(result, s)
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
