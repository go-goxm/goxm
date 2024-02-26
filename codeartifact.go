package main

import (
	"context"
	"fmt"
	"io"
	"path"
	"slices"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/codeartifact"
	codeartifactTypes "github.com/aws/aws-sdk-go-v2/service/codeartifact/types"
)

type CodeArtifactRepoConfig struct {
	RepoTypeConfig
	Domain      *string `json:"domain"`
	Namespace   *string `json:"namespace"`
	Repository  *string `json:"repository"`
	DomainOwner *string `json:"domain_owner"`
	Publish     bool    `json:"publish"`

	client *codeartifact.Client
}

func (r *CodeArtifactRepoConfig) Get(ctx context.Context, module, attifact string) (io.ReadCloser, error) {
	if attifact == "@v/list" || attifact == "@latest" {
		return nil, fmt.Errorf("Not implemented: %v/%v", module, attifact)
	}

	asset := attifact[3:]
	assetExt := path.Ext(asset)

	if !slices.Contains([]string{".info", ".mod", ".zip"}, assetExt) {
		return nil, fmt.Errorf("Asset extension not supported: %v/%v", module, attifact)
	}

	// Replaces slashes (/) with underscores (_) since
	// they are not permitted in CodeArtifact package names
	pkg := strings.ReplaceAll(module, "_", "__")
	pkg = strings.ReplaceAll(pkg, "/", "_")
	version := asset[:len(asset)-len(assetExt)]

	if r.client == nil {
		config, err := awsconfig.LoadDefaultConfig(ctx)
		if err != nil {
			return nil, fmt.Errorf("Error loading AWS config: %w", err)
		}
		r.client = codeartifact.NewFromConfig(config)
	}

	fmt.Printf(
		"Get CodeArtifact: Domain:%v, Repo:%v, Pkg:%v, Asset:%v, Version:%v\n",
		aws.ToString(r.Domain), aws.ToString(r.Repository), pkg, asset, version,
	)

	namespace := r.Namespace
	if namespace == nil {
		namespace = aws.String("go")
	}

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

	output, err := r.client.GetPackageVersionAsset(ctx, input)
	if err != nil {
		return nil, fmt.Errorf(
			"Error getting CodeArtifact asset: Domain:%v(%v) Repo:%v Ns:%v, Pkg:%v Asset:%v Version:%v: %w",
			aws.ToString(r.Domain), aws.ToString(r.DomainOwner),
			aws.ToString(r.Repository), aws.ToString(namespace),
			pkg, asset, version, err,
		)
	}

	return output.Asset, nil
}
