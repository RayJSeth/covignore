package coverage

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type Entry struct {
	Raw     string
	File    string // extracted file path (package-qualified)
	NumStmt int
	Count   int
}

type Profile struct {
	Mode    string // e.g. "set", "count", "atomic"
	Entries []Entry
}

func Parse(r io.Reader) (*Profile, error) {
	scanner := bufio.NewScanner(r)

	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("empty coverage file")
	}

	modeLine := scanner.Text()
	if !strings.HasPrefix(modeLine, "mode: ") {
		return nil, fmt.Errorf("invalid coverage file: missing mode line")
	}
	mode := strings.TrimPrefix(modeLine, "mode: ")

	var entries []Entry
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		file := parseFilePath(line)
		numStmt, count := parseStmtCount(line)
		entries = append(entries, Entry{Raw: line, File: file, NumStmt: numStmt, Count: count})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return &Profile{Mode: mode, Entries: entries}, nil
}

func parseFilePath(line string) string {
	token := line
	if before, _, ok := strings.Cut(line, " "); ok {
		token = before
	}
	if idx := strings.LastIndex(token, ":"); idx >= 0 {
		return token[:idx]
	}
	return token
}

func parseStmtCount(line string) (int, int) {
	parts := strings.Fields(line)
	if len(parts) < 3 {
		return 0, 0
	}
	numStmt, _ := strconv.Atoi(parts[1])
	count, _ := strconv.Atoi(parts[2])
	return numStmt, count
}
