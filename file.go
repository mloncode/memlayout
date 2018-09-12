package memlayout

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
)

// ErrNoTypeCheck is returned when the source cannot be type checked.
var ErrNoTypeCheck = errors.New("can't typecheck file")

// StructsFromSource retrieves all structs with their fields and their memory
// structure from the content of a file.
func StructsFromSource(filename string, content []byte) ([]Struct, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, content, 0)
	if err != nil {
		return nil, fmt.Errorf("unable to parse file %s: %s", filename, err)
	}

	config := types.Config{
		IgnoreFuncBodies:         true,
		FakeImportC:              true,
		DisableUnusedImportCheck: true,
	}

	pkg, err := config.Check(filename, fset, []*ast.File{f}, nil)
	if err != nil {
		return nil, ErrNoTypeCheck
	}

	scope := pkg.Scope()

	var result []Struct
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		s, ok := structFromObject(obj)
		if !ok {
			continue
		}

		result = append(result, Struct{
			Name:   obj.Name(),
			Pos:    posOf(fset, f, obj.Name()),
			Fields: Fields(s),
		})
	}

	return result, nil
}

func structFromObject(obj types.Object) (*types.Struct, bool) {
	named, ok := obj.Type().(*types.Named)
	if !ok {
		return nil, false
	}

	s, ok := named.Underlying().(*types.Struct)
	return s, ok
}

func posOf(fset *token.FileSet, f *ast.File, name string) Pos {
	for _, d := range f.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok {
			continue
		}

		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok || ts.Name.Name != name {
				continue
			}

			fi := fset.File(ts.Pos())

			return Pos{
				Start: fi.Line(ts.Pos()),
				End:   fi.Line(ts.End()),
			}
		}
	}

	return Pos{}
}
