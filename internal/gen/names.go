package gen

import (
	"errors"
	"fmt"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"
)

// FuncName is the name of a generated function: a Go identifier other than _ and init.
type FuncName string

// NewFuncName validates s as a function name.
func NewFuncName(s string) (FuncName, error) {
	if !token.IsIdentifier(s) || s == "_" || s == "init" {
		return "", fmt.Errorf("%w: %q", ErrInvalidFuncName, s)
	}

	return FuncName(s), nil
}

// FileName is the base name of a generated non-test Go file the go tool does not ignore:
// it ends with .go but not with _test.go, and does not start with . or _.
type FileName string

// NewFileName validates s as a file name.
func NewFileName(s string) (FileName, error) {
	valid := s == filepath.Base(s) &&
		strings.HasSuffix(s, ".go") &&
		!strings.HasSuffix(s, "_test.go") &&
		!strings.HasPrefix(s, ".") &&
		!strings.HasPrefix(s, "_")
	if !valid {
		return "", fmt.Errorf("%w: %q", ErrInvalidFileName, s)
	}

	return FileName(s), nil
}

// TestFile is the name of the test file generated next to f.
func (f FileName) TestFile() string {
	return strings.TrimSuffix(string(f), ".go") + "_test.go"
}

var (
	// ErrInvalidFuncName means a string is not usable as the name of a generated function.
	ErrInvalidFuncName = errors.New("invalid function name")
	// ErrInvalidFileName means a string is not usable as the name of a generated file.
	ErrInvalidFileName = errors.New("invalid file name")
)

// defaultFuncName derives the function name from the type of the members: AllColors
// for Color, allColors for color.
func defaultFuncName(typ types.Type, pkg *types.Package) (FuncName, error) {
	named, ok := types.Unalias(typ).(*types.Named)
	if !ok || named.Obj().Pkg() != pkg || named.TypeArgs().Len() > 0 {
		return "", fmt.Errorf("%w: members are of type %s", ErrNameRequired, typeString(typ, pkg))
	}

	typeName := named.Obj().Name()

	prefix := "all"
	if token.IsExported(typeName) {
		prefix = "All"
	}

	return NewFuncName(prefix + upperFirst(plural(typeName)))
}

// defaultFileName derives the file name from the function name: all_colors_gen.go for
// AllColors.
func defaultFileName(name FuncName) (FileName, error) {
	return NewFileName(snakeCase(string(name)) + "_gen.go")
}

// plural applies the regular English plural rules to the last word of a name:
// Status becomes Statuses, Category becomes Categories, Key becomes Keys.
func plural(word string) string {
	lower := strings.ToLower(word)

	switch {
	case hasAnySuffix(lower, "s", "x", "z", "ch", "sh") && !endsWithUpper(word):
		return word + "es"
	case strings.HasSuffix(lower, "y") && len(lower) > 1 && !strings.ContainsRune("aeiou", rune(lower[len(lower)-2])):
		return word[:len(word)-1] + "ies"
	default:
		return word + "s"
	}
}

// snakeCase converts a MixedCaps identifier to snake_case, keeping acronyms together:
// AllHTTPCodes becomes all_http_codes, AllIDs becomes all_ids.
func snakeCase(name string) string {
	runes := []rune(name)

	var b strings.Builder

	for i, r := range runes {
		if wordStart(runes, i) {
			b.WriteByte('_')
		}

		b.WriteRune(unicode.ToLower(r))
	}

	return b.String()
}

// wordStart reports whether runes[i] begins a new word: an upper-case letter after a
// lower-case letter or a digit, or the last letter of an acronym followed by a
// lower-case letter other than a plural s ending the name.
func wordStart(runes []rune, i int) bool {
	if i == 0 || !unicode.IsUpper(runes[i]) {
		return false
	}

	prev := runes[i-1]
	if unicode.IsLower(prev) || unicode.IsDigit(prev) {
		return true
	}

	hasNext := i+1 < len(runes)
	pluralS := i+2 == len(runes) && runes[i+1] == 's'

	return hasNext && unicode.IsUpper(prev) && unicode.IsLower(runes[i+1]) && !pluralS
}

func upperFirst(s string) string {
	r, size := utf8.DecodeRuneInString(s)

	return string(unicode.ToUpper(r)) + s[size:]
}

func hasAnySuffix(s string, suffixes ...string) bool {
	for _, suffix := range suffixes {
		if strings.HasSuffix(s, suffix) {
			return true
		}
	}

	return false
}

func endsWithUpper(s string) bool {
	r, _ := utf8.DecodeLastRuneInString(s)

	return unicode.IsUpper(r)
}
