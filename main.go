package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

func main() {
	if err := run(); err != nil {
		Logf("%v", err)

		var exitCode = 1
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
		os.Exit(exitCode)
	}
}

func Logf(format string, args ...any) {
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	fmt.Fprintf(os.Stderr, "GOXM: "+format, args...)
}

func run() error {

	config, err := LoadDefaultConfig()
	if err != nil {
		return err
	}

	if len(os.Args) > 1 && os.Args[1] == "publish" {
		return publish(os.Args[2:])
	}

	proxyServer := httptest.NewServer(newProxyHandler(config))
	defer proxyServer.Close()

	goProxy := os.Getenv("GOPROXY")
	if goProxy == "" {
		goProxy = "https://proxy.golang.org,direct"
	}
	goProxy = fmt.Sprintf("%s,%s", proxyServer.URL, goProxy)

	cmd := exec.Command("go")
	cmd.Args = append(cmd.Args, os.Args[1:]...)
	cmd.Env = append(os.Environ(), "GOPROXY="+goProxy)
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
			resp.WriteHeader(http.StatusBadRequest)
			return
		}

		module := strings.Trim(req.URL.Path[:atIndex], "/")
		attifact := req.URL.Path[atIndex:]

		for moduleRegexp, repository := range config.Repos {
			match, _ := regexp.MatchString(moduleRegexp, module)
			if !match {
				continue
			}

			reader, err := repository.Get(req.Context(), module, attifact)
			if err != nil {
				// Respond with `Forbidden`` to prevent Go from
				// trying to get the module from another proxy
				resp.WriteHeader(http.StatusForbidden)
				Logf("%v", err)
				return
			}

			_, err = io.Copy(resp, reader)
			if err != nil {
				Logf("Error writing response: %v: %v\n", req.URL.Path, err)
				return
			}
			reader.Close()

			return
		}

		resp.WriteHeader(http.StatusNotFound)
	})
}

func publish(args []string) error {
	return fmt.Errorf("Publish is not implemented")
}
