package memlayout

import (
	"io/ioutil"
	"os"
	"path/filepath"
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

	tmp, err := ioutil.TempDir(os.TempDir(), "tmp-memlayout")
	require.NoError(err)
	defer func() {
		require.NoError(os.RemoveAll(tmp))
	}()

	path := filepath.Join(tmp, "test.go")
	require.NoError(ioutil.WriteFile(path, []byte(testSourceHead), 0755))

	structs, err := StructsFromFile(path, []byte(testSourceHead))
	require.NoError(err)

	changed := ChangedStructs([]byte(testSource), []byte(testSourceHead), structs)

	expected := []Struct{
		{
			Name: "Mux",
			Pos:  Pos{Start: 21, End: 23},
			Fields: []Field{
				f("A", "int", 0, 8, 8, 8, false),
			},
		},
		{
			Name: "Qux",
			Pos:  Pos{Start: 15, End: 19},
			Fields: []Field{
				f("A", "int", 0, 8, 8, 8, false),
				f("B", "bool", 8, 9, 1, 1, false),
				f("", "", 9, 16, 7, 0, true),
				f("F", "*string", 16, 24, 8, 8, false),
			},
		},
	}

	require.Len(changed, len(expected))
	for i := range expected {
		structsEqual(t, expected[i], changed[i])
	}
}
