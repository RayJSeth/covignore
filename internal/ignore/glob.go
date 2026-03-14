package ignore

import (
	"path"
	"strings"
)

func globMatch(pattern, name string) bool {
	return doMatch(splitPath(pattern), splitPath(name))
}

func splitPath(s string) []string {
	s = strings.TrimPrefix(s, "/")
	if s == "" {
		return nil
	}
	return strings.Split(s, "/")
}

func doMatch(pattern, name []string) bool {
	for len(pattern) > 0 {
		seg := pattern[0]
		pattern = pattern[1:]

		if seg == "**" {
			return matchDoubleStar(pattern, name)
		}

		// Need a name segment to match against.
		if len(name) == 0 {
			return false
		}

		matched, err := path.Match(seg, name[0])
		if err != nil || !matched {
			return false
		}
		name = name[1:]
	}
	return len(name) == 0
}

func matchDoubleStar(pattern, name []string) bool {
	if len(pattern) == 0 {
		return true
	}
	for i := 0; i <= len(name); i++ {
		if doMatch(pattern, name[i:]) {
			return true
		}
	}
	return false
}
