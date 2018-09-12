package memlayout

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"

	"golang.org/x/tools/go/loader"
)

func StructsFromFile(filename string, content []byte) ([]Struct, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, content, 0)
	if err != nil {
		return nil, fmt.Errorf("unable to parse file %s: %s", filename, err)
	}

	files, err := filepath.Glob(filepath.Join(filepath.Dir(filename), "*.go"))
	if err != nil {
		return nil, err
	}

	conf := loader.Config{
		Build: &build.Default,
	}

	if _, err = conf.FromArgs(files, true); err != nil {
		return nil, err
	}

	lprog, err := conf.Load()
	if err != nil {
		return nil, err
	}

	scope := lprog.InitialPackages()[0].Pkg.Scope()

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
