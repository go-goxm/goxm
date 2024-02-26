GOXM: GO proXy Manager
=====================

The `go` command loads dependencies from the public proxy server (proxy.golang.org) or directly from the source version control system (VCS).

The `goxm` tool is a wrapper around the standard `go` command that can load (and publish) dependencies from alternate repositories or services like AWS CodeArtifact.

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
        "github.com/goxm/*": {
            "type": "CodeArtfact",
            "repository": "example_repo",
            "domain": "example_domain",
            "domain_owner": "111111111111"
        }
    }
}
```
