package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/mod/modfile"
)

type Info struct {
	Version string    // version string
	Time    time.Time // commit time
}

func logf(format string, args ...any) {
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	fmt.Fprintf(os.Stderr, "GOXM: "+format, args...)
}

func getGoInfoFromGit(ctx context.Context, version string) ([]byte, string, error) {

	gitRootPath, err := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return nil, "", fmt.Errorf("Current directory is not in a Git repository: %w", err)
	}

	gitCommitTime, err := exec.CommandContext(ctx, "git", "show", "--no-patch", "--format=%ct", version).Output()
	if err != nil {
		return nil, "", fmt.Errorf("Git revision not found: %s: %w", version, err)
	}

	gitCommitTimeInt64, err := strconv.ParseInt(string(bytes.TrimSpace(gitCommitTime)), 0, 64)
	if err != nil {
		return nil, "", fmt.Errorf("%w", err)
	}

	gitVersionInfo := Info{
		Version: version,
		Time:    time.Unix(gitCommitTimeInt64, 0),
	}

	gitVersionInfoJSON, err := json.MarshalIndent(gitVersionInfo, "", "    ")
	if err != nil {
		return nil, "", err
	}

	return gitVersionInfoJSON, string(bytes.TrimSpace(gitRootPath)), nil
}

func getGoModule(ctx context.Context) (string, []byte, string, error) {

	cwd, err := os.Getwd()
	if err != nil {
		return "", nil, "", err
	}

	goModFilePath := filepath.Join(cwd, "go.mod")
	goModFile, err := os.Open(goModFilePath)
	if err != nil {
		return "", nil, "", fmt.Errorf("Current directory does not contain a Go module file (go.mod)")
	}
	defer goModFile.Close()

	goModData, err := io.ReadAll(goModFile)
	if err != nil {
		return "", nil, "", fmt.Errorf("Go module file (go.mod) could not be read")
	}

	goMod, err := modfile.ParseLax(goModFilePath, goModData, nil)
	if err != nil {
		return "", nil, "", fmt.Errorf("Go module file (go.mod) could not be parsed")
	}

	goModName := goMod.Module.Mod.Path
	if goModName == "" {
		return "", nil, "", fmt.Errorf("Go module name not found")
	}

	return goModName, goModData, goModFilePath, nil
}
