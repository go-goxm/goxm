package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/codeartifact"
	codeartifactTypes "github.com/aws/aws-sdk-go-v2/service/codeartifact/types"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"
)

type MockCodeArtifactClient struct {
	GetPackageVersionAssetFunc func(
		ctx context.Context,
		params *codeartifact.GetPackageVersionAssetInput,
		optFns ...func(*codeartifact.Options),
	) (*codeartifact.GetPackageVersionAssetOutput, error)

	PublishPackageVersionFunc func(
		ctx context.Context,
		params *codeartifact.PublishPackageVersionInput,
		optFns ...func(*codeartifact.Options),
	) (*codeartifact.PublishPackageVersionOutput, error)
}

func (c *MockCodeArtifactClient) GetPackageVersionAsset(
	ctx context.Context,
	params *codeartifact.GetPackageVersionAssetInput,
	optFns ...func(*codeartifact.Options),
) (*codeartifact.GetPackageVersionAssetOutput, error) {
	return c.GetPackageVersionAssetFunc(ctx, params, optFns...)
}

func (c *MockCodeArtifactClient) PublishPackageVersion(
	ctx context.Context,
	params *codeartifact.PublishPackageVersionInput,
	optFns ...func(*codeartifact.Options),
) (*codeartifact.PublishPackageVersionOutput, error) {
	return c.PublishPackageVersionFunc(ctx, params, optFns...)
}

func TestCodeArtifactModDownload(t *testing.T) {
	t.Setenv("GOMODCACHE", t.TempDir())
	chdir(t, "./testdata/ca_module1")

	config, err := LoadDefaultConfig()
	require.Nilf(t, err, "Error loading default config: %v", err)

	results := make(map[string][]*codeartifact.GetPackageVersionAssetInput)

	for mre, repo := range config.Repos {
		modRegExp := mre

		repo := repo.(*CodeArtifactRepoConfig)

		repo.client = &MockCodeArtifactClient{
			GetPackageVersionAssetFunc: func(
				ctx context.Context,
				params *codeartifact.GetPackageVersionAssetInput,
				optFns ...func(*codeartifact.Options),
			) (*codeartifact.GetPackageVersionAssetOutput, error) {
				results[modRegExp] = append(results[modRegExp], params)
				return nil, fmt.Errorf("Mock Implementation")
			},
		}
	}

	err = runWithConfig(context.Background(), config, []string{"mod", "download"})
	require.Error(t, err)

	expectedResults := map[string][]*codeartifact.GetPackageVersionAssetInput{
		"golang\\.org/x/crypto": []*codeartifact.GetPackageVersionAssetInput{
			{
				Asset:          aws.String("v0.17.0.mod"),
				Package:        aws.String("golang.org+2Fx+2Fcrypto"),
				PackageVersion: aws.String("v0.17.0"),
				Namespace:      aws.String("goxm"),
				Repository:     aws.String("TestRepo1"),
				Domain:         aws.String("TestDomain1"),
				DomainOwner:    aws.String("111111111111"),
				Format:         codeartifactTypes.PackageFormatGeneric,
			},
		},
		"github\\.com/mattn/(.*)": []*codeartifact.GetPackageVersionAssetInput{
			{
				Asset:          aws.String("v0.0.20.mod"),
				Package:        aws.String("github.com+2Fmattn+2Fgo-isatty"),
				PackageVersion: aws.String("v0.0.20"),
				Namespace:      aws.String("Namespace2"),
				Repository:     aws.String("TestRepo2"),
				Domain:         aws.String("TestDomain2"),
				DomainOwner:    aws.String("222222222222"),
				Format:         codeartifactTypes.PackageFormatGeneric,
			},
			{
				Asset:          aws.String("v0.1.13.mod"),
				Package:        aws.String("github.com+2Fmattn+2Fgo-colorable"),
				PackageVersion: aws.String("v0.1.13"),
				Namespace:      aws.String("Namespace2"),
				Repository:     aws.String("TestRepo2"),
				Domain:         aws.String("TestDomain2"),
				DomainOwner:    aws.String("222222222222"),
				Format:         codeartifactTypes.PackageFormatGeneric,
			},
		},
	}

	require.ElementsMatch(t, maps.Keys(expectedResults), maps.Keys(results))
	for k, result := range results {
		require.ElementsMatch(t, expectedResults[k], result)
	}
}

