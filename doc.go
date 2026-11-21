// Package allgen keeps generated lists of const and var block members in sync with
// their blocks.
//
// The allgen command, run by go generate, reads the const or var block right below its
// directive and writes a function returning every member of the block:
//
//	//go:generate go tool allgen
//	const (
//		Red Color = iota
//		Green
//		Blue
//	)
//
// becomes
//
//	func AllColors() []Color {
//		return []Color{
//			Red,
//			Green,
//			Blue,
//		}
//	}
//
// The same works for any named type, for example one based on string:
//
//	type Status string
//
//	//go:generate go tool allgen
//	const (
//		StatusNew  Status = "new"
//		StatusPaid Status = "paid"
//	)
//
// gives AllStatuses() []Status.
//
// Next to the function the command writes a test calling [Check]. The test records the
// member names the function was generated from and fails once a member is added to or
// removed from the block without running go generate again.
//
// The package depends on the standard library only.
package allgen
