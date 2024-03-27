package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type RawConfig struct {
	Repos map[string]json.RawMessage `json:"repos"`
}

type Repository interface {
	Get(ctx context.Context, module, attifact string) (io.ReadCloser, error)
	Put(ctx context.Context, module, version string, goModData, goInfoData, goZipData []byte) error
}

type Config struct {
	Repos map[string]Repository
}

type RepoTypeConfig struct {
	Type string `json:"type"`
}

const defaultConfigName = ".goxm.json"

func LoadDefaultConfig() (*Config, error) {
	var err error
	var configPath string
	var prevConfigPath string

	for cp := defaultConfigName; ; cp = filepath.Join("..", cp) {
		prevConfigPath = configPath
		configPath, err = filepath.Abs(cp)
		if err != nil || prevConfigPath == configPath {
			return nil, fmt.Errorf("Config file not found: %v", defaultConfigName)
		}

		configFile, err := os.Open(configPath)
		if err != nil {
			continue
		}

		config, err := LoadConfig(configFile)
		if err != nil {
			return nil, fmt.Errorf("Error loading default config: %v: %w", configPath, err)
		}
		return config, nil
	}
}

func LoadConfig(configReader io.Reader) (*Config, error) {
	config := &Config{
		Repos: map[string]Repository{},
	}

	var rawConfig *RawConfig
	err := json.NewDecoder(configReader).Decode(&rawConfig)
	if err != nil {
		return nil, fmt.Errorf("Error reading file: %w", err)
	}

	for moduleGlob, rawRepoConfig := range rawConfig.Repos {
		_, err = regexp.Compile(globToRegexp(moduleGlob))
		if err != nil {
			return nil, fmt.Errorf("Malformed module glob: %v: %w", moduleGlob, err)
		}

		var repoTypeConfig *RepoTypeConfig
		err = json.Unmarshal(rawRepoConfig, &repoTypeConfig)
		if err != nil {
			return nil, fmt.Errorf("Error parsing repo config: %v: %w", moduleGlob, err)
		}

		switch strings.ToLower(repoTypeConfig.Type) {
		case "codeartifact":
			var codeArtifactRepoConfig *CodeArtifactRepoConfig
			err = json.Unmarshal(rawRepoConfig, &codeArtifactRepoConfig)
			if err != nil {
				return nil, fmt.Errorf("Error parsing repo config: %v: %w", repoTypeConfig.Type, err)
			}
			config.Repos[moduleGlob] = codeArtifactRepoConfig

		default:
			return nil, fmt.Errorf("Repository type not supported: %v", repoTypeConfig.Type)
		}
	}

	return config, nil
}

func globToRegexp(glob string) string {
	return strings.ReplaceAll(regexp.QuoteMeta(glob), "\\*", "(.*)")
}
