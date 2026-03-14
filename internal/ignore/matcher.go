package ignore

type Pattern struct {
	Glob   string
	Negate bool
}

type Matcher struct {
	patterns []Pattern
}

func NewMatcher(patterns []Pattern) *Matcher {
	return &Matcher{patterns: patterns}
}

func (m *Matcher) Match(path string) bool {
	ignored := false
	for _, p := range m.patterns {
		if globMatch(p.Glob, path) {
			ignored = !p.Negate
		}
	}
	return ignored
}

func (m *Matcher) Empty() bool {
	return len(m.patterns) == 0
}

func (m *Matcher) Patterns() []Pattern {
	return m.patterns
}
