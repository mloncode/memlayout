package memlayout

import (
	"context"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"go/types"
	"io/ioutil"
	"log"
	"path/filepath"

	"github.com/src-d/lookout"
	"golang.org/x/tools/go/loader"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"honnef.co/go/tools/gcsizes"
)

type Analyzer struct {
	Version          string
	DataClient       *lookout.DataClient
	RequestFilesPush bool
}

var _ lookout.AnalyzerServer = &Analyzer{}

func (a *Analyzer) NotifyReviewEvent(ctx context.Context, e *lookout.ReviewEvent) (*lookout.EventResponse, error) {
	fmt.Printf("REVIEW: %+v\n", e)
	root, err := clone(e)
	if err != nil {
		return nil, err
	}
	fmt.Println("root", root)

	changes, err := a.DataClient.GetChanges(ctx, &lookout.ChangesRequest{
		Base:         &e.CommitRevision.Base,
		Head:         &e.CommitRevision.Head,
		WantContents: true,
		WantUAST:     false,
	})
	if err != nil {
		return nil, err
	}

	var comments []*lookout.Comment
	for changes.Next() {
		change := changes.Change()
		fmt.Printf("\t%s\n", change.Head.Path)

		filename := filepath.Join(root, change.Head.Path)
		structnames := getStructNames(filename)
		for _, sname := range structnames {
			txt, e := isOK(root, filename, sname)
			if e != nil {
				continue
			}

			comments = append(comments, &lookout.Comment{
				File: change.Head.Path,
				Line: 0,
				Text: txt,
			})
		}
	}

	if err := changes.Err(); err != nil {
		return nil, err
	}

	resp := &lookout.EventResponse{AnalyzerVersion: a.Version, Comments: comments}
	return resp, nil
}

func (a *Analyzer) NotifyPushEvent(ctx context.Context, e *lookout.PushEvent) (*lookout.EventResponse, error) {
	resp := &lookout.EventResponse{AnalyzerVersion: a.Version}

	return resp, nil
}

func clone(review *lookout.ReviewEvent) (string, error) {
	tmp, err := ioutil.TempDir("/tmp", "memlayout-")
	if err != nil {
		return "", err
	}

	r, err := git.PlainClone(tmp, false, &git.CloneOptions{
		URL: review.Head.InternalRepositoryURL,
	})
	if err != nil {
		return "", err
	}

	w, err := r.Worktree()
	if err != nil {
		return "", err
	}

	err = w.Checkout(&git.CheckoutOptions{
		Hash: plumbing.NewHash(review.Head.Hash),
	})
	if err != nil {
		return "", err
	}

	return tmp, nil
}

func getStructNames(filename string) []string {
	fs := token.NewFileSet()

	f, err := parser.ParseFile(fs, filename, nil, parser.AllErrors)
	if err != nil {
		log.Printf("could not parse %s: %v", filename, err)
		return nil
	}
	v := newVisitor(f)
	ast.Walk(v, f)
	return v.structnames
}

func isOK(dir, filename, structname string) (string, error) {
	conf := loader.Config{
		Build: &build.Default,
	}

	arr, err := conf.FromArgs([]string{filename}, true)
	if err != nil {
		log.Println(err)
		return "", err
	}
	log.Println(arr)

	lprog, err := conf.Load()
	if err != nil {
		log.Println(err)
		return "", err
	}
	log.Println(lprog.InitialPackages())

	var typ types.Type
	obj := lprog.InitialPackages()[0].Pkg.Scope().Lookup(structname)
	if obj == nil {
		log.Println("couldn't find type", structname)
		return "", err
	}
	typ = obj.Type()
	st, ok := typ.Underlying().(*types.Struct)
	if !ok {
		log.Println("identifier is not a struct type")
		return "", err
	}

	var comment string
	fields := sizes(st, typ.(*types.Named).Obj().Name(), 0, nil)
	for _, f := range fields {
		comment += f.String() + "\n"
	}

	return comment, nil
}

func sizes(typ *types.Struct, prefix string, base int64, out []Field) []Field {
	s := gcsizes.ForArch(build.Default.GOARCH)
	n := typ.NumFields()
	var fields []*types.Var
	for i := 0; i < n; i++ {
		fields = append(fields, typ.Field(i))
	}
	offsets := s.Offsetsof(fields)
	for i := range offsets {
		offsets[i] += base
	}

	pos := base
	for i, field := range fields {
		if offsets[i] > pos {
			padding := offsets[i] - pos
			out = append(out, Field{
				IsPadding: true,
				Start:     pos,
				End:       pos + padding,
				Size:      padding,
			})
			pos += padding
		}
		size := s.Sizeof(field.Type())
		if typ2, ok := field.Type().Underlying().(*types.Struct); ok && typ2.NumFields() != 0 {
			out = sizes(typ2, prefix+"."+field.Name(), pos, out)
		} else {
			out = append(out, Field{
				Name:  prefix + "." + field.Name(),
				Type:  field.Type().String(),
				Start: offsets[i],
				End:   offsets[i] + size,
				Size:  size,
				Align: s.Alignof(field.Type()),
			})
		}
		pos += size
	}

	if len(out) == 0 {
		return out
	}
	field := &out[len(out)-1]
	if field.Size == 0 {
		field.Size = 1
		field.End++
	}
	pad := s.Sizeof(typ) - field.End
	if pad > 0 {
		out = append(out, Field{
			IsPadding: true,
			Start:     field.End,
			End:       field.End + pad,
			Size:      pad,
		})
	}

	return out
}

type visitor struct {
	structnames []string
}

func newVisitor(f *ast.File) visitor {
	return visitor{}
}

func (v visitor) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		return nil
	}

	switch d := n.(type) {
	case *ast.GenDecl:
		for _, spec := range d.Specs {
			if t, ok := spec.(*ast.TypeSpec); ok {
				fmt.Println(t.Name)
				v.structnames = append(v.structnames, t.Name.String())
			}
		}
	}

	return v
}

type Field struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Start     int64  `json:"start"`
	End       int64  `json:"end"`
	Size      int64  `json:"size"`
	Align     int64  `json:"align"`
	IsPadding bool   `json:"is_padding"`
}

func (f Field) String() string {
	if f.IsPadding {
		return fmt.Sprintf("%s: %d-%d (size %d, align %d)",
			"padding", f.Start, f.End, f.Size, f.Align)
	}
	return fmt.Sprintf("%s %s: %d-%d (size %d, align %d)",
		f.Name, f.Type, f.Start, f.End, f.Size, f.Align)
}
