package memlayout

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const testSource = `
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
	C struct {
		D int64
		E []byte
	}
	F *string
}
`

func TestStructsFromSource(t *testing.T) {
	require := require.New(t)

	structs, err := StructsFromSource("test.go", []byte(testSource))
	require.NoError(err)

	expected := []Struct{
		{"Bar", Pos{6, 9}, []Field{
			f("A", "int", 0, 8, 8, 8, false),
			f("B", "string", 8, 24, 16, 8, false),
		}},
		{"Qux", Pos{15, 23}, []Field{
			f("A", "int", 0, 8, 8, 8, false),
			f("B", "bool", 8, 9, 1, 1, false),
			f("", "", 9, 16, 7, 0, true),
			f("C", "struct{D int64; E []byte}", 16, 48, 32, 8, false,
				f("D", "int64", 16, 24, 8, 8, false),
				f("E", "[]byte", 24, 48, 24, 8, false),
			),
			f("F", "*string", 48, 56, 8, 8, false),
		}},
	}

	require.Len(structs, len(expected))
	for i := range expected {
		structsEqual(t, expected[i], structs[i])
	}
}

func f(
	name, typ string,
	start, end, size, align int64,
	padding bool,
	children ...Field,
) Field {
	return Field{
		Name:      name,
		Type:      typ,
		Start:     start,
		End:       end,
		Size:      size,
		Align:     align,
		IsPadding: padding,
		Children:  children,
	}
}
