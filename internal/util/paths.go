package util

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type ModuleInfo struct {
	Path string
	Dir  string
}

func ModulePath() (string, error) {
	cmd := exec.Command("go", "list", "-m")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func ModuleRoot() (*ModuleInfo, error) {
	cmd := exec.Command("go", "list", "-m", "-json")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var info struct {
		Path string `json:"Path"`
		Dir  string `json:"Dir"`
	}
	if err := json.Unmarshal(out, &info); err != nil {
		return nil, err
	}
	return &ModuleInfo{Path: info.Path, Dir: info.Dir}, nil
}

func WorkspaceModules() ([]ModuleInfo, error) {
	// check if go.work exists by looking upward from cwd.
	workDir := findGoWorkDir()
	if workDir == "" {
		return nil, nil
	}

	cmd := exec.Command("go", "list", "-m", "-json", "all")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("listing workspace modules: %w", err)
	}

	var modules []ModuleInfo
	dec := json.NewDecoder(strings.NewReader(string(out)))
	for dec.More() {
		var info struct {
			Path string `json:"Path"`
			Dir  string `json:"Dir"`
		}
		if err := dec.Decode(&info); err != nil {
			break
		}
		if info.Dir == "" {
			continue
		}
		// only include modules that live under the workspace root.
		absDir, err1 := filepath.Abs(info.Dir)
		absWork, err2 := filepath.Abs(workDir)
		if err1 != nil || err2 != nil {
			continue
		}
		if strings.HasPrefix(absDir, absWork+string(filepath.Separator)) || absDir == absWork {
			modules = append(modules, ModuleInfo{Path: info.Path, Dir: info.Dir})
		}
	}
	return modules, nil
}

func findGoWorkDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	return findGoWorkDirFrom(dir)
}

func findGoWorkDirFrom(dir string) string {
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.work")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func FindCovignore(moduleDir string) string {
	const covignoreFile = ".covignore"
	modCovignore := filepath.Join(moduleDir, covignoreFile)
	if _, err := os.Stat(modCovignore); err == nil {
		return modCovignore
	}
	if _, err := os.Stat(covignoreFile); err == nil {
		return covignoreFile
	}
	return covignoreFile
}
