package memlayout

import (
	"bytes"
	"go/ast"
	"go/build"
	"go/printer"
	"go/token"
	"go/types"
	"sort"
	"strings"

	"honnef.co/go/tools/gcsizes"
)

// Pos has the start and end line of a struct.
type Pos struct {
	Start int
	End   int
}

// Struct represents a struct with its fields.
type Struct struct {
	Name string
	Pos
	Fields []Field
}

// Field represents a struct field.
type Field struct {
	Name      string
	Type      string
	Start     int64
	End       int64
	Size      int64
	Align     int64
	IsPadding bool
	Children  []Field
}

func structType(fields []Field) *ast.StructType {
	var fs []*ast.Field
	for _, f := range fields {
		if !f.IsPadding {
			var typ ast.Expr
			if len(f.Children) > 0 {
				typ = structType(f.Children)
			} else {
				typ = ast.NewIdent(f.Type)
			}

			fs = append(fs, &ast.Field{
				Names: []*ast.Ident{ast.NewIdent(f.Name)},
				Type:  typ,
			})

		}
	}

	return &ast.StructType{Fields: &ast.FieldList{List: fs}}
}

func (s Struct) String() string {
	fset := token.NewFileSet()
	var buf bytes.Buffer
	_ = printer.Fprint(&buf, fset, &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{&ast.TypeSpec{
			Name: ast.NewIdent(s.Name),
			Type: structType(s.Fields),
		}},
	})
	return buf.String()
}

// Size returns the total size of the struct.
func (s Struct) Size() int64 {
	var total int64

	for _, f := range s.Fields {
		total += f.Size
	}

	return total
}

// Padding returns the total size of padding in the struct.
func (s Struct) Padding() int64 {
	var total int64

	for _, f := range s.Fields {
		if f.IsPadding {
			total += f.Size
		}
	}

	return total
}

// Fields returns the fields of a struct with their memory layout info,
func Fields(typ *types.Struct) []Field {
	return sizes(typ, 0)
}

func sizes(typ *types.Struct, base int64) (out []Field) {
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
			out = append(out, Field{
				Name:     field.Name(),
				Type:     field.Type().String(),
				Start:    offsets[i],
				End:      offsets[i] + size,
				Size:     size,
				Align:    s.Alignof(field.Type()),
				Children: sizes(typ2, pos),
			})
		} else {
			out = append(out, Field{
				Name:  field.Name(),
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

// HasBetterAlignment returns whether the head has better alignment than
// the base.
func HasBetterAlignment(base, head Struct) bool {
	return base.Padding() > head.Padding()
}

// Optimize optimizes the struct for a better aligned memory layout.
func Optimize(s Struct) Struct {
	var fields []Field
	for _, f := range s.Fields {
		if !f.IsPadding {
			fields = append(fields, f)
		}
	}

	sortFields(fields)
	return Struct{Name: s.Name, Pos: s.Pos, Fields: addPadding(fields)}
}

func sortFields(fields []Field) {
	sort.Stable(byAlignSizeAndName(fields))
	for _, f := range fields {
		if len(f.Children) > 0 {
			sortFields(f.Children)
		}
	}
}

type byAlignSizeAndName []Field

func (s byAlignSizeAndName) Len() int      { return len(s) }
func (s byAlignSizeAndName) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s byAlignSizeAndName) Less(i, j int) bool {
	if s[i].Size == 0 && s[j].Size != 0 {
		return true
	}

	if s[j].Size == 0 && s[i].Size != 0 {
		return false
	}

	if s[i].Align != s[j].Align {
		return s[i].Align > s[j].Align
	}

	if s[i].Size != s[j].Size {
		return s[i].Size > s[j].Size
	}

	return strings.Compare(s[i].Name, s[j].Name) < 0
}

func addPadding(fs []Field) []Field {
	var result []Field
	var total int64
	for _, f := range fs {
		if len(f.Children) > 0 {
			f.Children = addPadding(f.Children)
		}

		f.Size = Struct{Fields: f.Children}.Size()
		f.Start = total
		f.End = f.Start + f.Size
		result = append(result, f)
		total += f.Size

		if total%8 != 0 {
			padding := 8*(total/8+1) - total
			result = append(result, Field{
				Start:     total,
				End:       total + padding,
				Size:      padding,
				IsPadding: true,
			})
		}
	}

	if total%8 != 0 {
		padding := 8*(total/8+1) - total
		result = append(result, Field{
			Start:     total,
			End:       total + padding,
			Size:      padding,
			IsPadding: true,
		})
	}

	return result
}