func TestCodeArtifactGet(t *testing.T) {
	t.Setenv("GOMODCACHE", t.TempDir())
	chdir(t, "./testdata/ca_module2")

	config, err := LoadDefaultConfig()
	require.Nilf(t, err, "Error loading default config: %v", err)

	results := make(map[string][]*codeartifact.GetPackageVersionAssetInput)

	for mre, repo := range config.Repos {
		modRegExp := mre

		repo := repo.(*CodeArtifactRepoConfig)

		repo.client = &MockCodeArtifactClient{
			GetPackageVersionAssetFunc: func(
				ctx context.Context,
				params *codeartifact.GetPackageVersionAssetInput,
				optFns ...func(*codeartifact.Options),
			) (*codeartifact.GetPackageVersionAssetOutput, error) {
				results[modRegExp] = append(results[modRegExp], params)
				return nil, fmt.Errorf("Mock Implementation")
			},
		}
	}

	err = runWithConfig(context.Background(), config, []string{"get", "github.com/kelseyhightower/envconfig@v1.4.0"})
	require.Error(t, err)

	expectedResults := map[string][]*codeartifact.GetPackageVersionAssetInput{
		"github\\.com/kelseyhightower/envconfig": []*codeartifact.GetPackageVersionAssetInput{
			{
				Asset:          aws.String("v1.4.0.info"),
				Package:        aws.String("github.com+2Fkelseyhightower+2Fenvconfig"),
				PackageVersion: aws.String("v1.4.0"),
				Namespace:      aws.String("goxm"),
				Repository:     aws.String("TestRepo1"),
				Domain:         aws.String("TestDomain1"),
				DomainOwner:    aws.String("111111111111"),
				Format:         codeartifactTypes.PackageFormatGeneric,
			},
		},
	}

	require.ElementsMatch(t, maps.Keys(expectedResults), maps.Keys(results))
	for k, result := range results {
		require.ElementsMatch(t, expectedResults[k], result)
	}
}

func TestCodeArtifactBuild(t *testing.T) {
	// Cache is created with read-write permissions
	// to avoid error on temp directory cleanup
	t.Setenv("GOFLAGS", "-modcacherw") 
	t.Setenv("GOMODCACHE", t.TempDir())
	chdir(t, "./testdata/ca_module3")

	config, err := LoadDefaultConfig()
	require.Nilf(t, err, "Error loading default config: %v", err)

	results := make(map[string][]*codeartifact.GetPackageVersionAssetInput)

	for mre, repo := range config.Repos {
		modRegExp := mre

		repo := repo.(*CodeArtifactRepoConfig)

		repo.client = &MockCodeArtifactClient{
			GetPackageVersionAssetFunc: func(
				ctx context.Context,
				params *codeartifact.GetPackageVersionAssetInput,
				optFns ...func(*codeartifact.Options),
			) (*codeartifact.GetPackageVersionAssetOutput, error) {
				results[modRegExp] = append(results[modRegExp], params)
				return &codeartifact.GetPackageVersionAssetOutput{
					Asset: io.NopCloser(fileToReader(t, "../ca_module1_assets/"+aws.ToString(params.Asset))),
				},  nil
			},
		}
	}

	buildOutputPath := t.TempDir()+"/ca_module3"
	err = runWithConfig(context.Background(), config, []string{"build", "-o", buildOutputPath})
	require.Nil(t, err, err)
	require.FileExists(t, buildOutputPath)

	expectedResults := map[string][]*codeartifact.GetPackageVersionAssetInput{
		"github\\.com/go-goxm/ca_module1": []*codeartifact.GetPackageVersionAssetInput{
			{
				Asset:          aws.String("v0.1.0.zip"),
				Package:        aws.String("github.com+2Fgo-goxm+2Fca_module1"),
				PackageVersion: aws.String("v0.1.0"),
				Namespace:      aws.String("goxm"),
				Repository:     aws.String("TestRepo2"),
				Domain:         aws.String("TestDomain2"),
				DomainOwner:    aws.String("222222222222"),
				Format:         codeartifactTypes.PackageFormatGeneric,
			},
			{
				Asset:          aws.String("v0.1.0.mod"),
				Package:        aws.String("github.com+2Fgo-goxm+2Fca_module1"),
				PackageVersion: aws.String("v0.1.0"),
				Namespace:      aws.String("goxm"),
				Repository:     aws.String("TestRepo2"),
				Domain:         aws.String("TestDomain2"),
				DomainOwner:    aws.String("222222222222"),
				Format:         codeartifactTypes.PackageFormatGeneric,
			},
			{
				Asset:          aws.String("v0.1.0.info"),
				Package:        aws.String("github.com+2Fgo-goxm+2Fca_module1"),
				PackageVersion: aws.String("v0.1.0"),
				Namespace:      aws.String("goxm"),
				Repository:     aws.String("TestRepo2"),
				Domain:         aws.String("TestDomain2"),
				DomainOwner:    aws.String("222222222222"),
				Format:         codeartifactTypes.PackageFormatGeneric,
			},
		},
	}

	require.ElementsMatch(t, maps.Keys(expectedResults), maps.Keys(results))
	for k, result := range results {
		require.ElementsMatch(t, expectedResults[k], result)
	}
}

