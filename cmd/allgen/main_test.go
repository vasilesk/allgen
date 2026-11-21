package main

import (
	"strconv"
	"testing"

	"github.com/samber/mo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vasilesk/allgen/internal/gen"
)

func TestParseParams(t *testing.T) {
	setEnv(t, "color.go", "7")

	got, err := parseParams([]string{"-name", "AllColors", "-output", "colors.go", "-test=false"})
	require.NoError(t, err)
	assert.Equal(t, "color.go", got.File)
	assert.Equal(t, 7, got.Line)
	assert.Equal(t, mo.Some(gen.FuncName("AllColors")), got.Name)
	assert.Equal(t, mo.Some(gen.FileName("colors.go")), got.Output)
	assert.False(t, got.Test)
	assert.NotEmpty(t, got.Dir)

	defaults, err := parseParams(nil)
	require.NoError(t, err)
	assert.Equal(t, mo.None[gen.FuncName](), defaults.Name)
	assert.Equal(t, mo.None[gen.FileName](), defaults.Output)
	assert.True(t, defaults.Test)
}

func TestParseParamsErrors(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		file    string
		line    string
		wantErr error
	}{
		{name: "not run by go generate", wantErr: errNotGoGenerate},
		{name: "invalid line", file: "color.go", line: "x", wantErr: strconv.ErrSyntax},
		{
			name:    "invalid name",
			args:    []string{"-name", "all-colors"},
			file:    "color.go",
			line:    "7",
			wantErr: gen.ErrInvalidFuncName,
		},
		{
			name:    "test file as output",
			args:    []string{"-output", "x_test.go"},
			file:    "color.go",
			line:    "7",
			wantErr: gen.ErrInvalidFileName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setEnv(t, tt.file, tt.line)

			_, err := parseParams(tt.args)
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestParseParamsRejectsArguments(t *testing.T) {
	setEnv(t, "color.go", "7")

	_, err := parseParams([]string{"extra"})
	require.Error(t, err)
}

// setEnv sets the variables go generate passes to the command. Tests using it cannot run
// in parallel, as the environment is shared by the process.
func setEnv(t *testing.T, file, line string) {
	t.Helper()

	t.Setenv("GOFILE", file)
	t.Setenv("GOLINE", line)
}
