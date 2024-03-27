package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/codeartifact"
	codeartifactTypes "github.com/aws/aws-sdk-go-v2/service/codeartifact/types"
	"golang.org/x/exp/slices"
)

type CodeArtifactClient interface {
	ListPackageVersions(
		ctx context.Context,
		params *codeartifact.ListPackageVersionsInput,
		optFns ...func(*codeartifact.Options),
	) (*codeartifact.ListPackageVersionsOutput, error)

	GetPackageVersionAsset(
		ctx context.Context,
		params *codeartifact.GetPackageVersionAssetInput,
		optFns ...func(*codeartifact.Options),
	) (*codeartifact.GetPackageVersionAssetOutput, error)

	PublishPackageVersion(
		ctx context.Context,
		params *codeartifact.PublishPackageVersionInput,
		optFns ...func(*codeartifact.Options),
	) (*codeartifact.PublishPackageVersionOutput, error)
}

type CodeArtifactRepoConfig struct {
	RepoTypeConfig
	Domain      *string `json:"domain"`
	Namespace   *string `json:"namespace"`
	Repository  *string `json:"repository"`
	DomainOwner *string `json:"domain_owner"`
	Publish     bool    `json:"publish"`

	client CodeArtifactClient
}

func (r *CodeArtifactRepoConfig) Get(ctx context.Context, module, attifact string) (io.ReadCloser, error) {
	if attifact == "@latest" {
		return nil, nil
	}

	pkg := codeArtPackageEscape(module)
	namespace := codeArtNamespaceDefault(r.Namespace)

	client, err := r.getClient(ctx)
	if err != nil {
		return nil, err
	}

	if attifact == "@v/list" {
		count := 0
		next := (*string)(nil)
		buf := bytes.NewBuffer(nil)

		input := &codeartifact.ListPackageVersionsInput{
			Package:     aws.String(pkg),
			Domain:      r.Domain,
			Namespace:   namespace,
			Repository:  r.Repository,
			DomainOwner: r.DomainOwner,
			Format:      codeartifactTypes.PackageFormatGeneric,
			Status:      codeartifactTypes.PackageVersionStatusPublished,
			MaxResults:  aws.Int32(50),
		}

		// Get all pages of version data
		for first := true; first || next != nil; first = false {
			input.NextToken = next

			output, err := client.ListPackageVersions(ctx, input)
			if err != nil {
				var rnfErr *codeartifactTypes.ResourceNotFoundException
				if errors.As(err, &rnfErr) {
					return nil, nil
				}
				return nil, fmt.Errorf("Error listing CodeArtifact versions: %v: %w", codeArtListVersionsString(input), err)
			}

			count += len(output.Versions)
			next = output.NextToken
			for _, version := range output.Versions {
				fmt.Fprintf(buf, "%v\n", aws.ToString(version.Version))
			}
		}
		logf("Got CodeArtifact versions: %v Count:%d", codeArtListVersionsString(input), count)

		return io.NopCloser(buf), nil
	}

	asset := attifact[3:]
	assetExt := path.Ext(asset)

	if !slices.Contains([]string{".info", ".mod", ".zip"}, assetExt) {
		return nil, fmt.Errorf("Asset extension not supported: %v/%v", module, attifact)
	}

	version := asset[:len(asset)-len(assetExt)]

	input := &codeartifact.GetPackageVersionAssetInput{
		Asset:          aws.String(asset),
		Package:        aws.String(pkg),
		PackageVersion: aws.String(version),
		Domain:         r.Domain,
		Namespace:      namespace,
		Repository:     r.Repository,
		DomainOwner:    r.DomainOwner,
		Format:         codeartifactTypes.PackageFormatGeneric,
	}

	output, err := client.GetPackageVersionAsset(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("Error getting CodeArtifact asset: %v: %w", codeArtGetAssetString(input), err)
	}
	logf("Got CodeArtifact asset: %v", codeArtGetAssetString(input))

	return output.Asset, nil
}