func TestCodeArtifactPublish(t *testing.T) {
	t.Setenv("GOMODCACHE", t.TempDir())
	chdir(t, "./testdata/ca_module1")

	config, err := LoadDefaultConfig()
	require.Nilf(t, err, "Error loading default config: %v", err)

	results := make(map[string][]*codeartifact.PublishPackageVersionInput)

	for mre, repo := range config.Repos {
		modRegExp := mre

		repo := repo.(*CodeArtifactRepoConfig)

		repo.client = &MockCodeArtifactClient{
			PublishPackageVersionFunc: func(
				ctx context.Context,
				params *codeartifact.PublishPackageVersionInput,
				optFns ...func(*codeartifact.Options),
			) (*codeartifact.PublishPackageVersionOutput, error) {
				results[modRegExp] = append(results[modRegExp], params)
				return &codeartifact.PublishPackageVersionOutput{}, nil
			},
		}
	}

	err = runWithConfig(context.Background(), config, []string{"publish", "v0.1.0"})
	require.Nil(t, err, err)

	expectedResults := map[string][]*codeartifact.PublishPackageVersionInput{
		"github\\.com/go-goxm/(.*)": []*codeartifact.PublishPackageVersionInput{
			{
				AssetName:      aws.String("v0.1.0.info"),
				AssetSHA256:    aws.String("8b754452eea6eb03ae76b8c39572b90940a1a037026c1f9ca0f5d87e4fc3791b"),
				AssetContent:   fileToReader(t, "../ca_module1_assets/v0.1.0.info"),
				Package:        aws.String("github.com+2Fgo-goxm+2Fca_module1"),
				PackageVersion: aws.String("v0.1.0"),
				Namespace:      aws.String("goxm"),
				Repository:     aws.String("TestRepo1"),
				Domain:         aws.String("TestDomain1"),
				DomainOwner:    aws.String("111111111111"),
				Format:         codeartifactTypes.PackageFormatGeneric,
				Unfinished:     aws.Bool(true),
			},
			{
				AssetName:      aws.String("v0.1.0.mod"),
				AssetSHA256:    aws.String("84ab8e2a063142265a796cb95446d794a61e0568f91b56f519fd84e11e23f0a7"),
				AssetContent:   fileToReader(t, "../ca_module1_assets/v0.1.0.mod"),
				Package:        aws.String("github.com+2Fgo-goxm+2Fca_module1"),
				PackageVersion: aws.String("v0.1.0"),
				Namespace:      aws.String("goxm"),
				Repository:     aws.String("TestRepo1"),
				Domain:         aws.String("TestDomain1"),
				DomainOwner:    aws.String("111111111111"),
				Format:         codeartifactTypes.PackageFormatGeneric,
				Unfinished:     aws.Bool(true),
			},
			{
				AssetName:      aws.String("v0.1.0.zip"),
				AssetSHA256:    aws.String("f3f283d638bd8e446d593aec7a6c5bf1e234ed9c91a611362a192fc3f508cf99"),
				AssetContent:   fileToReader(t, "../ca_module1_assets/v0.1.0.zip"),
				Package:        aws.String("github.com+2Fgo-goxm+2Fca_module1"),
				PackageVersion: aws.String("v0.1.0"),
				Namespace:      aws.String("goxm"),
				Repository:     aws.String("TestRepo1"),
				Domain:         aws.String("TestDomain1"),
				DomainOwner:    aws.String("111111111111"),
				Format:         codeartifactTypes.PackageFormatGeneric,
				Unfinished:     aws.Bool(false),
			},
		},
	}

	require.ElementsMatch(t, maps.Keys(expectedResults), maps.Keys(results))
	for k, result := range results {
		// // Use to write assets to directory
		// for _, r := range result {
		// 	readerToFile(t, r.AssetContent, "../ca_module1_assets/" + *r.AssetName)
		// }
		require.ElementsMatch(t, expectedResults[k], result)
	}
}

func fileToReader(t *testing.T, p string) io.Reader {
	f, err := os.Open(p)
	require.Nil(t, err)
	defer f.Close()

	b, err := io.ReadAll(f)
	require.Nil(t, err)

	return bytes.NewReader(b)
}

func readerToFile(t *testing.T, r io.Reader, p string) {
	f, err := os.Create(p)
	require.Nil(t, err)
	defer f.Close()

	_, err = io.Copy(f, r)
	require.Nil(t, err)
}
