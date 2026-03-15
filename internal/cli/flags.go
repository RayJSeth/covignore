package cli

import "runtime/debug"

var Version = "dev"

func init() {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		Version = info.Main.Version
	}
}

type Flags struct {
	Min     float64
	HTML    string
	JSON    bool
	Summary bool
	Init    bool
	Preset  string
	Output  string
	DryRun  bool
	Verbose bool
	Check   bool
	ShowVer bool
}
