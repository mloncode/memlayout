package memlayout

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const testSourceHead = `
package main

func Foo() int { return 5 }

type Bar struct {
	A int
	B string
}

type Baz interface {
	Bazer() string
}

type Qux struct {
	A int
	B bool
	F *string
}

type Mux struct {
	A int
}
`

func TestChangedStructs(t *testing.T) {
	require := require.New(t)

	base, err := StructsFromSource("test.go", []byte(testSource))
	require.NoError(err)
	head, err := StructsFromSource("test.go", []byte(testSourceHead))
	require.NoError(err)

	changed := ChangedStructs([]byte(testSource), []byte(testSourceHead), base, head)

	expected := []ChangedStruct{
		{
			Head: Struct{
				Name: "Mux",
				Pos:  Pos{Start: 21, End: 23},
				Fields: []Field{
					f("A", "int", 0, 8, 8, 8, false),
				},
			},
		},
		{
			Base: &Struct{
				Name: "Qux",
				Pos:  Pos{Start: 15, End: 23},
				Fields: []Field{
					f("A", "int", 0, 8, 8, 8, false),
					f("B", "bool", 8, 9, 1, 1, false),
					f("", "", 9, 16, 7, 0, true),
					f("C", "struct{D int64; E []byte}", 16, 48, 32, 8, false,
						f("D", "int64", 16, 24, 8, 8, false),
						f("E", "[]byte", 24, 48, 24, 8, false),
					),
					f("F", "*string", 48, 56, 8, 8, false),
				},
			},
			Head: Struct{
				Name: "Qux",
				Pos:  Pos{Start: 15, End: 19},
				Fields: []Field{
					f("A", "int", 0, 8, 8, 8, false),
					f("B", "bool", 8, 9, 1, 1, false),
					f("", "", 9, 16, 7, 0, true),
					f("F", "*string", 16, 24, 8, 8, false),
				},
			},
		},
	}

	require.Len(changed, len(expected))
	for i := range expected {
		if expected[i].Base == nil {
			require.Nil(changed[i].Base)
		} else {
			require.NotNil(changed[i].Base)
			structsEqual(t, *expected[i].Base, *changed[i].Base)
		}

		structsEqual(t, expected[i].Head, changed[i].Head)
	}
}
