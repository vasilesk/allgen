package example

// Color is a color of a traffic light.
type Color int

//go:generate go run github.com/vasilesk/allgen/cmd/allgen
const (
	// Red means stop.
	Red Color = iota
	// Yellow means prepare to stop.
	Yellow
	// Green means go.
	Green
	colorCount //allgen:skip
)
