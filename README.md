GOXM: GO proXy Manager
=====================

The `go` command loads dependencies from the public proxy server (proxy.golang.org) or directly from the source version control system (VCS).

The `goxm` tool is a wrapper around the standard `go` command that can load (and publish) dependencies from alternate repositories or services like AWS CodeArtifact. All arguments are passed to the `go` command, except `publish` which is handled by `goxm`.

## Installation

Install the `goxm` command using the following command:

```bash
go install github.com/go-goxm/goxm
```

## Configuration

An exmaple `.goxm.json` is below:

```json
{
    "repos": {
        "github.com/example/*": {
            "type": "CodeArtfact",
            "repository": "example_repo",
            "domain": "example_domain",
            "domain_owner": "111111111111"
        }
    }
}
```

## Usage

### Publish module to an artifact repository:

```sh
git checkout $version
goxm publish $version
```
where `$version` in the Git tag to publish

NOTE: There is a known limitation requiring the version being published to be currently checked out.

### Download module from an artifact repository:

```sh
goxm mod download
```

or

```sh
goxm build ./...
```