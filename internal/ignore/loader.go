package ignore

import (
	"bufio"
	"os"
	"strings"
)

func Load(path string) (*Matcher, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return NewMatcher(nil), nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var patterns []Pattern
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		neg := false
		if strings.HasPrefix(line, "!") {
			neg = true
			line = line[1:]
		}
		patterns = append(patterns, Pattern{Glob: line, Negate: neg})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return NewMatcher(patterns), nil
}
