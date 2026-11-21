// Package source finds the const or var block a go:generate directive is placed above
// and lists the identifiers the block declares.
//
// The package works on syntax alone, so the result does not depend on whether the
// package containing the file type-checks.
package source

import (
	"errors"
	"fmt"
	"go/ast"
	"go/build/constraint"
	"go/parser"
	"go/token"
	"os"
	"slices"
	"strings"
)

// File is a parsed Go source file.
type File struct {
	fset *token.FileSet
	ast  *ast.File
}

// Parse reads and parses the Go file at path. A missing file yields an error
// wrapping [fs.ErrNotExist].
func Parse(path string) (File, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return File{}, fmt.Errorf("reading file: %w", err)
	}

	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, path, src, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return File{}, fmt.Errorf("parsing file: %w", err)
	}

	return File{fset: fset, ast: file}, nil
}

// Package returns the package name from the package clause.
func (f File) Package() string {
	return f.ast.Name.Name
}

// BuildConstraint returns the //go:build line of the file, if it has one.
func (f File) BuildConstraint() (string, bool) {
	header := filter(f.ast.Comments, func(group *ast.CommentGroup) bool { return group.Pos() < f.ast.Package })
	for _, group := range header {
		idx := slices.IndexFunc(group.List, func(c *ast.Comment) bool { return constraint.IsGoBuild(c.Text) })
		if idx >= 0 {
			return group.List[idx].Text, true
		}
	}

	return "", false
}

// DirectiveAt returns the text of the go:generate directive at the 1-based line.
func (f File) DirectiveAt(line int) (string, error) {
	matches := f.comments(func(c *ast.Comment) bool {
		return f.line(c.Pos()) == line && isDirective(c.Text)
	})
	if len(matches) == 0 {
		return "", fmt.Errorf("%w at line %d", ErrNoDirective, line)
	}

	return strings.TrimSpace(matches[0].Text), nil
}

// DirectiveLine returns the 1-based line of the only directive with the given text.
// Surrounding white space is ignored in the comparison.
func (f File) DirectiveLine(directive string) (int, error) {
	text := strings.TrimSpace(directive)
	matches := f.comments(func(c *ast.Comment) bool { return strings.TrimSpace(c.Text) == text })

	switch len(matches) {
	case 0:
		return 0, fmt.Errorf("%w: %q", ErrNoDirective, text)
	case 1:
		return f.line(matches[0].Pos()), nil
	default:
		return 0, fmt.Errorf("%w: %q", ErrAmbiguousDirective, text)
	}
}

// Block returns the top-level const or var declaration that follows the directive at
// the 1-based line, with nothing but comments between them.
func (f File) Block(line int) (Block, error) {
	idx := slices.IndexFunc(f.ast.Decls, func(d ast.Decl) bool { return f.line(d.Pos()) > line })
	if idx < 0 {
		return Block{}, fmt.Errorf("%w at line %d", ErrNoBlock, line)
	}

	if idx > 0 && f.line(f.ast.Decls[idx-1].End()) >= line {
		return Block{}, fmt.Errorf("%w: directive at line %d is inside a declaration", ErrNoBlock, line)
	}

	decl, ok := f.ast.Decls[idx].(*ast.GenDecl)
	if !ok || (decl.Tok != token.CONST && decl.Tok != token.VAR) {
		return Block{}, fmt.Errorf("%w: line %d declares something else", ErrNoBlock, f.line(f.ast.Decls[idx].Pos()))
	}

	names := members(decl)
	if len(names) == 0 {
		return Block{}, fmt.Errorf("%w at line %d", ErrEmptyBlock, f.line(decl.Pos()))
	}

	return Block{Kind: decl.Tok, Names: names}, nil
}

// Block is a top-level const or var declaration and the identifiers it contributes to a list.
type Block struct {
	// Kind is token.CONST or token.VAR.
	Kind token.Token
	// Names are the declared identifiers in declaration order, without blank identifiers
	// and specs marked with [SkipMarker]. Names is never empty.
	Names []string
}

// SkipMarker excludes the identifiers of a spec from the members of its block when placed
// in the doc or line comment of the spec.
const SkipMarker = "//allgen:skip"

var (
	// ErrNoDirective means a file has no go:generate directive at the requested place.
	ErrNoDirective = errors.New("directive not found")
	// ErrAmbiguousDirective means a file contains the same directive text more than once,
	// so the text does not identify a block.
	ErrAmbiguousDirective = errors.New("directive is not unique in file")
	// ErrNoBlock means a directive is not followed by a top-level const or var declaration.
	ErrNoBlock = errors.New("no const or var declaration below directive")
	// ErrEmptyBlock means a block declares no identifiers once blank and skipped ones are removed.
	ErrEmptyBlock = errors.New("block has no members")
)

func (f File) comments(match func(*ast.Comment) bool) []*ast.Comment {
	all := slices.Concat(mapSlice(f.ast.Comments, func(g *ast.CommentGroup) []*ast.Comment { return g.List })...)

	return filter(all, match)
}

func (f File) line(pos token.Pos) int {
	return f.fset.Position(pos).Line
}

func members(decl *ast.GenDecl) []string {
	specs := filter(valueSpecs(decl), func(spec *ast.ValueSpec) bool { return !skipped(spec) })
	idents := slices.Concat(mapSlice(specs, func(spec *ast.ValueSpec) []*ast.Ident { return spec.Names })...)
	named := filter(idents, func(id *ast.Ident) bool { return id.Name != "_" })

	return mapSlice(named, func(id *ast.Ident) string { return id.Name })
}

func valueSpecs(decl *ast.GenDecl) []*ast.ValueSpec {
	specs := make([]*ast.ValueSpec, 0, len(decl.Specs))
	for _, spec := range decl.Specs {
		if vs, ok := spec.(*ast.ValueSpec); ok {
			specs = append(specs, vs)
		}
	}

	return specs
}

func skipped(spec *ast.ValueSpec) bool {
	return hasSkipMarker(spec.Doc) || hasSkipMarker(spec.Comment)
}

func hasSkipMarker(group *ast.CommentGroup) bool {
	return group != nil && slices.ContainsFunc(group.List, func(c *ast.Comment) bool {
		return c.Text == SkipMarker || strings.HasPrefix(c.Text, SkipMarker+" ")
	})
}

func isDirective(text string) bool {
	const directivePrefix = "//go:generate"

	return text == directivePrefix || strings.HasPrefix(text, directivePrefix+" ")
}

func filter[T any](items []T, keep func(T) bool) []T {
	return slices.DeleteFunc(slices.Clone(items), func(item T) bool { return !keep(item) })
}

func mapSlice[T, R any](items []T, fn func(T) R) []R {
	out := make([]R, 0, len(items))
	for _, item := range items {
		out = append(out, fn(item))
	}

	return out
}
