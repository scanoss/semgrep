# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
### Added
- Upcoming changes...

## [0.2.0] - 2025-09-29
### Added
- Added gRPC `GetComponentsIssues` and REST endpoint POST `/v2/semgrep/issues/components` for Semgrep security analysis
- Added gRPC `GetComponentIssues` and REST endpoint GET `/v2/semgrep/issues/component` for single component Semgrep analysis
- Added new response message types `ComponentsIssueResponse` and `ComponentIssueResponse` for enhanced component handling
- Enhance documentation
- Added unit tests
- Added REST server
### Changed
- Replaced local purl module by `github.com/scanoss/go-purl-helper`
- Replaced local logger by `github.com/scanoss/zap-logging-helper`
- Updated dependencies to latest versions
### Refactored
- Refactor on semgrep service
### Fixed 
- Fixed incorrect artifact names in release workflow

## [0.0.1] - ?
### Added
- ?

[0.2.0]: https://github.com/scanoss/semgrep/compare/v0.1.0...v0.2.0
[0.0.1]: https://github.com/scanoss/semgrep/compare/v0.0.0...v0.0.1
