package source_test

import (
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vasilesk/allgen/internal/source"
)

func TestFileBlock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		src     string
		want    source.Block
		wantErr error
	}{
		{
			name: "iota block with implicit types",
			src: `package p
//go:generate allgen
const (
	A T = iota
	B
	C
)`,
			want: source.Block{Kind: token.CONST, Names: []string{"A", "B", "C"}},
		},
		{
			name: "var block with several names per spec",
			src: `package p
//go:generate allgen
var (
	a, b = 1, 2
	c    = 3
)`,
			want: source.Block{Kind: token.VAR, Names: []string{"a", "b", "c"}},
		},
		{
			name: "single declaration without parentheses",
			src: `package p
//go:generate allgen
var A = 1`,
			want: source.Block{Kind: token.VAR, Names: []string{"A"}},
		},
		{
			name: "blank identifiers and skip markers",
			src: `package p
//go:generate allgen
const (
	_ T = iota
	A
	//allgen:skip
	B
	C //allgen:skip count of values
	D // not skipped: //allgen:skip
)`,
			want: source.Block{Kind: token.CONST, Names: []string{"A", "D"}},
		},
		{
			name: "comments between directive and block",
			src: `package p
//go:generate allgen

// Colors.
const (
	A = 1
)`,
			want: source.Block{Kind: token.CONST, Names: []string{"A"}},
		},
		{
			name: "type declaration below directive",
			src: `package p
//go:generate allgen
type T int`,
			wantErr: source.ErrNoBlock,
		},
		{
			name: "directive inside a function",
			src: `package p
func f() {
//go:generate allgen
}
const A = 1`,
			wantErr: source.ErrNoBlock,
		},
		{
			name: "nothing below directive",
			src: `package p
//go:generate allgen`,
			wantErr: source.ErrNoBlock,
		},
		{
			name: "every member skipped",
			src: `package p
//go:generate allgen
const (
	_ = iota
	A //allgen:skip
)`,
			wantErr: source.ErrEmptyBlock,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			file := parse(t, tt.src)

			got, err := file.Block(2)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFileDirectiveAt(t *testing.T) {
	t.Parallel()

	file := parse(t, `package p
//go:generate  allgen -name X  
// go:generate not a directive
//go:generated not a directive either
const A = 1`)

	got, err := file.DirectiveAt(2)
	require.NoError(t, err)
	assert.Equal(t, "//go:generate  allgen -name X", got, "surrounding white space is trimmed")

	for _, line := range []int{1, 3, 4, 5} {
		_, err := file.DirectiveAt(line)
		require.ErrorIs(t, err, source.ErrNoDirective, "line %d", line)
	}
}

func TestFileDirectiveLine(t *testing.T) {
	t.Parallel()

	file := parse(t, `package p
//go:generate allgen -name X
const A = 1
//go:generate allgen
const B = 1
//go:generate allgen
const C = 1`)

	line, err := file.DirectiveLine("  //go:generate allgen -name X ")
	require.NoError(t, err)
	assert.Equal(t, 2, line)

	_, err = file.DirectiveLine("//go:generate allgen")
	require.ErrorIs(t, err, source.ErrAmbiguousDirective)

	_, err = file.DirectiveLine("//go:generate allgen -name Y")
	require.ErrorIs(t, err, source.ErrNoDirective)
}

func TestFileBuildConstraint(t *testing.T) {
	t.Parallel()

	constrained := parse(t, "// Copyright.\n\n//go:build linux && !cgo\n\npackage p\n\n//go:build ignored\nconst A = 1\n")
	got, ok := constrained.BuildConstraint()
	assert.True(t, ok)
	assert.Equal(t, "//go:build linux && !cgo", got)

	_, ok = parse(t, "package p\n\n//go:build ignored\nconst A = 1\n").BuildConstraint()
	assert.False(t, ok, "a //go:build line after the package clause is not a constraint")
}

func TestParse(t *testing.T) {
	t.Parallel()

	file := parse(t, "package colors\n")
	assert.Equal(t, "colors", file.Package())

	_, err := source.Parse(filepath.Join(t.TempDir(), "missing.go"))
	require.ErrorIs(t, err, fs.ErrNotExist)

	_, err = source.Parse(writeFile(t, "package"))
	require.Error(t, err)
}

func parse(t *testing.T, src string) source.File {
	t.Helper()

	file, err := source.Parse(writeFile(t, src))
	require.NoError(t, err)

	return file
}

func writeFile(t *testing.T, src string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "file.go")
	require.NoError(t, os.WriteFile(path, []byte(src), 0o600))

	return path
}
