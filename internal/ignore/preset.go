package ignore

import "fmt"

var Presets = map[string][]Pattern{
	"generated": {
		{Glob: "**/*.pb.go"},
		{Glob: "**/*_gen.go"},
		{Glob: "**/*_generated.go"},
		{Glob: "**/mock/**"},
		{Glob: "**/mocks/**"},
		{Glob: "**/*_mock.go"},
		{Glob: "**/ent/**"},
		{Glob: "**/sqlc/**"},
	},
}

func PresetNames() []string {
	names := make([]string, 0, len(Presets))
	for k := range Presets {
		names = append(names, k)
	}
	return names
}

func LoadWithPreset(path string, preset string) (*Matcher, error) {
	m, err := Load(path)
	if err != nil {
		return nil, err
	}

	if preset == "" {
		return m, nil
	}

	presetPatterns, ok := Presets[preset]
	if !ok {
		return nil, fmt.Errorf("unknown preset %q (available: %v)", preset, PresetNames())
	}

	combined := make([]Pattern, 0, len(m.Patterns())+len(presetPatterns))
	combined = append(combined, presetPatterns...)
	combined = append(combined, m.Patterns()...)
	return NewMatcher(combined), nil
}