func (r *CodeArtifactRepoConfig) Put(ctx context.Context, modPath, version string, goModData, infoData, zipData []byte) error {

	client, err := r.getClient(ctx)
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	pkg := codeArtPackageEscape(modPath)
	namespace := codeArtNamespaceDefault(r.Namespace)

	// Publish info file
	input := &codeartifact.PublishPackageVersionInput{
		AssetName:      aws.String(version + ".info"),
		AssetSHA256:    aws.String(codeArtAssetSHA256(infoData)),
		AssetContent:   bytes.NewReader(infoData),
		Package:        aws.String(pkg),
		PackageVersion: aws.String(version),
		Domain:         r.Domain,
		Namespace:      namespace,
		Repository:     r.Repository,
		DomainOwner:    r.DomainOwner,
		Format:         codeartifactTypes.PackageFormatGeneric,
		Unfinished:     aws.Bool(true),
	}

	_, err = client.PublishPackageVersion(ctx, input)
	if err != nil {
		return fmt.Errorf("Error publishing CodeArtifact asset: %v: %w", codeArtPublishAssetString(input), err)
	}
	logf("Published CodeArtifact asset: %v", codeArtPublishAssetString(input))

	// Publish mod file
	input = &codeartifact.PublishPackageVersionInput{
		AssetName:      aws.String(version + ".mod"),
		AssetSHA256:    aws.String(codeArtAssetSHA256(goModData)),
		AssetContent:   bytes.NewReader(goModData),
		Package:        aws.String(pkg),
		PackageVersion: aws.String(version),
		Domain:         r.Domain,
		Namespace:      namespace,
		Repository:     r.Repository,
		DomainOwner:    r.DomainOwner,
		Format:         codeartifactTypes.PackageFormatGeneric,
		Unfinished:     aws.Bool(true),
	}

	_, err = client.PublishPackageVersion(ctx, input)
	if err != nil {
		return fmt.Errorf("Error publishing CodeArtifact asset: %v: %w", codeArtPublishAssetString(input), err)
	}
	logf("Published CodeArtifact asset: %v", codeArtPublishAssetString(input))

	// Publish zip file
	input = &codeartifact.PublishPackageVersionInput{
		AssetName:      aws.String(version + ".zip"),
		AssetSHA256:    aws.String(codeArtAssetSHA256(zipData)),
		AssetContent:   bytes.NewReader(zipData),
		Package:        aws.String(pkg),
		PackageVersion: aws.String(version),
		Domain:         r.Domain,
		Namespace:      namespace,
		Repository:     r.Repository,
		DomainOwner:    r.DomainOwner,
		Format:         codeartifactTypes.PackageFormatGeneric,
		Unfinished:     aws.Bool(false),
	}

	_, err = client.PublishPackageVersion(ctx, input)
	if err != nil {
		return fmt.Errorf("Error publishing CodeArtifact asset: %v: %w", codeArtPublishAssetString(input), err)
	}
	logf("Published CodeArtifact asset: %v", codeArtPublishAssetString(input))

	return nil
}

func (r *CodeArtifactRepoConfig) getClient(ctx context.Context) (CodeArtifactClient, error) {
	if r.client == nil {
		config, err := awsconfig.LoadDefaultConfig(ctx)
		if err != nil {
			return nil, fmt.Errorf("Error loading AWS config: %w", err)
		}
		r.client = codeartifact.NewFromConfig(config)
	}
	return r.client, nil
}

func codeArtPackageEscape(pkg string) string {
	// Code Artifact package names must
	// match the following regular expression:
	//   ([a-zA-Z0-9])+([-_+.]?[a-zA-Z0-9])*
	// Go Module paths should be restricted to [-._~/]
	//   https://go.dev/ref/mod#go-mod-file-ident
	// Escape unsupported characters using "plus" encoding
	pkg = strings.ReplaceAll(pkg, "+", "+2B")
	pkg = strings.ReplaceAll(pkg, "/", "+2F")
	pkg = strings.ReplaceAll(pkg, "~", "+7E")
	return pkg
}

func codeArtNamespaceDefault(namespace *string) *string {
	if namespace == nil {
		return aws.String("goxm")
	}
	return namespace
}

func codeArtAssetSHA256(data []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

func codeArtListVersionsString(input *codeartifact.ListPackageVersionsInput) string {
	return fmt.Sprintf(
		"Domain:%v(%v) Repo:%v NS:%v Pkg:%v",
		aws.ToString(input.Domain), aws.ToString(input.DomainOwner),
		aws.ToString(input.Repository), aws.ToString(input.Namespace),
		aws.ToString(input.Package),
	)
}

func codeArtGetAssetString(input *codeartifact.GetPackageVersionAssetInput) string {
	return fmt.Sprintf(
		"Domain:%v(%v) Repo:%v NS:%v Pkg:%v Version:%v: Asset:%v",
		aws.ToString(input.Domain), aws.ToString(input.DomainOwner),
		aws.ToString(input.Repository), aws.ToString(input.Namespace),
		aws.ToString(input.Package), aws.ToString(input.PackageVersion),
		aws.ToString(input.Asset),
	)
}

func codeArtPublishAssetString(input *codeartifact.PublishPackageVersionInput) string {
	return fmt.Sprintf(
		"Domain:%v(%v) Repo:%v NS:%v Pkg:%v Version:%v Asset:%v",
		aws.ToString(input.Domain), aws.ToString(input.DomainOwner),
		aws.ToString(input.Repository), aws.ToString(input.Namespace),
		aws.ToString(input.Package), aws.ToString(input.PackageVersion),
		aws.ToString(input.AssetName),
	)
}
