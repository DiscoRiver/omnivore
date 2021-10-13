package group

import "github.com/discoriver/massh"

type HostOutputGroup interface {
	// Length of output with an offset of the difference between hosts, for example for time, you might have "15:38:55",
	// but if the seconds are either 1 ahead/behind on some hosts, you would get a length of 8-3, since the value of seconds
	// could either be 55, 54, or 56, so we're measuring the number of deviations.
	GetOutputlength()

	// Get hosts in this group.
	GetHosts()

	// Get the difference between the original output on which other hosts are compared, and the max deviation. I think we
	// may want to be able to configure the max deviation to ensure we fine-tune how grouping works. We could end up with
	// rare situations where two unrelated outputs are grouped together.
	MaxDifference()

	// Compare the output within a mass.Result to that of a group output, and measure it's deviation. This will be how we
	// ultimately decide how to group output together, at least for now.
	CompareOutput(massh.Result) int
}

// TODO: Investigate performance difference between Levenshtein distance values and hashing comparison in long output.

// Grouping for short output groups. We can perform grouping here based on an exact match, or by a Levenshtein
// distance value. We're typically expecting single-line output from these commands. Some examples could for commands
// that would fall into this category could be "date", "lsb_release -v", or "uname".
type ShortOutputGroup struct {
	Hosts map[string]int

	Output []byte
	// Output length
	len int

	MaxDeviation int
}

// It's unclear and not obvious how we group long output hosts right now. Using a Levenshtein distance for commands
// that a providing an undetermined length is tricky due to the resource requirements and speed.
type LongOutputGroup struct {
	Hosts []string

	Output []string
	// Output length
	len int

	MaxDeviation int
}