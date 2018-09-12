package memlayout

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOptimize(t *testing.T) {
	require := require.New(t)

	optimized := Optimize(Struct{
		Name: "Foo",
		Fields: []Field{
			{Name: "A", Type: "string", Size: 16, Start: 0, End: 16, Align: 8},
			{Name: "B", Type: "bool", Size: 1, Start: 16, End: 17, Align: 1},
			{Start: 17, End: 24, Size: 7, IsPadding: true}, // padding
			{Name: "C", Type: "int64", Size: 8, Start: 24, End: 32, Align: 8},
			{Name: "D", Type: "bool", Size: 1, Start: 32, End: 33, Align: 1},
			{Start: 33, End: 40, Size: 7, IsPadding: true}, // padding
			{Name: "E", Type: "uint16", Size: 2, Start: 40, End: 42},
			{Start: 42, End: 48, Size: 6, IsPadding: true}, // padding
		},
	})

	expected := Struct{
		Name: "Foo",
		Fields: []Field{
			{Name: "A", Type: "string", Size: 16, Start: 0, End: 16, Align: 8},
			{Name: "C", Type: "int64", Size: 8, Start: 16, End: 24, Align: 8},
			{Name: "E", Type: "uint16", Size: 2, Start: 24, End: 26},
			{Name: "B", Type: "bool", Size: 1, Start: 26, End: 27, Align: 1},
			{Name: "D", Type: "bool", Size: 1, Start: 27, End: 28, Align: 1},
			{Size: 4, IsPadding: true, Start: 28, End: 32}, // padding
		},
	}

	require.Equal(expected, optimized)
}
