package runner

import (
	"io"
	"os/exec"
	"strings"
)

type Result struct {
	ExitCode   int
	RawProfile string
}

func RunGoTest(args []string, rawProfilePath string, stdout, stderr io.Writer, stdin io.Reader) (*Result, error) {
	goTestArgs := []string{"test"}

	hasCoverProfile := false
	for _, a := range args {
		if strings.HasPrefix(a, "-coverprofile") || strings.HasPrefix(a, "--coverprofile") {
			hasCoverProfile = true
			break
		}
	}

	if !hasCoverProfile {
		goTestArgs = append(goTestArgs, "-coverprofile="+rawProfilePath)
	}
	goTestArgs = append(goTestArgs, args...)

	cmd := exec.Command("go", goTestArgs...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = stdin

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return nil, err
		}
	}

	profilePath := rawProfilePath
	if hasCoverProfile {
		profilePath = extractCoverProfile(args)
	}

	return &Result{ExitCode: exitCode, RawProfile: profilePath}, nil
}

func extractCoverProfile(args []string) string {
	for i, a := range args {
		if a == "-coverprofile" || a == "--coverprofile" {
			if i+1 < len(args) {
				return args[i+1]
			}
		}
		for _, prefix := range []string{"-coverprofile=", "--coverprofile="} {
			if after, ok := strings.CutPrefix(a, prefix); ok {
				return after
			}
		}
	}
	return ""
}
