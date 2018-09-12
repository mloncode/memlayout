package memlayout

import (
	"bytes"
	"fmt"
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
	field     *types.Var
}

func (f Field) String() string {
	if f.IsPadding {
		return fmt.Sprintf("*%s: %d-%d (size %d, align %d)*",
			"padding", f.Start, f.End, f.Size, f.Align)
	}
	return fmt.Sprintf("%s %s: %d-%d (size %d, align %d)",
		f.Name, f.Type, f.Start, f.End, f.Size, f.Align)
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

var gcSizes = gcsizes.ForArch(build.Default.GOARCH)

func sizes(typ *types.Struct, base int64) (out []Field) {
	n := typ.NumFields()
	var fields []*types.Var
	for i := 0; i < n; i++ {
		fields = append(fields, typ.Field(i))
	}
	offsets := gcSizes.Offsetsof(fields)
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

		size := gcSizes.Sizeof(field.Type())
		if typ2, ok := field.Type().Underlying().(*types.Struct); ok && typ2.NumFields() != 0 {
			out = append(out, Field{
				Name:     field.Name(),
				Type:     field.Type().String(),
				Start:    offsets[i],
				End:      offsets[i] + size,
				Size:     size,
				Align:    gcSizes.Alignof(field.Type()),
				Children: sizes(typ2, pos),
				field:    field,
			})
		} else {
			out = append(out, Field{
				Name:  field.Name(),
				Type:  field.Type().String(),
				Start: offsets[i],
				End:   offsets[i] + size,
				Size:  size,
				Align: gcSizes.Alignof(field.Type()),
				field: field,
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
	pad := gcSizes.Sizeof(typ) - field.End
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

	var typeFields = make([]*types.Var, len(fields))
	for i, f := range fields {
		if !f.IsPadding {
			typeFields[i] = f.field
		}
	}

	fieldsWithPadding := Fields(types.NewStruct(typeFields, nil))

	return Struct{
		Name:   s.Name,
		Pos:    s.Pos,
		Fields: fieldsWithPadding,
	}
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
