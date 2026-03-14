package cli

var Version = "dev"

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
