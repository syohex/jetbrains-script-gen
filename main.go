package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func main() {
	os.Exit(_main())
}

func unixPathToWinPath(unixPath string) (string, error) {
	cmd := exec.Command("cygpath", "-d", "-f", "'"+unixPath+"'")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

func isGitBash() bool {
	return os.Getenv("MSYSTEM") != ""
}

func isPlugin(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasPrefix(lower, "resharper") || strings.HasPrefix(lower, "dot")
}

type jetbrainsTool struct {
	name string
	dir  string
}

func (j *jetbrainsTool) findExecutable() (string, error) {
	var exeName string
	if j.name == "AndroidStudio" {
		exeName = "studio"
	} else {
		exeName = strings.ToLower(j.name)
	}

	exes, err := filepath.Glob(fmt.Sprintf("%s\\ch-0\\*\\bin\\%s64.exe", j.dir, exeName))
	if err != nil {
		return "", err
	}

	if len(exes) == 0 {
		return "", nil
	}

	return exes[0], nil
}

func (j *jetbrainsTool) generateScript(outputDir string) error {
	exe, err := j.findExecutable()
	if err != nil {
		return err
	}
	if exe == "" {
		fmt.Println("Found IDE directory but not found its executables. Skip")
		return nil
	}

	exe = strings.ReplaceAll(exe, "\\", "\\\\")
	script := fmt.Sprintf(`#!/usr/bin/env bash
exec cmd "/c start %s $@"
`, exe)
	out := filepath.Join(outputDir, strings.ToLower(j.name))
	if err := ioutil.WriteFile(out, []byte(script), 0755); err != nil {
		return err
	}

	return nil
}

func collectInstalledJetBrainsTools() ([]*jetbrainsTool, error) {
	installDir := filepath.Join(os.Getenv("LOCALAPPDATA"), "JetBrains", "Toolbox", "apps")
	files, err := ioutil.ReadDir(installDir)
	if err != nil {
		return nil, err
	}

	var ret []*jetbrainsTool
	for _, file := range files {
		name := file.Name()
		if name == "Toolbox" {
			continue
		}

		if isPlugin(name) {
			continue
		}

		// Some IDE has multiple edition(e.g. Pycharm Professional, Education edition)
		if strings.ContainsRune(name, '-') {
			name = strings.Split(name, "-")[0]
		}

		ret = append(ret, &jetbrainsTool{
			name: name,
			dir:  filepath.Join(installDir, file.Name()),
		})
	}

	return ret, nil
}

func _main() int {
	if runtime.GOOS != "windows" {
		fmt.Println("This script supports only Windows")
		return 1
	}

	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s output_dir\n", os.Args[0])
		return 1
	}

	var outputDir string
	if isGitBash() {
		var err error
		outputDir, err = unixPathToWinPath(os.Args[1])
		if err != nil {
			fmt.Println(err)
			return 1
		}
	} else {
		outputDir = os.Args[1]
	}

	ides, err := collectInstalledJetBrainsTools()
	if err != nil {
		fmt.Println(err)
		return 1
	}

	if err := os.MkdirAll(outputDir, 0777); err != nil {
		fmt.Println(err)
		return 1
	}

	for _, ide := range ides {
		fmt.Printf("Generate %s script -> %s\n", ide.name, outputDir)
		if err := ide.generateScript(outputDir); err != nil {
			fmt.Println(err)
			return 1
		}
	}

	return 0
}
