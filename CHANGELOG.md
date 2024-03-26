All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.4.2] - 2024-03-26

### Changed
- Rely with 404 status code sumdb requests

## [0.4.1] - 2024-03-26

### Changed
- Rely with 404 status code for list requests

## [0.4.0] - 2024-03-26

### Added
- Support listing version from AWS CodeArtifact

## [0.3.0] - 2024-03-25

### Fixed
- Fixed issue with files not included in zip file when module is in the root of the repository

### Added
- Set the `GONOSUMDB` for all module patterns specified in the configuration file

## [0.2.0] - 2024-03-22

### Changed
- Default CodeArtifact namespace changed to `goxm`
- Use UTC zone for time in info file

### Added
- Add tests for AWS CodeArtifact support

## [0.1.0] - 2023-03-05

### Added
- Initial release with basic support for AWS CodeArtifact


[unreleased]: https://github.com/go-goxm/goxm/compare/v0.4.1...HEAD
[0.4.1]: https://github.com/go-goxm/goxm/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/go-goxm/goxm/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/go-goxm/goxm/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/go-goxm/goxm/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/go-goxm/goxm/releases/tag/v0.1.0
