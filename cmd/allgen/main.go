// Command allgen writes a function returning every member of the const or var block
// right below its go:generate directive, and a test that fails once the function no
// longer matches the block.
//
// Usage:
//
//	//go:generate go tool allgen [-name AllColors] [-output colors_gen.go] [-test=false]
//
// Flags:
//
//	-name    name of the generated function; defaults to All<Type>s, or all<Type>s for an
//	         unexported type, and is required for members of other types
//	-output  name of the generated file; defaults to the function name in snake_case
//	         with the _gen.go suffix
//	-test    also write <output>_test.go checking that the function is up to date
//	         (default true)
//
// The command must be run by go generate, which tells it the file and line of the
// directive.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"

	"github.com/samber/mo"

	"github.com/vasilesk/allgen/internal/gen"
)

func main() {
	err := run(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "allgen:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	params, err := parseParams(args)
	if err != nil {
		return err
	}

	err = gen.Generate(ctx, params)
	if errors.Is(err, gen.ErrNameRequired) {
		return fmt.Errorf("generating: %w; pass -name", err)
	}

	if err != nil {
		return fmt.Errorf("generating: %w", err)
	}

	return nil
}

var errNotGoGenerate = errors.New("GOFILE and GOLINE are not set; run allgen with go generate")

// parseParams builds the parameters from the command line and the environment set by
// go generate. On invalid flags and on -h it prints the usage and exits.
func parseParams(args []string) (gen.Params, error) {
	flags := flag.NewFlagSet("allgen", flag.ExitOnError)
	name := flags.String("name", "", "name of the generated `function` (default All<Type>s)")
	output := flags.String("output", "", "name of the generated `file` (default <function>_gen.go in snake_case)")
	withTest := flags.Bool("test", true, "also write a test checking that the function is up to date")

	_ = flags.Parse(args) // with ExitOnError, Parse exits instead of returning an error

	if flags.NArg() > 0 {
		return gen.Params{}, fmt.Errorf("unexpected arguments: %v", flags.Args())
	}

	file, line, err := directivePosition()
	if err != nil {
		return gen.Params{}, err
	}

	dir, err := packageDir()
	if err != nil {
		return gen.Params{}, err
	}

	maybeName, err := optional(*name, gen.NewFuncName)
	if err != nil {
		return gen.Params{}, fmt.Errorf("parsing -name: %w", err)
	}

	maybeOutput, err := optional(*output, gen.NewFileName)
	if err != nil {
		return gen.Params{}, fmt.Errorf("parsing -output: %w", err)
	}

	return gen.Params{
		Dir:    dir,
		File:   file,
		Line:   line,
		Name:   maybeName,
		Output: maybeOutput,
		Test:   *withTest,
	}, nil
}

// directivePosition reads the file and line of the directive set by go generate.
func directivePosition() (string, int, error) {
	file, rawLine := os.Getenv("GOFILE"), os.Getenv("GOLINE")
	if file == "" || rawLine == "" {
		return "", 0, errNotGoGenerate
	}

	line, err := strconv.Atoi(rawLine)
	if err != nil {
		return "", 0, fmt.Errorf("parsing GOLINE: %w", err)
	}

	return file, line, nil
}

// packageDir returns the working directory, set by go generate to the package directory,
// with symbolic links resolved so that paths match those reported by the go command.
func packageDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}

	dir, err := filepath.EvalSymlinks(wd)
	if err != nil {
		return "", fmt.Errorf("resolving working directory: %w", err)
	}

	return dir, nil
}

// optional parses a flag whose empty value means it is not set.
func optional[T any](raw string, parse func(string) (T, error)) (mo.Option[T], error) {
	if raw == "" {
		return mo.None[T](), nil
	}

	value, err := parse(raw)
	if err != nil {
		return mo.None[T](), err
	}

	return mo.Some(value), nil
}
