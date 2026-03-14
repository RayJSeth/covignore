package coverage

import "fmt"

type Stats struct {
	Statements int
	Covered    int
	Files      []string
}

func (s Stats) Percent() float64 {
	if s.Statements == 0 {
		return 0
	}
	return float64(s.Covered) / float64(s.Statements) * 100
}

func (s Stats) String() string {
	return fmt.Sprintf("%.1f%%", s.Percent())
}

func ComputeStats(profile *Profile) Stats {
	var st Stats
	seen := map[string]struct{}{}
	for _, e := range profile.Entries {
		st.Statements += e.NumStmt
		if e.Count > 0 {
			st.Covered += e.NumStmt
		}
		if _, ok := seen[e.File]; !ok {
			seen[e.File] = struct{}{}
			st.Files = append(st.Files, e.File)
		}
	}
	return st
}
