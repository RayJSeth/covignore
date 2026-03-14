package coverage

import (
	"sort"
	"strings"

	"github.com/RayJSeth/covignore/internal/ignore"
)

type ModuleMatcher struct {
	ModulePrefix string
	Matcher      *ignore.Matcher
}

func Filter(profile *Profile, matcher *ignore.Matcher, modulePrefix string) *Profile {
	if matcher.Empty() {
		return profile
	}

	return FilterMultiModule(profile, []ModuleMatcher{{ModulePrefix: modulePrefix, Matcher: matcher}})
}

func FilterMultiModule(profile *Profile, modules []ModuleMatcher) *Profile {
	normalized := make([]ModuleMatcher, len(modules))
	copy(normalized, modules)
	for i := range normalized {
		if normalized[i].ModulePrefix != "" && !strings.HasSuffix(normalized[i].ModulePrefix, "/") {
			normalized[i].ModulePrefix += "/"
		}
	}
	// longest match wins as child.
	// TODO: does this logic work with symlinks?
	sort.Slice(normalized, func(i, j int) bool {
		return len(normalized[i].ModulePrefix) > len(normalized[j].ModulePrefix)
	})

	filtered := &Profile{Mode: profile.Mode}
	for _, e := range profile.Entries {
		if shouldIgnoreEntry(e, normalized) {
			continue
		}
		filtered.Entries = append(filtered.Entries, e)
	}
	return filtered
}

func shouldIgnoreEntry(e Entry, modules []ModuleMatcher) bool {
	for _, mm := range modules {
		if mm.Matcher.Empty() {
			continue
		}
		if mm.ModulePrefix == "" {
			if mm.Matcher.Match(e.File) {
				return true
			}
			continue
		}
		if after, ok := strings.CutPrefix(e.File, mm.ModulePrefix); ok {
			rel := after
			return mm.Matcher.Match(rel)
		}
	}
	return false
}
