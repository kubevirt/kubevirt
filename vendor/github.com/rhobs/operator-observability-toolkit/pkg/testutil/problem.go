package testutil

type Problem struct {
	// The name of the Metric, Recording Rule, or Alert indicated by the Problem
	ResourceName string

	// The description of the problem found
	Description string
}
