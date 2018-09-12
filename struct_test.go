package memlayout

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const unoptimized = `
package foo

type Foo struct {
	A string
	B bool
	C int64
	D bool
	E uint16
}
`

func TestOptimize(t *testing.T) {
	require := require.New(t)

	structs, err := StructsFromSource("foo.go", []byte(unoptimized))
	require.NoError(err)
	require.Len(structs, 1)

	optimized := Optimize(structs[0])

	expected := Struct{
		Name: "Foo",
		Fields: []Field{
			{Name: "A", Type: "string", Size: 16, Start: 0, End: 16, Align: 8},
			{Name: "C", Type: "int64", Size: 8, Start: 16, End: 24, Align: 8},
			{Name: "E", Type: "uint16", Size: 2, Start: 24, End: 26, Align: 2},
			{Name: "B", Type: "bool", Size: 1, Start: 26, End: 27, Align: 1},
			{Name: "D", Type: "bool", Size: 1, Start: 27, End: 28, Align: 1},
			{Size: 4, IsPadding: true, Start: 28, End: 32}, // padding
		},
	}

	structsEqual(t, expected, optimized)
}

func structsEqual(t *testing.T, expected, result Struct) {
	t.Helper()
	require := require.New(t)

	require.Equal(expected.Name, result.Name)
	require.Len(result.Fields, len(expected.Fields))
	for i := range expected.Fields {
		fieldsEqual(t, expected.Fields[i], result.Fields[i])
	}
}

func fieldsEqual(t *testing.T, e, r Field) {
	t.Helper()

	if len(e.Children) > 0 {
		require.Len(t, r.Children, len(e.Children))
		for i := range e.Children {
			e.Children[i].field = nil
			r.Children[i].field = nil
			fieldsEqual(t, e.Children[i], r.Children[i])
		}
	}

	e.field = nil
	r.field = nil

	require.Equal(t, e, r)
}
