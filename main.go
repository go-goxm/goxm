package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/exp/maps"
	"golang.org/x/mod/module"
	"golang.org/x/mod/zip"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		logf("%v", err)

		var exitCode = 1
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
		os.Exit(exitCode)
	}
}

func run(ctx context.Context, args []string) error {
	config, err := LoadDefaultConfig()
	if err != nil {
		return err
	}
	return runWithConfig(ctx, config, args)
}

func runWithConfig(ctx context.Context, config *Config, args []string) error {

	if len(args) > 0 && args[0] == "publish" {
		return publish(context.Background(), config, args[1:])
	}

	proxyServer := httptest.NewServer(newProxyHandler(config))
	defer proxyServer.Close()

	goProxy := os.Getenv("GOPROXY")
	if goProxy == "" {
		goProxy = "https://proxy.golang.org,direct"
	}
	goProxy = fmt.Sprintf("%s,%s", proxyServer.URL, goProxy)

	goNoSumDB := os.Getenv("GONOSUMDB")
	if goNoSumDB == "" {
		goNoSumDB = strings.Join(maps.Keys(config.Repos), ",")
	} else {
		goNoSumDB += "," + strings.Join(maps.Keys(config.Repos), ",")
	}

	cmd := exec.Command("go")
	cmd.Args = append(cmd.Args, args...)
	cmd.Env = append(os.Environ(), "GOPROXY="+goProxy, "GONOSUMDB="+goNoSumDB)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func newProxyHandler(config *Config) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			resp.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		atIndex := strings.Index(req.URL.Path, "@")
		if atIndex < 0 {
			logf("Error parsing request path: %v: '@' expected", req.URL.Path)
			resp.WriteHeader(http.StatusBadRequest)
			return
		}

		modPath, err := module.UnescapePath(strings.Trim(req.URL.Path[:atIndex], "/"))
		if err != nil {
			logf("Error unescaping module path: %v", err)
			resp.WriteHeader(http.StatusBadRequest)
			return
		}

		attifact := req.URL.Path[atIndex:]

		for moduleGlob, repository := range config.Repos {
			match, _ := regexp.MatchString(globToRegexp(moduleGlob), modPath)
			if !match {
				continue
			}

			reader, status, err := repository.Get(req.Context(), modPath, attifact)
			if err != nil {
				// Respond with `Forbidden`` to prevent Go from
				// trying to get the module from another proxy
				resp.WriteHeader(status)
				logf("%v", err)
				return
			}

			_, err = io.Copy(resp, reader)
			if err != nil {
				logf("Error writing response: %v: %v", req.URL.Path, err)
				return
			}
			reader.Close()

			return
		}

		resp.WriteHeader(http.StatusNotFound)
	})
}

func publish(ctx context.Context, config *Config, args []string) error {

	if len(args) == 0 || len(args) > 1 {
		return fmt.Errorf("Unsupported arguments: Usage: goxm publish <version>")
	}

	version := strings.TrimSpace(args[0])

	modPath, goModData, goModFilePath, err := getGoModule(ctx)
	if err != nil {
		return err
	}

	infoData, gitRootPath, err := getGoInfoFromGit(ctx, version)
	if err != nil {
		return err
	}

	subDir, err := filepath.Rel(gitRootPath, filepath.Dir(goModFilePath))
	if err != nil {
		return fmt.Errorf("Unable to resolve relative path to Git repository: %w", err)
	}

	if subDir == "." {
		// If the Git root and the module directories are the same
		// then clear `subDir` so that all paths are included in
		// the zip file and not just the ones starting with "."
		// See the `CreateFromVCS()` docs for more information
		subDir = ""
	} else if strings.HasPrefix(subDir, "..") {
		return fmt.Errorf("Unable to resolve go.mod path within Git repository")
	}

	modVersion := module.Version{
		Path:    modPath,
		Version: version,
	}

	zipBuffer := bytes.NewBuffer(nil)
	err = zip.CreateFromVCS(zipBuffer, modVersion, gitRootPath, version, subDir)
	if err != nil {
		return err
	}

	for moduleGlob, repository := range config.Repos {
		match, _ := regexp.MatchString(globToRegexp(moduleGlob), modPath)
		if !match {
			continue
		}

		return repository.Put(
			context.Background(),
			modPath,
			version,
			goModData,
			infoData,
			zipBuffer.Bytes(),
		)
	}

	return fmt.Errorf("No repository found matching module: %v", modPath)
}
