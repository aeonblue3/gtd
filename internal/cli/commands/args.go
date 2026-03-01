package commands

import "strings"

// SplitLeadingPositional returns an initial non-flag token and the remaining args.
// If the first arg looks like a flag, positional is empty and remaining is unchanged.
func SplitLeadingPositional(args []string) (positional string, remaining []string) {
	if len(args) == 0 {
		return "", args
	}
	first := strings.TrimSpace(args[0])
	if strings.HasPrefix(first, "-") || first == "" {
		return "", args
	}
	return first, args[1:]
}
