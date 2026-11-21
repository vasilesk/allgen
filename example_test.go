package allgen_test

import (
	"fmt"

	"github.com/vasilesk/allgen"
)

func ExampleBlock_Verify() {
	err := allgen.Block{
		File:      "example/color.go",
		Directive: "//go:generate go run github.com/vasilesk/allgen/cmd/allgen",
		Names:     []string{"Red", "Green"},
	}.Verify()

	fmt.Println(err)
	// Output:
	// generated list is outdated: block in example/color.go: added Yellow
}
