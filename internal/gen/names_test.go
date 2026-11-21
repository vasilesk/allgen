package gen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlural(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"Color":    "Colors",
		"Status":   "Statuses",
		"Box":      "Boxes",
		"Match":    "Matches",
		"Hash":     "Hashes",
		"Category": "Categories",
		"Key":      "Keys",
		"ID":       "IDs",
		"OS":       "OSs",
		"y":        "ys",
	}

	for word, want := range tests {
		assert.Equal(t, want, plural(word), word)
	}
}

func TestSnakeCase(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"AllColors":    "all_colors",
		"allColors":    "all_colors",
		"AllHTTPCodes": "all_http_codes",
		"AllIDs":       "all_ids",
		"AllV2Colors":  "all_v2_colors",
		"HTTP":         "http",
		"X":            "x",
	}

	for name, want := range tests {
		assert.Equal(t, want, snakeCase(name), name)
	}
}

func TestNewFuncName(t *testing.T) {
	t.Parallel()

	for _, valid := range []string{"AllColors", "allColors", "_all", "Цвета"} {
		got, err := NewFuncName(valid)
		require.NoError(t, err, valid)
		assert.Equal(t, FuncName(valid), got)
	}

	for _, invalid := range []string{"", "_", "init", "func", "1all", "all-colors", "a.b"} {
		_, err := NewFuncName(invalid)
		require.ErrorIs(t, err, ErrInvalidFuncName, invalid)
	}
}

func TestNewFileName(t *testing.T) {
	t.Parallel()

	got, err := NewFileName("colors_gen.go")
	require.NoError(t, err)
	assert.Equal(t, FileName("colors_gen.go"), got)
	assert.Equal(t, "colors_gen_test.go", got.TestFile())

	for _, invalid := range []string{"", "colors", "colors_test.go", "_colors.go", ".colors.go", "sub/colors.go"} {
		_, err := NewFileName(invalid)
		require.ErrorIs(t, err, ErrInvalidFileName, invalid)
	}
}

func TestDefaultFileName(t *testing.T) {
	t.Parallel()

	got, err := defaultFileName("AllHTTPCodes")
	require.NoError(t, err)
	assert.Equal(t, FileName("all_http_codes_gen.go"), got)

	_, err = defaultFileName("_all")
	require.ErrorIs(t, err, ErrInvalidFileName, "a name starting with _ gives a file ignored by the go tool")
}
