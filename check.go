package allgen

import (
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"
	"testing"

	"github.com/vasilesk/allgen/internal/source"
)

// Check fails tb when the members of the block described by b differ from b.Names.
//
// It skips tb when b.File does not exist, which happens when a test binary runs away from
// the package sources: the check cannot tell a stale list from a missing file.
func Check(tb testing.TB, b Block) {
	tb.Helper()

	err := b.Verify()
	if errors.Is(err, fs.ErrNotExist) {
		tb.Skipf("allgen: %v", err)

		return
	}

	if err != nil {
		tb.Errorf("allgen: %v; run go generate", err)
	}
}

// Block is a const or var block as it was when the list of its members was generated.
type Block struct {
	// File is the path of the file holding the block, relative to the package directory.
	File string
	// Directive is the go:generate comment placed above the block, verbatim.
	// It identifies the block within File, so it must occur in File only once.
	Directive string
	// Names are the members of the block in declaration order, without blank identifiers
	// and specs marked with //allgen:skip.
	Names []string
}

// Verify reads b.File and compares the members of the block below b.Directive with
// b.Names, order included.
//
// It returns an error wrapping [ErrOutdated] when they differ and one wrapping
// [fs.ErrNotExist] when b.File does not exist.
func (b Block) Verify() error {
	names, err := readMembers(b.File, b.Directive)
	if err != nil {
		return fmt.Errorf("reading block in %s: %w", b.File, err)
	}

	if slices.Equal(names, b.Names) {
		return nil
	}

	return fmt.Errorf("%w: block in %s: %s", ErrOutdated, b.File, describeChange(b.Names, names))
}

// ErrOutdated means the members of a block differ from the names its list was generated from.
var ErrOutdated = errors.New("generated list is outdated")

func readMembers(path, directive string) ([]string, error) {
	file, err := source.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("parsing source: %w", err)
	}

	line, err := file.DirectiveLine(directive)
	if err != nil {
		return nil, fmt.Errorf("locating directive: %w", err)
	}

	block, err := file.Block(line)
	if err != nil {
		return nil, fmt.Errorf("reading members: %w", err)
	}

	return block.Names, nil
}

// describeChange tells how the members of a block changed from was to now,
// for example "added Purple; removed Green".
func describeChange(was, now []string) string {
	changes := slices.DeleteFunc([]string{
		joinNonEmpty("added ", difference(now, was)),
		joinNonEmpty("removed ", difference(was, now)),
	}, func(s string) bool { return s == "" })

	if len(changes) == 0 {
		return "members reordered"
	}

	return strings.Join(changes, "; ")
}

// difference returns the elements of a missing from b, in the order of a.
func difference(a, b []string) []string {
	return slices.DeleteFunc(slices.Clone(a), func(s string) bool { return slices.Contains(b, s) })
}

func joinNonEmpty(prefix string, names []string) string {
	if len(names) == 0 {
		return ""
	}

	return prefix + strings.Join(names, ", ")
}
