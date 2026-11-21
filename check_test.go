package allgen_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vasilesk/allgen"
)

const (
	directive = "//go:generate go tool allgen"
	source    = "package colors\n\n" + directive + "\nconst (\n\tRed Color = iota\n\tGreen\n\tBlue\n)\n"
)

func TestBlockVerify(t *testing.T) {
	t.Parallel()

	path := writeSource(t, source)

	tests := []struct {
		name    string
		names   []string
		wantErr string
	}{
		{name: "up to date", names: []string{"Red", "Green", "Blue"}},
		{name: "member added", names: []string{"Red", "Green"}, wantErr: "added Blue"},
		{name: "member removed", names: []string{"Red", "Green", "Blue", "Cyan"}, wantErr: "removed Cyan"},
		{name: "member renamed", names: []string{"Red", "Lime", "Blue"}, wantErr: "added Green; removed Lime"},
		{name: "members reordered", names: []string{"Green", "Red", "Blue"}, wantErr: "members reordered"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := allgen.Block{File: path, Directive: directive, Names: tt.names}.Verify()
			if tt.wantErr == "" {
				require.NoError(t, err)

				return
			}

			require.ErrorIs(t, err, allgen.ErrOutdated)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestBlockVerifyMissingSource(t *testing.T) {
	t.Parallel()

	missing := filepath.Join(t.TempDir(), "colors.go")

	err := allgen.Block{File: missing, Directive: directive, Names: []string{"Red"}}.Verify()
	require.ErrorIs(t, err, fs.ErrNotExist)
}

func TestBlockVerifyDirectiveChanged(t *testing.T) {
	t.Parallel()

	path := writeSource(t, source)

	err := allgen.Block{File: path, Directive: directive + " -name X", Names: []string{"Red"}}.Verify()
	require.Error(t, err)
	assert.NotErrorIs(t, err, fs.ErrNotExist, "a changed directive must fail the check, not skip it")
}

func TestCheck(t *testing.T) {
	t.Parallel()

	path := writeSource(t, source)

	tests := []struct {
		name        string
		block       allgen.Block
		wantFailed  bool
		wantSkipped bool
	}{
		{
			name:  "up to date",
			block: allgen.Block{File: path, Directive: directive, Names: []string{"Red", "Green", "Blue"}},
		},
		{
			name:       "outdated",
			block:      allgen.Block{File: path, Directive: directive, Names: []string{"Red"}},
			wantFailed: true,
		},
		{
			name:        "source missing",
			block:       allgen.Block{File: path + ".missing", Directive: directive, Names: []string{"Red"}},
			wantSkipped: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rec := &recorder{TB: t}
			allgen.Check(rec, tt.block)

			assert.Equal(t, tt.wantFailed, rec.failed, "failed")
			assert.Equal(t, tt.wantSkipped, rec.skipped, "skipped")
		})
	}
}

// recorder captures the outcome of a check instead of applying it to the running test.
type recorder struct {
	testing.TB

	failed  bool
	skipped bool
}

func (r *recorder) Helper() {}

func (r *recorder) Errorf(format string, args ...any) {
	r.failed = true
	r.Logf("Errorf: "+format, args...)
}

func (r *recorder) Skipf(format string, args ...any) {
	r.skipped = true
	r.Logf("Skipf: "+format, args...)
}

func writeSource(t *testing.T, src string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "colors.go")
	require.NoError(t, os.WriteFile(path, []byte(src), 0o600))

	return path
}
